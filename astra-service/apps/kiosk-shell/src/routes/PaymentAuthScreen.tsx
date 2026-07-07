import { Suspense, lazy } from "react";
import { useSessionStore } from "@astra/kiosk-state";
import type { PaymentAuthorizationResult } from "@astra/shared-types";

const PaymentRemote = lazy(() => import("astra_payment/PaymentApp"));

/**
 * Payment Auth — the ONLY screen where a biometric PIN pad / NFC employee
 * card / Verifone terminal interaction is triggered, per the security
 * mandate ("Auth factor ONLY triggers at payment confirmation, not at app
 * start"). The remote owns all Verifone SDK bridging; the shell only reacts
 * to the terminal outcome to route to Receipt or back to Cart on decline.
 */
export function PaymentAuthScreen(): React.JSX.Element {
  const goToStage = useSessionStore((s) => s.goToStage);

  const handleResult = (result: PaymentAuthorizationResult): void => {
    if (result.status === "authorized" || result.status === "captured") {
      goToStage("receipt");
    } else if (result.status === "declined") {
      goToStage("cart_review");
    }
    // "queued_offline" / "failed_network" are handled inside the remote itself
    // (it keeps the customer on this screen with a reassuring offline-queue message).
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-surface">
      <Suspense fallback={<PaymentSkeleton />}>
        <PaymentRemote onResult={handleResult} onCancel={() => { goToStage("cart_review"); }} />
      </Suspense>
    </div>
  );
}

function PaymentSkeleton(): React.JSX.Element {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 p-6" aria-busy="true">
      <div className="h-24 w-24 animate-pulse rounded-full bg-surface-sunken" />
      <div className="h-4 w-48 animate-pulse rounded bg-surface-sunken" />
    </div>
  );
}
