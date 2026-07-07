import { useSnapshot } from "valtio";
import { AnimatePresence, motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { PrimaryButton, TransparencyPanel, EmptyState } from "@astra/ui-kit";
import { cartProxy, removeLineItem, updateLineQuantity } from "@astra/kiosk-state";
import { useCartTotals } from "@astra/cart-engine";

export interface CartAppProps {
  readonly onBackToMenu: () => void;
  readonly onProceedToPayment: () => void;
}

/**
 * Federated Cart micro-frontend. Reads the host's Valtio cart proxy directly
 * (shared singleton via Module Federation's `shared` config in vite.config.ts
 * for `valtio` itself would create two proxy instances — instead we import
 * the *state module* from the host at runtime, keeping exactly one source of
 * truth for cart contents across federation boundaries).
 */
export default function CartApp({ onBackToMenu, onProceedToPayment }: CartAppProps): React.JSX.Element {
  const cart = useSnapshot(cartProxy);
  const totals = useCartTotals(cart.lines, false);

  if (cart.lines.length === 0) {
    return (
      <EmptyState
        title="Your cart is empty"
        description="Browse the menu to add items — everything you scan or select appears here."
        actionLabel="Back to Menu"
        onAction={onBackToMenu}
      />
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <div className="hairline flex h-14 shrink-0 items-center justify-between px-4">
        <button type="button" onClick={onBackToMenu} className="text-sm font-medium text-primary">
          &larr; Add more items
        </button>
        <h2 className="font-heading text-lg font-semibold">Your Order</h2>
        <span className="w-24" />
      </div>

      <ul className="flex flex-1 flex-col gap-2 overflow-y-auto p-4" aria-label="Cart items">
        <AnimatePresence initial={false}>
          {cart.lines.map((line) => (
            <motion.li
              key={line.lineId}
              layout
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: motionTokens.durationBase, ease: motionTokens.easeStandard }}
              className="hairline flex items-center justify-between rounded-md bg-surface p-3"
            >
              <div className="flex-1">
                <p className="font-medium text-ink">{line.nameSnapshot}</p>
                {line.modifiers.length > 0 ? (
                  <p className="text-sm text-ink-muted">
                    {line.modifiers.length} modifier{line.modifiers.length > 1 ? "s" : ""}
                  </p>
                ) : null}
                <p className="mt-1 text-sm font-medium tabular-nums text-ink">
                  ${(line.unitPriceCentsSnapshot / 100).toFixed(2)}
                </p>
              </div>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  aria-label={`Decrease quantity of ${line.nameSnapshot}`}
                  onClick={() => { updateLineQuantity(line.lineId, line.quantity - 1); }}
                  className="flex h-12 w-12 items-center justify-center rounded-md border border-border-strong text-xl"
                >
                  &minus;
                </button>
                <span className="w-6 text-center text-lg font-semibold tabular-nums">
                  {line.quantity}
                </span>
                <button
                  type="button"
                  aria-label={`Increase quantity of ${line.nameSnapshot}`}
                  onClick={() => { updateLineQuantity(line.lineId, line.quantity + 1); }}
                  className="flex h-12 w-12 items-center justify-center rounded-md border border-border-strong text-xl"
                >
                  +
                </button>
                <button
                  type="button"
                  aria-label={`Remove ${line.nameSnapshot} from cart`}
                  onClick={() => { removeLineItem(line.lineId); }}
                  className="ml-1 flex h-12 w-12 items-center justify-center rounded-md text-error"
                >
                  <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth={2}>
                    <path d="M6 6l12 12M18 6L6 18" strokeLinecap="round" />
                  </svg>
                </button>
              </div>
            </motion.li>
          ))}
        </AnimatePresence>
      </ul>

      <div className="hairline flex flex-col gap-3 border-t bg-surface p-4">
        <TransparencyPanel totals={totals} />
        <div className="flex items-center justify-between">
          <span className="font-heading text-lg font-semibold">Total</span>
          <span className="font-heading text-2xl font-bold tabular-nums">
            ${(totals.totalCents / 100).toFixed(2)}
          </span>
        </div>
        <PrimaryButton variant="accent" onClick={onProceedToPayment} className="w-full">
          Checkout
        </PrimaryButton>
      </div>
    </div>
  );
}
