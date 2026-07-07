import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import type { CartTotals } from "@astra/shared-types";

export interface TransparencyPanelProps {
  readonly totals: CartTotals;
}

/**
 * Deep-improvement #8: "Why this price?" transparency panel. Renders the
 * full tax/discount/fee/loyalty breakdown behind a single tap so the
 * default cart view stays uncluttered but full price transparency is
 * always one interaction away — this is both a UX differentiator and a
 * growing legal requirement (junk-fee disclosure laws in several US states).
 */
export function TransparencyPanel({ totals }: TransparencyPanelProps): React.JSX.Element {
  const [open, setOpen] = useState(false);

  return (
    <div className="w-full">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-12 w-full items-center justify-between text-sm font-medium text-primary underline decoration-dotted underline-offset-4"
        aria-expanded={open}
        aria-controls="transparency-panel-detail"
      >
        Why this price?
        <span aria-hidden="true">{open ? "\u2212" : "+"}</span>
      </button>
      <AnimatePresence initial={false}>
        {open ? (
          <motion.div
            id="transparency-panel-detail"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: motionTokens.durationBase, ease: motionTokens.easeStandard }}
            className="overflow-hidden"
          >
            <ul className="flex flex-col gap-2 rounded-md bg-surface-sunken p-4 text-sm">
              {totals.breakdown.map((entry, idx) => (
                <li key={idx} className="flex items-center justify-between text-ink-muted">
                  <span>{entry.label}</span>
                  <span className="tabular-nums">
                    {entry.amountCents < 0 ? "-" : ""}$
                    {(Math.abs(entry.amountCents) / 100).toFixed(2)}
                  </span>
                </li>
              ))}
            </ul>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}
