import { useSnapshot } from "valtio";
import { AnimatePresence, motion } from "framer-motion";
import { cartProxy } from "@astra/kiosk-state";
import { PrimaryButton } from "@astra/ui-kit";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";

/**
 * Persistent floating cart summary — visible while browsing the menu. Tapping
 * it drives the primary "review cart" transition. Bottom-aligned per the
 * thumb-zone layout spec.
 */
export function CartSummary(): React.JSX.Element | null {
  const cart = useSnapshot(cartProxy);
  const { state, send } = useKioskMachine();

  const itemCount = cart.lines.reduce((sum, line) => sum + line.quantity, 0);
  const totalCents = cart.lines.reduce(
    (sum, line) =>
      sum +
      line.quantity *
        (line.unitPriceCentsSnapshot + line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
    0,
  );

  const visible = state.value === "MENU_BROWSE" || state.value === "ITEM_MODAL";

  return (
    <AnimatePresence>
      {visible && itemCount > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 24 }}
          transition={{
            duration: motionTokens.durationBase,
            ease: motionTokens.easeStandard,
          }}
          className="absolute bottom-6 left-4 right-4"
        >
          <PrimaryButton
            variant="accent"
            className="w-full"
            style={{ minHeight: "72px" }}
            onClick={() => {
              send({ type: "GO_TO_CART" });
            }}
            aria-label={`Review cart, ${String(itemCount)} items, total $${(totalCents / 100).toFixed(2)}`}
          >
            <span className="flex w-full items-center justify-between">
              <span className="flex items-center gap-3 font-heading text-xl font-semibold">
                <span className="flex h-9 w-9 items-center justify-center rounded-full bg-white/20 text-base">
                  {itemCount}
                </span>
                View Cart
              </span>
              <span className="font-heading text-2xl font-bold tabular-nums">${(totalCents / 100).toFixed(2)}</span>
            </span>
          </PrimaryButton>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
