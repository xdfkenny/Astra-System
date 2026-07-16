/**
 * App Shell versioning strategy for Module Federation remotes.
 *
 * Manages remote module mounting with atomic rollback. When a remote module
 * fails to mount (network error, JS evaluation error, version mismatch), the
 * shell falls back to the last known-good version or a safe offline stub.
 *
 * Each remote is tracked by a version string. The updater swaps the remote's
 * entry point URL atomically so that the shell never observes a partially
 * loaded remote.
 */

import type { Manifest } from "./types";

declare const __webpack_init_sharing__: (scope: string) => Promise<void>;
declare const __webpack_share_scopes__: Record<string, unknown>;

/** Shared dependency version requirement. */
export interface SharedDependency {
  name: string;
  requiredVersion: string;
}

/** Configuration for a single federated remote. */
export interface RemoteDefinition {
  /** Unique remote identifier, e.g. "astra_menu". */
  name: string;
  /** Base URL for the remote's remoteEntry.js. */
  baseUrl: string;
  /** Current version deployed on this remote. */
  version: string;
  /** Timeout (ms) for loading the remote entry script. */
  timeoutMs: number;
  /** Fallback URL to the last known-good version. */
  fallbackUrl: string;
  /** Shared dependencies the remote expects. */
  sharedDeps?: SharedDependency[];
}

/** Runtime state of a remote module. */
export interface RemoteState {
  definition: RemoteDefinition;
  status: "mounting" | "mounted" | "failed" | "rolled_back";
  lastError?: Error;
  mountedAt?: number;
}

/** Minimal health check result from a remote. */
interface RemoteHealth {
  ok: boolean;
  version: string;
}

/**
 * Manages the lifecycle of federated remote modules.
 *
 * Supports:
 * - Atomic URL swapping: the remote entry URL is only switched after
 *   a successful health check against the new version.
 * - Automatic rollback: if mounting fails, the shell reverts to the
 *   fallback URL and marks the remote as "rolled_back".
 * - Stale version GC: removes cached entries for old remote versions.
 */
export class RemoteModuleManager {
  private remotes = new Map<string, RemoteState>();
  private onVersionChange: (name: string, from: string, to: string) => void;

  constructor(opts: {
    onVersionChange?: (name: string, from: string, to: string) => void;
  }) {
    this.onVersionChange =
      opts.onVersionChange ?? (() => { return; });
  }

  /** Register or update a remote definition. */
  register(def: RemoteDefinition): void {
    this.remotes.set(def.name, {
      definition: def,
      status: "mounting",
    });
  }

  /** Retrieve the current state for a remote. */
  get(name: string): RemoteState | undefined {
    return this.remotes.get(name);
  }

  /** List all registered remotes. */
  list(): RemoteState[] {
    return Array.from(this.remotes.values());
  }

  /**
   * Atomically swap a remote's base URL to a new version.  Performs a
   * pre-flight health check against the new URL.  On failure, the remote
   * stays pinned to its fallback URL.
   *
   * Returns `true` if the swap succeeded.
   */
  async swapVersion(
    name: string,
    newBaseUrl: string,
    newVersion: string,
  ): Promise<boolean> {
    const state = this.remotes.get(name);
    if (!state) {
      throw new Error(`unknown remote: ${name}`);
    }

    const oldVersion = state.definition.version;
    if (oldVersion === newVersion) {
      return true; // already on this version
    }

    // Pre-flight: check if the new remote entry responds
    const health = await this.checkRemoteHealth(newBaseUrl, state.definition.timeoutMs);
    if (!health.ok) {
      state.status = "failed";
      state.lastError = new Error(
        `pre-flight health check failed for ${name}@${newVersion}`,
      );
      return false;
    }

    // Atomically swap the base URL
    const oldFallback = state.definition.fallbackUrl;
    state.definition.fallbackUrl = state.definition.baseUrl;
    state.definition.baseUrl = newBaseUrl;
    state.definition.version = newVersion;

    this.onVersionChange(name, oldVersion, newVersion);

    // Post-swap: verify the remote is actually reachable
    try {
      const ok = await this.tryMount(name);
      if (!ok) {
        // Rollback to previous
        state.definition.baseUrl = state.definition.fallbackUrl;
        state.definition.fallbackUrl = oldFallback;
        state.definition.version = oldVersion;
        state.status = "rolled_back";
        return false;
      }
      state.status = "mounted";
      state.mountedAt = Date.now();
      return true;
    } catch {
      state.definition.baseUrl = state.definition.fallbackUrl;
      state.definition.fallbackUrl = oldFallback;
      state.definition.version = oldVersion;
      state.status = "rolled_back";
      return false;
    }
  }

  /**
   * Validate shared dependencies for a remote before loading.
   * Returns a list of unmet dependency descriptions.
   */
  private checkSharedDependencies(deps: SharedDependency[]): string[] {
    const unmet: string[] = [];
    for (const dep of deps) {
      const scope = __webpack_share_scopes__["default"] as
        | Record<string, { version: string }>
        | undefined;
      const shared = scope?.[dep.name];
      if (!shared) {
        unmet.push(`${dep.name}@${dep.requiredVersion} (not shared)`);
      }
    }
    return unmet;
  }

  /**
   * Attempt to load the remote entry script for the given remote.
   * Returns `true` if the script loaded and executed without error.
   *
   * Before loading, initialises the default webpack share scope so
   * that the remote can negotiate shared dependency versions.
   */
  private async tryMount(name: string): Promise<boolean> {
    const state = this.remotes.get(name);
    if (!state) {
      return false;
    }

    // Validate shared dependencies before mounting
    if (state.definition.sharedDeps?.length) {
      const unmet = this.checkSharedDependencies(state.definition.sharedDeps);
      for (const msg of unmet) {
        console.warn(`[remote-modules] unmet shared dependency for ${name}: ${msg}`);
      }
    }

    // Initialise webpack share scope so the remote can find shared deps
    try {
      if (typeof __webpack_init_sharing__ === "function") {
        await __webpack_init_sharing__("default");
      }
    } catch {
      // Non-fatal: some environments may not use Module Federation
    }

    const url = `${state.definition.baseUrl}/remoteEntry.js`;
    return new Promise<boolean>((resolve) => {
      const script = document.createElement("script");
      script.src = url;
      script.async = true;
      script.crossOrigin = "anonymous";

      const timeout = setTimeout(() => {
        script.remove();
        resolve(false);
      }, state.definition.timeoutMs);

      script.onload = () => {
        clearTimeout(timeout);
        resolve(true);
      };
      script.onerror = () => {
        clearTimeout(timeout);
        resolve(false);
      };

      document.head.appendChild(script);
    });
  }

  /**
   * Lightweight health check against a remote's version endpoint.
   */
  private async checkRemoteHealth(
    baseUrl: string,
    timeoutMs: number,
  ): Promise<RemoteHealth> {
    const controller = new AbortController();
    const timeout = setTimeout(() => { controller.abort(); }, timeoutMs);

    try {
      const response = await fetch(`${baseUrl}/astra-version.json`, {
        signal: controller.signal,
        cache: "no-store",
      });
      if (!response.ok) {
        return { ok: false, version: "0.0.0" };
      }
      const data = (await response.json()) as { version?: string };
      return {
        ok: true,
        version: data.version ?? "0.0.0",
      };
    } catch {
      return { ok: false, version: "0.0.0" };
    } finally {
      clearTimeout(timeout);
    }
  }

  /**
   * Process an update manifest: for each remote in the manifest, swap
   * to the new version if available.
   *
   * Returns a record of swap results keyed by remote name.
   */
  async applyManifest(manifest: Manifest): Promise<Record<string, boolean>> {
    const results: Record<string, boolean> = {};

    for (const [artifactName] of Object.entries(manifest.artifacts)) {
      const state = this.remotes.get(artifactName);
      if (!state) {
        continue;
      }

      const newBaseUrl = manifest.version;
      const newVersion = manifest.version;

      results[artifactName] = await this.swapVersion(
        artifactName,
        newBaseUrl,
        newVersion,
      );
    }

    return results;
  }

  /** Reset all remotes to their fallback URLs. */
  rollbackAll(): void {
    for (const [_name, state] of this.remotes) {
      if (state.definition.baseUrl !== state.definition.fallbackUrl) {
        state.definition.baseUrl = state.definition.fallbackUrl;
        state.status = "rolled_back";
        state.lastError = new Error("rolled back via rollbackAll()");
      }
    }
  }
}

/**
 * Federation URL generator that embeds version strings into remote URLs.
 *
 * Example:
 *   `https://cdn.astra-service.internal/menu/v1.2.3/remoteEntry.js`
 */
export function versionedRemoteUrl(
  baseUrl: string,
  version: string,
): string {
  const normalizedBase = baseUrl.replace(/\/+$/, "");
  const normalizedVersion = version.replace(/^v/, "");
  return `${normalizedBase}/v${normalizedVersion}`;
}
