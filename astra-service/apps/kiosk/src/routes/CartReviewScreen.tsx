import { Suspense, lazy } from "react";
import { useKioskMachine } from "../machines/KioskMachineProvider";

const CartRemote = lazy(() => import("astra_cart/CartApp"));

/**
 * Cart Review — federated remote handles line items, quantity edits, and the
 * transition into Payment Auth once the customer confirms.
 */
export function CartReviewScreen(): React.JSX.Element {
  const { send } = useKioskMachine();

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-surface">
      <Suspense fallback={<CartSkeleton />}>
        <CartRemote
          onBackToMenu={() => {
            send({ type: "BACK_TO_MENU" });
          }}
          onProceedToPayment={() => {
            send({ type: "PROCEED_TO_PAYMENT" });
          }}
        />
      </Suspense>
    </div>
  );
}

function CartSkeleton(): React.JSX.Element {
  return (
    <div className="flex flex-1 flex-col gap-3 overflow-hidden p-4" aria-busy="true">
      {Array.from({ length: 4 }, (_, i) => (
        <div key={i} className="hairline h-20 animate-pulse rounded-md bg-surface-sunken" />
      ))}
    </div>
  );
}
