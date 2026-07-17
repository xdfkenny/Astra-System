import { useSnapshot } from "valtio";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cartProxy } from "@astra/kiosk-state";
import { BottomSheet } from "./BottomSheet";
import { useState } from "react";
import { cn } from "@/utils/cn";
import { useTranslation } from "../i18n";
import { useCurrencyFormat } from "../i18n/useCurrencyFormat";

export interface CartSummaryProps {
  readonly className?: string;
  readonly onCheckout?: () => void;
}

export function CartSummary({ className, onCheckout }: CartSummaryProps): React.JSX.Element | null {
  const { t } = useTranslation();
  const { formatCurrency } = useCurrencyFormat();
  const cart = useSnapshot(cartProxy);
  const [expanded, setExpanded] = useState(false);

  const itemCount = cart.lines.reduce((sum, line) => sum + line.quantity, 0);
  const totalCents = cart.lines.reduce(
    (sum, line) =>
      sum +
      line.quantity *
        (line.unitPriceCentsSnapshot +
          line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
    0,
  );

  if (itemCount === 0) return null;

  return (
    <>
      <motion.button
        type="button"
        onClick={() => {
          if (onCheckout) {
            onCheckout();
          } else {
            setExpanded(true);
          }
        }}
        className={cn(
          "sticky bottom-0 z-20 flex w-full items-center justify-between bg-warm-cream/90 px-3 py-2 backdrop-blur-[8px] border-t border-taupe/30",
          className,
        )}
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
        aria-label={t("cart.summary") + `: ${itemCount === 1 ? t("cart.itemSingular", { count: itemCount }) : t("cart.itemCount", { count: itemCount })}, ${t("cart.total")} ${formatCurrency(totalCents)}.`}
        aria-expanded={expanded}
      >
        <span className="font-sans text-[13px] font-medium uppercase tracking-[0.08em] text-stone">
          {itemCount === 1 ? t("cart.itemSingular", { count: itemCount }) : t("cart.itemCount", { count: itemCount })}
        </span>
        <span className="font-sans text-[28px] font-semibold text-charcoal tabular-nums">
          {formatCurrency(totalCents)}
        </span>
      </motion.button>

      <BottomSheet open={expanded} onClose={() => { setExpanded(false); }} aria-label={t("cart.summary")}>
        <h2 className="font-heading text-[32px] font-semibold text-charcoal mb-4">
          {t("cart.title")}
        </h2>
        <ul className="flex flex-col gap-3" role="list">
          {cart.lines.map((line) => (
            <li
              key={line.lineId}
              className="flex items-center justify-between border-b border-dashed border-taupe/40 pb-3"
            >
              <div className="flex flex-col">
                <span className="font-sans text-[18px] font-medium text-charcoal truncate max-w-[200px]">
                  {line.nameSnapshot}
                </span>
                {line.modifiers.length > 0 && (
                  <span className="font-sans text-[14px] text-stone">
                    {line.modifiers.length > 1
                      ? t("item.modifierPlural", { count: line.modifiers.length })
                      : t("item.modifierSingular", { count: line.modifiers.length })}
                  </span>
                )}
              </div>
              <span className="font-sans text-[18px] font-semibold text-charcoal tabular-nums">
                {formatCurrency(
                  line.unitPriceCentsSnapshot +
                    line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0),
                )}
              </span>
            </li>
          ))}
        </ul>
        <div className="mt-4 flex items-center justify-between border-t border-taupe pt-3">
          <span className="font-sans text-[13px] font-medium uppercase tracking-[0.08em] text-stone">
            {t("cart.total")}
          </span>
          <span className="font-sans text-[42px] font-semibold text-amber tabular-nums">
            {formatCurrency(totalCents)}
          </span>
        </div>
      </BottomSheet>
    </>
  );
}

