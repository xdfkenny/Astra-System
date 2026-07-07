import type { MenuItem } from "@astra/shared-types";

/**
 * TypeScript integration layer for computer-vision produce recognition.
 *
 * The actual ONNX inference happens in a separate Rust/WASM module that this
 * layer loads and calls. The cart-engine package owns the contract so both the
 * kiosk host and the cart micro-frontend share the same recognition semantics
 * and confidence thresholds.
 */

export interface ProduceMatch {
  readonly itemId: string;
  readonly plu: string;
  readonly name: string;
  readonly confidence: number;
}

export interface ProduceRecognitionResult {
  readonly matches: readonly ProduceMatch[];
  readonly bestMatch: ProduceMatch | null;
}

export interface ProduceRecognizer {
  readonly recognize: (image: ImageBitmap) => Promise<ProduceRecognitionResult>;
  readonly isReady: () => boolean;
}

const CONFIDENCE_THRESHOLD = 0.75;

/**
 * Synchronous PLU lookup used by the manual-entry fallback. In production this
 * would be hydrated from the cached catalog; the reference implementation keeps
 * a small built-in table so the fallback path is always runnable.
 */
export const PLU_CATALOG: Readonly<Record<string, { itemId: string; name: string }>> = {
  "4011": { itemId: "prod-banana", name: "Bananas" },
  "4013": { itemId: "prod-apple-gala", name: "Gala Apples" },
  "4062": { itemId: "prod-avocado", name: "Avocados" },
  "4087": { itemId: "prod-tomato-roma", name: "Roma Tomatoes" },
  "4405": { itemId: "prod-lemon", name: "Lemons" },
  "4899": { itemId: "prod-cucumber", name: "Cucumbers" },
};

export function lookupByPlu(plu: string): ProduceMatch | null {
  const normalized = plu.trim();
  const entry = PLU_CATALOG[normalized];
  if (!entry) return null;
  return {
    itemId: entry.itemId,
    plu: normalized,
    name: entry.name,
    confidence: 1,
  };
}

/**
 * ONNX-backed recognizer. Loads the WASM module lazily and delegates inference.
 * If the module is unavailable or returns a low-confidence result, the caller
 * falls back to manual PLU entry.
 */
export function createOnnxProduceRecognizer(
  moduleUrl: string | undefined = import.meta.env["VITE_PRODUCE_WASM_URL"] as string | undefined,
): ProduceRecognizer {
  let module: { infer: (image: ImageBitmap) => Promise<ProduceRecognitionResult> } | null = null;
  let loading: Promise<void> | null = null;

  const load = async (): Promise<void> => {
    if (module) return;
    if (!moduleUrl) {
      throw new Error("VITE_PRODUCE_WASM_URL is not configured");
    }
    const wasm = await import(/* @vite-ignore */ moduleUrl);
    if (typeof wasm.infer !== "function") {
      throw new Error("Produce WASM module must export an infer function");
    }
    module = wasm as { infer: (image: ImageBitmap) => Promise<ProduceRecognitionResult> };
  };

  return {
    isReady: () => module !== null,
    recognize: async (image) => {
      loading ??= load();
      await loading;
      if (!module) {
        return { matches: [], bestMatch: null };
      }
      const result = await module.infer(image);
      const best = result.matches[0] ?? null;
      return {
        matches: result.matches,
        bestMatch: best && best.confidence >= CONFIDENCE_THRESHOLD ? best : null,
      };
    },
  };
}

/**
 * Deterministic recognizer used in tests and local dev when the WASM module is
 * not built. It never claims a high-confidence match, forcing the fallback path
 * and ensuring the manual PLU flow is exercised.
 */
export function createTestProduceRecognizer(): ProduceRecognizer {
  return {
    isReady: () => true,
    recognize: async () => ({ matches: [], bestMatch: null }),
  };
}

export function matchToMenuItem(match: ProduceMatch, catalog: readonly MenuItem[]): MenuItem | null {
  return catalog.find((item) => item.plu === match.plu || item.itemId === match.itemId) ?? null;
}
