import { useSnapshot } from "valtio";
import { AnimatePresence, motion } from "framer-motion";
import { cartProxy, derivedCart, useSessionStore } from "@astra/kiosk-state";
import { motion as motionTokens } from "@astra/design-tokens";

/**
 * Persistent floating cart summary — visible during menu browsing, tapping
 * it drives the primary "review cart" transition. Bottom-aligned per the
 * thumb-zone layout spec. Uses only transform/opacity for the enter/exit
 * animation (hardware-accelerated, 60fps on embedded ARM GPUs).
 */
export function FloatingCartSummary(): React.JSX.Element | null {
  const cart = useSnapshot(cartProxy);
  const derived = useSnapshot(derivedCart);
  const goToStage = useSessionStore((s) => s.goToStage);

  const totalCents = cart.lines.reduce(
    (sum, line) =>
      sum +
      line.quantity *
        (line.unitPriceCentsSnapshot +
          line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
    0,
  );

  return (
    <AnimatePresence>
      {!derived.isEmpty && (
        <motion.button
          type="button"
          onClick={() => { goToStage("cart_review"); }}
          initial={{ opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 24 }}
          transition={{ duration: motionTokens.durationBase, ease: motionTokens.easeStandard }}
          className="absolute bottom-6 left-4 right-4 flex h-[var(--touch-primary-action)] items-center justify-between rounded-lg bg-primary px-6 text-white shadow-lg active:bg-primary-pressed"
          style={{ minHeight: "88px" }}
          aria-label={`Review cart, ${String(derived.itemCount)} items, total $${(totalCents / 100).toFixed(2)}`}
        >
          <span className="flex items-center gap-3 font-heading text-xl font-semibold">
            <span className="flex h-9 w-9 items-center justify-center rounded-full bg-white/20 text-base">
              {derived.itemCount}
            </span>
            View Cart
          </span>
          <span className="font-heading text-2xl font-bold tabular-nums">
            ${(totalCents / 100).toFixed(2)}
          </span>
        </motion.button>
      )}
    </AnimatePresence>
  );
}
