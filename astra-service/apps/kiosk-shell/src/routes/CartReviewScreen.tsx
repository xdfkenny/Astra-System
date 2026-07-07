import { Suspense, lazy } from "react";
import { useSessionStore } from "@astra/kiosk-state";

const CartRemote = lazy(() => import("astra_cart/CartApp"));

/**
 * Cart Review — federated remote handles line items, quantity edits, the
 * "Why this price?" transparency panel (deep-improvement #8), and the
 * transition into Payment Auth once the customer confirms.
 */
export function CartReviewScreen(): React.JSX.Element {
  const goToStage = useSessionStore((s) => s.goToStage);

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-surface">
      <Suspense fallback={<CartSkeleton />}>
        <CartRemote
          onBackToMenu={() => { goToStage("menu"); }}
          onProceedToPayment={() => { goToStage("payment_auth"); }}
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
