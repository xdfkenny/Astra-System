import { sha256 } from "@noble/hashes/sha2.js";
import * as ed from "@noble/ed25519";
import { vi } from "vitest";
import {
  bytesToBase64,
  canonicalizeManifest,
  importPublicKey,
  verifyManifestSignature,
} from "./crypto";
import type { Manifest } from "./types";
import { compareVersions, Updater } from "./updater";

describe("compareVersions", () => {
  it("returns negative when current is older", () => {
    expect(compareVersions("v1.0.0", "v1.1.0")).toBeLessThan(0);
  });

  it("returns positive when current is newer", () => {
    expect(compareVersions("v2.0.0", "v1.9.0")).toBeGreaterThan(0);
  });

  it("returns zero for equal versions", () => {
    expect(compareVersions("v1.2.3", "v1.2.3")).toBe(0);
  });
});

describe("Updater", () => {
  let secretKey: Uint8Array;
  let publicKey: Uint8Array;

  beforeEach(async () => {
    const keys = await ed.keygenAsync();
    secretKey = keys.secretKey;
    publicKey = keys.publicKey;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("detects a newer signed manifest and stages an update", async () => {
    const artifactContent: Uint8Array = new TextEncoder().encode(
      "hello artifact",
    );
    const manifest = await signedManifestWithArtifact(
      "v1.2.3",
      secretKey,
      artifactContent,
    );

    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(manifest))
      .mockResolvedValueOnce(blobResponse(artifactContent));

    const onApply = vi.fn().mockResolvedValue(undefined);
    const onRollback = vi.fn().mockResolvedValue(undefined);

    const updater = new Updater(
      {
        manifestUrl: "https://updates.astra/manifest.json",
        publicKey,
        pollIntervalMs: 60_000,
        platform: "linux/arm64",
        artifactName: "kiosk-shell",
        onApply,
        onRollback,
      },
      "v1.0.0",
    );

    const result = await updater.check();

    expect(result.state).toBe("pending");
    expect(result.pendingVersion).toBe("v1.2.3");
    expect(updater.getPendingVersion()).toBe("v1.2.3");
  });

  it("rejects a manifest with an invalid signature", async () => {
    const manifest = await signedManifest("v1.2.3", secretKey);
    manifest.signature = "aW52YWxpZA=="; // base64("invalid")

    global.fetch = vi.fn().mockResolvedValueOnce(jsonResponse(manifest));

    const updater = new Updater(
      {
        manifestUrl: "https://updates.astra/manifest.json",
        publicKey,
        pollIntervalMs: 60_000,
        platform: "linux/arm64",
        artifactName: "kiosk-shell",
        onApply: vi.fn(),
        onRollback: vi.fn(),
      },
      "v1.0.0",
    );

    const result = await updater.check();

    expect(result.state).toBe("idle");
    expect(result.error?.message).toContain("signature");
  });

  it("does nothing when the manifest version is not newer", async () => {
    const manifest = await signedManifest("v1.0.0", secretKey);
    global.fetch = vi.fn().mockResolvedValueOnce(jsonResponse(manifest));

    const updater = new Updater(
      {
        manifestUrl: "https://updates.astra/manifest.json",
        publicKey,
        pollIntervalMs: 60_000,
        platform: "linux/arm64",
        artifactName: "kiosk-shell",
        onApply: vi.fn(),
        onRollback: vi.fn(),
      },
      "v1.0.0",
    );

    const result = await updater.check();

    expect(result.state).toBe("idle");
    expect(result.pendingVersion).toBeUndefined();
  });

  it("applies a pending update and rolls back on failed health check", async () => {
    const artifactContent: Uint8Array = new TextEncoder().encode(
      "hello artifact",
    );
    const manifest = await signedManifestWithArtifact(
      "v1.2.3",
      secretKey,
      artifactContent,
    );
    manifest.rollout.healthCheckSeconds = 0;
    manifest.signature = await signManifest(manifest, secretKey);

    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(manifest))
      .mockResolvedValueOnce(blobResponse(artifactContent));

    const onApply = vi.fn().mockResolvedValue(undefined);
    const onRollback = vi.fn().mockResolvedValue(undefined);

    const updater = new Updater(
      {
        manifestUrl: "https://updates.astra/manifest.json",
        publicKey,
        pollIntervalMs: 60_000,
        platform: "linux/arm64",
        artifactName: "kiosk-shell",
        onApply,
        onRollback,
        healthCheck: vi.fn().mockResolvedValue(false),
      },
      "v1.0.0",
    );

    const checkResult = await updater.check();
    expect(checkResult.state).toBe("pending");

    const applyResult = await updater.applyWhenIdle();

    expect(applyResult.state).toBe("healthy");
    expect(onApply).toHaveBeenCalledWith(manifest, expect.any(Blob));

    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(onRollback).toHaveBeenCalledWith("v1.0.0");
    expect(updater.getState()).toBe("rolled_back");
  });

  it("imports a base64 public key and verifies a manifest", async () => {
    const manifest = await signedManifest("v1.5.0", secretKey);
    const imported = importPublicKey(bytesToBase64(publicKey));

    const valid = await verifyManifestSignature(manifest, imported);
    expect(valid).toBe(true);
  });
});

async function signedManifest(
  version: string,
  secretKey: Uint8Array,
): Promise<Manifest> {
  const manifest: Manifest = {
    version,
    channel: "stable",
    releasedAt: new Date().toISOString(),
    artifacts: {
      "kiosk-shell": {
        url: `https://cdn.astra-service.internal/kiosk-shell/${version}/kiosk-shell.tar.gz`,
        checksum:
          "sha256:0000000000000000000000000000000000000000000000000000000000000000",
        platforms: ["linux/amd64", "linux/arm64"],
      },
      "sync-daemon": {
        url: `https://cdn.astra-service.internal/sync-daemon/${version}/sync-daemon.tar.gz`,
        checksum:
          "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        platforms: ["linux/amd64", "linux/arm64"],
      },
    },
    rollout: {
      strategy: "idle-only",
      maxConcurrent: 1,
      healthCheckSeconds: 300,
    },
    signature: "",
  };

  manifest.signature = await signManifest(manifest, secretKey);
  return manifest;
}

async function signedManifestWithArtifact(
  version: string,
  secretKey: Uint8Array,
  artifactContent: Uint8Array,
): Promise<Manifest> {
  const checksum = sha256Hex(artifactContent);
  const manifest = await signedManifest(version, secretKey);
  const kioskArtifact = manifest.artifacts["kiosk-shell"];
  if (kioskArtifact === undefined) {
    throw new Error("missing kiosk-shell artifact");
  }
  kioskArtifact.checksum = `sha256:${checksum}`;
  manifest.signature = await signManifest(manifest, secretKey);
  return manifest;
}

async function signManifest(
  manifest: Manifest,
  secretKey: Uint8Array,
): Promise<string> {
  const canonical = canonicalizeManifest(manifest);
  const message = new TextEncoder().encode(canonical);
  const signature = await ed.signAsync(message, secretKey);
  return bytesToBase64(signature);
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function blobResponse(data: Uint8Array): Response {
  return new Response(new Blob([data as BlobPart]), { status: 200 });
}

function sha256Hex(data: Uint8Array): string {
  const digest = sha256(data);
  return Array.from(digest)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

