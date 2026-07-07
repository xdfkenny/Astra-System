import { useEffect, useMemo, useRef, useState } from "react";
import type { CartTotals, ReadonlyCartLineItem } from "@astra/shared-types";
import { computeCartTotals } from "./computeTotals";
import type { TotalsWorkerRequest, TotalsWorkerResponse } from "./totals.worker";

/** Above this many (line x modifier) combinations, delegate to the Web Worker. */
const WORKER_OFFLOAD_THRESHOLD = 25;

export function useCartTotals(
  lines: ReadonlyArray<ReadonlyCartLineItem>,
  hasLoyaltyAccount: boolean,
): CartTotals {
  const complexity = lines.reduce((sum, l) => sum + 1 + l.modifiers.length, 0);
  const workerRef = useRef<Worker | null>(null);
  const [workerTotals, setWorkerTotals] = useState<CartTotals | null>(null);

  const syncTotals = useMemo(
    () => computeCartTotals(lines, { hasLoyaltyAccount }),
    [lines, hasLoyaltyAccount],
  );

  useEffect(() => {
    if (complexity <= WORKER_OFFLOAD_THRESHOLD) return;

    workerRef.current ??= new Worker(new URL("./totals.worker.ts", import.meta.url), {
      type: "module",
    });
    const worker = workerRef.current;
    const requestId = crypto.randomUUID();

    const handleMessage = (event: MessageEvent<TotalsWorkerResponse>): void => {
      if (event.data.requestId === requestId) {
        setWorkerTotals(event.data.totals);
      }
    };
    worker.addEventListener("message", handleMessage);

    const request: TotalsWorkerRequest = { requestId, lines, hasLoyaltyAccount };
    worker.postMessage(request);

    return () => worker.removeEventListener("message", handleMessage);
  }, [lines, hasLoyaltyAccount, complexity]);

  useEffect(() => {
    return () => {
      workerRef.current?.terminate();
      workerRef.current = null;
    };
  }, []);

  return complexity > WORKER_OFFLOAD_THRESHOLD && workerTotals ? workerTotals : syncTotals;
}
