import * as ed from "@noble/ed25519";
import { sha256 } from "@noble/hashes/sha2.js";
import type { Artifact, Manifest } from "./types";

const SUPPORTED_ALGORITHMS = new Set(["sha256"]);

/**
 * Decode a base64-encoded Ed25519 public key into a raw 32-byte Uint8Array.
 */
export function importPublicKey(base64: string): Uint8Array {
  return base64ToBytes(base64);
}

/**
 * Produce a deterministic canonical JSON representation of a manifest that
 * matches the canonicalization performed by the Go update server. Artifacts
 * and platform lists are sorted by key so source ordering does not affect the
 * signature.
 */
export function canonicalizeManifest(m: Manifest): string {
  const sortedArtifacts: Record<string, Artifact> = {};
  for (const name of Object.keys(m.artifacts).sort()) {
    const a = m.artifacts[name];
    if (a === undefined) {
      continue;
    }
    sortedArtifacts[name] = {
      url: a.url,
      checksum: a.checksum,
      platforms: [...a.platforms].sort(),
    };
  }

  const canonical = {
    version: m.version,
    channel: m.channel,
    releasedAt: m.releasedAt,
    artifacts: sortedArtifacts,
    rollout: m.rollout,
  };

  return JSON.stringify(canonical);
}

/**
 * Verify the detached Ed25519 signature on a manifest.
 */
export async function verifyManifestSignature(
  m: Manifest,
  publicKey: Uint8Array,
): Promise<boolean> {
  try {
    const canonical = canonicalizeManifest(m);
    const signature = base64ToBytes(m.signature);
    const message = new TextEncoder().encode(canonical);
    return await ed.verifyAsync(signature, message, publicKey);
  } catch {
    return false;
  }
}

/**
 * Download an artifact and verify its SHA-256 checksum.
 */
export async function downloadAndVerifyArtifact(
  artifact: Artifact,
): Promise<Blob> {
  const response = await fetch(artifact.url, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(
      `artifact download failed: ${response.status} ${response.statusText}`,
    );
  }

  const blob = await response.blob();
  const valid = await verifyBlobChecksum(blob, artifact.checksum);
  if (!valid) {
    throw new Error("artifact checksum mismatch");
  }

  return blob;
}

/**
 * Verify a Blob against a `sha256:<hex>` checksum.
 */
export async function verifyBlobChecksum(
  blob: Blob,
  expected: string,
): Promise<boolean> {
  const { algorithm, hash } = parseChecksum(expected);
  if (!SUPPORTED_ALGORITHMS.has(algorithm)) {
    throw new Error(`unsupported checksum algorithm: ${algorithm}`);
  }

  const buffer = await blob.arrayBuffer();
  const digest = sha256(new Uint8Array(buffer));
  return constantTimeEqual(digest, hash);
}

export function parseChecksum(checksum: string): {
  algorithm: string;
  hash: Uint8Array;
} {
  const parts = checksum.split(":");
  if (parts.length !== 2) {
    throw new Error(`invalid checksum format: ${checksum}`);
  }

  const [algorithmPart, hex] = parts;
  if (algorithmPart === undefined || hex === undefined) {
    throw new Error(`invalid checksum format: ${checksum}`);
  }

  const algorithm = algorithmPart.toLowerCase();
  if (hex.length % 2 !== 0) {
    throw new Error(`invalid checksum hex length: ${hex.length}`);
  }

  const hash = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    hash[i / 2] = Number.parseInt(hex.slice(i, i + 2), 16);
  }

  return { algorithm, hash };
}

function base64ToBytes(value: string): Uint8Array {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function constantTimeEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }
  let diff = 0;
  for (let i = 0; i < a.length; i++) {
    const av = a[i];
    const bv = b[i];
    if (av === undefined || bv === undefined) {
      return false;
    }
    diff |= av ^ bv;
  }
  return diff === 0;
}

/**
 * Convert a Uint8Array to a base64 string. Exposed for tests and tooling.
 */
export function bytesToBase64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes));
}
