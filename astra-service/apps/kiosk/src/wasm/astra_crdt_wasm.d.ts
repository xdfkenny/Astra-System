/**
 * Type declarations for the Rust/WASM CRDT module.
 *
 * The actual `.js` glue and `.wasm` binary are produced by `wasm-pack` from
 * `astra-service/sync-daemon/crates/astra-crdt-wasm`. These declarations let
 * the kiosk typecheck and build before the WASM artifact is generated.
 */
export default function init(): Promise<void>;
export function merge_cart_ops(localState: Uint8Array, remoteOps: Uint8Array): Uint8Array;
export function hash_event_chain(previousHash: string, payload: Uint8Array): string;

