import type { Manifest } from "./types";

/**
 * Fetch and parse a signed update manifest.
 *
 * @throws Error when the network request fails or the response is not valid JSON.
 */
export async function fetchManifest(url: string): Promise<Manifest> {
  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(
      `manifest fetch failed: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as Manifest;
  validateManifestShape(data);
  return data;
}

function validateManifestShape(data: unknown): asserts data is Manifest {
  if (typeof data !== "object" || data === null) {
    throw new Error("manifest is not an object");
  }

  const m = data as Partial<Manifest>;
  if (typeof m.version !== "string" || m.version === "") {
    throw new Error("manifest.version is required");
  }
  if (typeof m.channel !== "string") {
    throw new Error("manifest.channel is required");
  }
  if (typeof m.signature !== "string" || m.signature === "") {
    throw new Error("manifest.signature is required");
  }
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- runtime guard against malformed JSON
  if (typeof m.artifacts !== "object" || m.artifacts === null) {
    throw new Error("manifest.artifacts is required");
  }
}
