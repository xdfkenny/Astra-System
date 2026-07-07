import { Suspense, lazy } from "react";
import type { PaymentAuthorizationResult } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";

const PaymentRemote = lazy(() => import("astra_payment/PaymentApp"));

/**
 * Payment Auth — the ONLY screen where the external terminal/PIN pad auth
 * factor is triggered. The remote owns Verifone SDK bridging; the shell reacts
 * to terminal outcomes to route to Receipt or back to Cart on decline.
 */
export function PaymentAuthScreen(): React.JSX.Element {
  const { send } = useKioskMachine();

  const handleResult = (result: PaymentAuthorizationResult): void => {
    if (result.status === "authorized" || result.status === "captured" || result.status === "queued_offline") {
      send({ type: "PAYMENT_AUTHORIZED", result });
    } else {
      send({ type: "PAYMENT_DECLINED", result });
    }
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-surface">
      <Suspense fallback={<PaymentSkeleton />}>
        <PaymentRemote
          onResult={handleResult}
          onCancel={() => {
            send({ type: "CANCEL_PAYMENT" });
          }}
        />
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
