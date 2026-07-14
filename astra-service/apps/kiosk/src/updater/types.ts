/**
 * Types for the Astra kiosk over-the-air (OTA) update system.
 */

/** A single downloadable artifact in a manifest. */
export interface Artifact {
  url: string;
  checksum: string;
  platforms: string[];
}

/** Rollout policy controlling how updates are distributed. */
export interface Rollout {
  strategy: string;
  maxConcurrent: number;
  healthCheckSeconds: number;
}

/** Signed update manifest returned by the update server. */
export interface Manifest {
  version: string;
  channel: string;
  releasedAt: string;
  artifacts: Record<string, Artifact>;
  rollout: Rollout;
  signature: string;
}

/** Phases the updater can be in. */
export type UpdateState =
  | "idle"
  | "checking"
  | "downloading"
  | "pending"
  | "applying"
  | "healthy"
  | "rolled_back";

/** Configuration for the kiosk updater. */
export interface UpdaterConfig {
  /** Base URL of the update server manifest endpoint. */
  manifestUrl: string;
  /** Ed25519 public key as raw 32-byte Uint8Array used to verify manifest signatures. */
  publicKey: Uint8Array;
  /** How often to poll for updates (ms). */
  pollIntervalMs: number;
  /** Target platform, e.g. linux/arm64. */
  platform: string;
  /** Artifact name to download, e.g. kiosk-shell. */
  artifactName: string;
  /**
   * Called when an update should be applied. The implementer is responsible
   * for staging the new bundle and reloading/restarting the kiosk.
   */
  onApply: (manifest: Manifest, artifactBlob: Blob) => Promise<void>;
  /**
   * Called when the post-update health check fails and the kiosk must roll
   * back to the previous known-good version.
   */
  onRollback: (previousVersion: string) => Promise<void>;
  /**
   * Optional health check used after an update is applied. Defaults to true.
   */
  healthCheck?: () => Promise<boolean>;
}

/** Summary of a completed or failed update check. */
export interface UpdateResult {
  state: UpdateState;
  currentVersion: string;
  pendingVersion: string | undefined;
  error: Error | undefined;
}

