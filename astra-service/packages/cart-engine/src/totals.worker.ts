/// <reference lib="webworker" />
import { computeCartTotals } from "./computeTotals";
import type { CartTotals, ReadonlyCartLineItem } from "@astra/shared-types";

/**
 * Offloads total computation to a Web Worker once the cart crosses a
 * complexity threshold (many lines with nested modifier trees — e.g. a
 * catering order with 40+ customized items). Below the threshold the main
 * thread computes synchronously (see useCartTotals.ts) since worker
 * postMessage overhead would dominate for small carts.
 */
export interface TotalsWorkerRequest {
  readonly requestId: string;
  readonly lines: ReadonlyArray<ReadonlyCartLineItem>;
  readonly hasLoyaltyAccount: boolean;
}

export interface TotalsWorkerResponse {
  readonly requestId: string;
  readonly totals: CartTotals;
}

self.addEventListener("message", (event: MessageEvent<TotalsWorkerRequest>) => {
  const { requestId, lines, hasLoyaltyAccount } = event.data;
  const totals = computeCartTotals(lines, { hasLoyaltyAccount });
  const response: TotalsWorkerResponse = { requestId, totals };
  postMessage(response);
});
