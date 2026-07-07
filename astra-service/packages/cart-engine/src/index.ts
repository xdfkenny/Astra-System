/**
 * `@astra/cart-engine` — shared, pure cart pricing logic used by both the
 * Cart and Payment micro-frontends. Extracted as a standalone package (not
 * left inside kiosk-cart's src/) specifically so kiosk-payment never
 * reaches across a sibling app's source tree — each federated remote must
 * be independently buildable/deployable from its own dependency graph.
 */
export * from "./computeTotals";
export * from "./useCartTotals";
export * from "./produceRecognition";
