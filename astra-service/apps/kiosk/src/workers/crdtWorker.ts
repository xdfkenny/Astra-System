/// <reference lib="webworker" />

/**
 * CRDT merge worker.
 *
 * Heavy CRDT merge operations run in a dedicated Web Worker so the main thread
 * stays free for 60fps touch input. The WASM bridge is loaded dynamically on
 * first use rather than at kiosk boot.
 */

interface CrdtWasmModule {
  merge_cart_ops(localState: Uint8Array, remoteOps: Uint8Array): Uint8Array;
  hash_event_chain(previousHash: string, payload: Uint8Array): string;
}

let wasmModule: CrdtWasmModule | null = null;

async function ensureWasmLoaded(): Promise<CrdtWasmModule> {
  if (wasmModule) return wasmModule;
  const mod = (await import("../wasm/astra_crdt_wasm.js")) as unknown as {
    default: () => Promise<void>;
  } & CrdtWasmModule;
  await mod.default();
  wasmModule = mod;
  return mod;
}

export type CrdtWorkerRequest =
  | { kind: "merge_cart"; requestId: string; localState: Uint8Array; remoteOps: Uint8Array }
  | { kind: "hash_event"; requestId: string; previousHash: string; payload: Uint8Array };

export type CrdtWorkerResponse =
  | { kind: "merge_cart_result"; requestId: string; mergedState: Uint8Array }
  | { kind: "hash_event_result"; requestId: string; hash: string }
  | { kind: "error"; requestId: string; message: string };

self.addEventListener("message", (event: MessageEvent<CrdtWorkerRequest>) => {
  void handleMessage(event.data);
});

async function handleMessage(msg: CrdtWorkerRequest): Promise<void> {
  try {
    const wasm = await ensureWasmLoaded();
    if (msg.kind === "merge_cart") {
      const mergedState = wasm.merge_cart_ops(msg.localState, msg.remoteOps);
      const response: CrdtWorkerResponse = {
        kind: "merge_cart_result",
        requestId: msg.requestId,
        mergedState,
      };
      postMessage(response, { transfer: [mergedState.buffer] });
    } else {
      const hash = wasm.hash_event_chain(msg.previousHash, msg.payload);
      const response: CrdtWorkerResponse = {
        kind: "hash_event_result",
        requestId: msg.requestId,
        hash,
      };
      postMessage(response);
    }
  } catch (err: unknown) {
    const response: CrdtWorkerResponse = {
      kind: "error",
      requestId: msg.requestId,
      message: err instanceof Error ? err.message : "Unknown CRDT worker error",
    };
    postMessage(response);
  }
}

