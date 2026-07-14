import {
  downloadAndVerifyArtifact,
  verifyBlobChecksum,
  verifyManifestSignature,
} from "./crypto";
import { fetchManifest } from "./manifest";
import type {
  Artifact,
  Manifest,
  UpdateResult,
  UpdateState,
  UpdaterConfig,
} from "./types";

/**
 * Compare two semantic-ish version strings. Returns a positive number when
 * next > current, zero when equal, and a negative number when next < current.
 *
 * Supports simple dotted versions such as `v1.2.3`. Non-numeric segments are
 * compared lexicographically.
 */
export function compareVersions(current: string, next: string): number {
  const normalize = (v: string) =>
    v
      .replace(/^v/i, "")
      .split(".")
      .map((part) => {
        const n = Number.parseInt(part, 10);
        return Number.isNaN(n) ? part : n;
      });

  const a = normalize(current);
  const b = normalize(next);
  const len = Math.max(a.length, b.length);

  for (let i = 0; i < len; i++) {
    const left = a[i] ?? 0;
    const right = b[i] ?? 0;
    if (left < right) return -1;
    if (left > right) return 1;
  }

  return 0;
}

/**
 * Updater polls the update server, downloads signed artifacts, and applies
 * them when the kiosk is idle. It supports rollback if a post-update health
 * check fails within the configured window.
 */
export class Updater {
  private config: UpdaterConfig;
  private currentVersion: string;
  private state: UpdateState = "idle";
  private pendingManifest: Manifest | undefined;
  private pendingBlob: Blob | undefined;
  private pollTimer: ReturnType<typeof setInterval> | undefined;
  private rollbackTimer: ReturnType<typeof setTimeout> | undefined;

  constructor(config: UpdaterConfig, currentVersion = "v0.0.0") {
    this.config = config;
    this.currentVersion = currentVersion;
  }

  getState(): UpdateState {
    return this.state;
  }

  getCurrentVersion(): string {
    return this.currentVersion;
  }

  getPendingVersion(): string | undefined {
    return this.pendingManifest?.version;
  }

  /**
   * Begin polling for updates at the configured interval.
   */
  start(): void {
    if (this.pollTimer) {
      return;
    }
    void this.check();
    this.pollTimer = setInterval(() => void this.check(), this.config.pollIntervalMs);
  }

  /**
   * Stop polling for updates.
   */
  stop(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = undefined;
    }
    if (this.rollbackTimer) {
      clearTimeout(this.rollbackTimer);
      this.rollbackTimer = undefined;
    }
  }

  /**
   * Manually trigger an update check. Returns true if a new update is pending.
   */
  async check(): Promise<UpdateResult> {
    if (this.state === "checking" || this.state === "downloading") {
      return this.result();
    }

    this.setState("checking");

    try {
      const manifest = await fetchManifest(this.config.manifestUrl);

      if (compareVersions(this.currentVersion, manifest.version) >= 0) {
        this.setState("idle");
        return this.result();
      }

      const artifact = this.selectArtifact(manifest);
      if (!artifact) {
        throw new Error(
          `no artifact ${this.config.artifactName} for platform ${this.config.platform}`,
        );
      }

      const signatureValid = await verifyManifestSignature(
        manifest,
        this.config.publicKey,
      );
      if (!signatureValid) {
        throw new Error("manifest signature verification failed");
      }

      this.setState("downloading");
      const blob = await downloadAndVerifyArtifact(artifact);

      this.pendingManifest = manifest;
      this.pendingBlob = blob;
      this.setState("pending");

      return this.result();
    } catch (err) {
      this.setState("idle");
      return this.result(err instanceof Error ? err : new Error(String(err)));
    }
  }

  /**
   * Apply a pending update when the kiosk is idle. After applying, a health
   * check is scheduled; if it fails the kiosk rolls back.
   */
  async applyWhenIdle(): Promise<UpdateResult> {
    if (this.state !== "pending" || !this.pendingManifest || !this.pendingBlob) {
      return this.result(new Error("no pending update to apply"));
    }

    const previousVersion = this.currentVersion;
    const manifest = this.pendingManifest;
    const blob = this.pendingBlob;

    this.setState("applying");

    try {
      await this.config.onApply(manifest, blob);
      this.currentVersion = manifest.version;
      const healthCheckSeconds = manifest.rollout.healthCheckSeconds;
      this.pendingManifest = undefined;
      this.pendingBlob = undefined;
      this.setState("healthy");
      this.scheduleRollbackCheck(previousVersion, healthCheckSeconds);
      return this.result();
    } catch (err) {
      this.setState("rolled_back");
      await this.config.onRollback(previousVersion);
      return this.result(err instanceof Error ? err : new Error(String(err)));
    }
  }

  /**
   * Re-verify a downloaded artifact blob against the manifest checksum. Useful
   * right before applying an update staged earlier.
   */
  async verifyPendingArtifact(): Promise<boolean> {
    if (!this.pendingManifest || !this.pendingBlob) {
      return false;
    }
    const artifact = this.selectArtifact(this.pendingManifest);
    if (!artifact) {
      return false;
    }
    return verifyBlobChecksum(this.pendingBlob, artifact.checksum);
  }

  private selectArtifact(manifest: Manifest): Artifact | undefined {
    const artifact = manifest.artifacts[this.config.artifactName];
    if (!artifact) {
      return undefined;
    }
    const exact = artifact.platforms.includes(this.config.platform);
    const any = artifact.platforms.includes("any");
    return exact || any ? artifact : undefined;
  }

  private scheduleRollbackCheck(
    previousVersion: string,
    healthCheckSeconds: number,
  ): void {
    const delaySeconds = this.config.healthCheck
      ? Math.max(0, healthCheckSeconds)
      : 300;

    this.rollbackTimer = setTimeout(async () => {
      const healthy = this.config.healthCheck
        ? await this.config.healthCheck()
        : true;

      if (!healthy) {
        this.setState("rolled_back");
        await this.config.onRollback(previousVersion);
      }
    }, delaySeconds * 1000);
  }

  private setState(state: UpdateState): void {
    this.state = state;
  }

  private result(error?: Error): UpdateResult {
    return {
      state: this.state,
      currentVersion: this.currentVersion,
      pendingVersion: this.pendingManifest?.version,
      error,
    };
  }
}

