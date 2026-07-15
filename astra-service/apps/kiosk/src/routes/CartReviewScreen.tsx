import { useMemo, useEffect, useState, useCallback } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useQueryClient } from "@tanstack/react-query";
import { useSnapshot } from "valtio";
import { cartProxy } from "@astra/kiosk-state";
import type { MenuItem, ReadonlyCartLineItem } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { cartService } from "../state/cartService";
import { defaultLogger } from "../utils/logger";

const log = defaultLogger.child("CartReviewScreen");

const SILENT_ASSIST_DELAY_MS = 40_000;
const TAX_RATE = Number.parseFloat(
  (import.meta.env as Record<string, string | undefined>)["VITE_TAX_RATE"] ?? "0.08",
);

function formatCents(cents: number): string {
  return (cents / 100).toFixed(2);
}

function lineTotalCents(line: ReadonlyCartLineItem): number {
  return (
    line.quantity *
    (line.unitPriceCentsSnapshot +
      line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0))
  );
}

export function CartReviewScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const cart = useSnapshot(cartProxy);
  const [silentAssist, setSilentAssist] = useState(false);
  const queryClient = useQueryClient();

  const handleItemTap = useCallback(
    (menuItemId: string) => {
      const cached = queryClient.getQueryData<{ items: readonly MenuItem[] }>(["menu-catalog"]);
      const item = cached?.items.find((i) => i.itemId === menuItemId);
      if (item) {
        send({ type: "SELECT_ITEM", item });
      }
    },
    [queryClient, send],
  );

  const itemCount = cart.lines.reduce((sum, l) => sum + l.quantity, 0);
  const isFullScreen = itemCount > 5;

  const subtotalCents = useMemo(
    () => cart.lines.reduce((sum, l) => sum + lineTotalCents(l), 0),
    [cart.lines],
  );
  const taxCents = Math.round(subtotalCents * TAX_RATE);
  const totalCents = subtotalCents + taxCents;

  useEffect(() => {
    const timer = setTimeout(() => { setSilentAssist(true); }, SILENT_ASSIST_DELAY_MS);
    return () => { clearTimeout(timer); };
  }, []);

  const handleQuantityChange = useCallback(
    (lineId: string, delta: number) => {
      const line = cart.lines.find((l) => l.lineId === lineId);
      if (!line) return;
      const next = line.quantity + delta;
      try {
        if (next <= 0) {
          cartService.removeItem(lineId);
        } else {
          cartService.updateQuantity(lineId, next);
        }
      } catch (error) {
        log.error("Failed to update cart quantity", error);
      }
    },
    [cart.lines],
  );

  const handlePay = useCallback(() => {
    send({ type: "PROCEED_TO_PAYMENT" });
  }, [send]);

  const handleBack = useCallback(() => {
    send({ type: "BACK_TO_MENU" });
  }, [send]);

  const content = (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="px-3 pb-2 pt-4">
        <h1 className="font-heading text-[32px] font-semibold text-charcoal">
          Your cart
        </h1>
      </div>

      {/* Cart items */}
      <div
        className="flex-1 overflow-y-auto px-3"
        role="list"
        aria-label="Cart items"
      >
        {cart.lines.length === 0 && (
          <div className="flex flex-col items-center justify-center py-16">
            <p className="font-sans text-[18px] text-stone">Your cart is empty</p>
          </div>
        )}
        {cart.lines.map((line, idx) => (
          <div
            key={line.lineId}
            role="listitem"
            className="cursor-pointer active:bg-warm-cream/50 rounded-[12px] transition-colors duration-100"
            onClick={() => { handleItemTap(line.menuItemId); }}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") handleItemTap(line.menuItemId);
            }}
            tabIndex={0}
            aria-label={`Edit ${line.nameSnapshot}. Quantity: ${String(line.quantity)}. Price: $${formatCents(lineTotalCents(line))}`}
          >
            <div className="flex items-start gap-3 py-3 px-1">
              {/* Thumbnail */}
              <div className="h-16 w-16 shrink-0 rounded-[12px] bg-stone/10 overflow-hidden">
                <div
                  className="h-full w-full"
                  style={{
                    background:
                      "linear-gradient(135deg, rgba(107,104,98,0.08), rgba(196,184,168,0.08))",
                  }}
                  aria-hidden="true"
                />
              </div>

              {/* Details */}
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex items-start justify-between gap-2">
                  <span className="font-sans text-[18px] font-medium text-charcoal truncate">
                    {line.nameSnapshot}
                  </span>
                  <span className="font-sans text-[18px] font-semibold text-charcoal tabular-nums shrink-0">
                    ${formatCents(lineTotalCents(line))}
                  </span>
                </div>

                {/* Modifiers */}
                {line.modifiers.length > 0 && (
                  <p className="font-sans text-[14px] text-stone truncate">
                    {line.modifiers
                      .map((m) => `${m.modifierId}: +$${formatCents(m.priceDeltaCents)}`)
                      .join(", ")}
                  </p>
                )}

                {/* Quantity stepper */}
                <div className="mt-1 flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      handleQuantityChange(line.lineId, -1);
                    }}
                    className="h-12 w-12 rounded-full bg-linen border border-taupe flex items-center justify-center active:bg-white/80 transition-colors duration-100"
                    aria-label={`Decrease quantity of ${line.nameSnapshot}`}
                  >
                    <svg
                      viewBox="0 0 20 20"
                      className="h-5 w-5 text-charcoal"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth={2}
                      aria-hidden="true"
                    >
                      <path d="M5 10h10" strokeLinecap="round" />
                    </svg>
                  </button>
                  <span
                    className="font-sans text-[20px] font-semibold text-charcoal tabular-nums text-center min-w-[48px]"
                    aria-label={`Quantity: ${String(line.quantity)}`}
                  >
                    {line.quantity}
                  </span>
                  <button
                    type="button"
                    onClick={() => {
                      handleQuantityChange(line.lineId, 1);
                    }}
                    className="h-12 w-12 rounded-full bg-linen border border-taupe flex items-center justify-center active:bg-white/80 transition-colors duration-100"
                    aria-label={`Increase quantity of ${line.nameSnapshot}`}
                  >
                    <svg
                      viewBox="0 0 20 20"
                      className="h-5 w-5 text-charcoal"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth={2}
                      aria-hidden="true"
                    >
                      <path d="M10 5v10M5 10h10" strokeLinecap="round" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            {/* Dashed divider */}
            {idx < cart.lines.length - 1 && (
              <div className="border-t border-dashed border-taupe/40" />
            )}
          </div>
        ))}

        {/* Tap to edit hint */}
        {cart.lines.length > 0 && (
          <p className="mt-4 text-center font-sans text-[14px] text-stone">
            Tap an item to edit
          </p>
        )}
      </div>

      {/* Summary */}
      <div className="px-3 pb-3">
        <div className="flex flex-col gap-2 border-t border-taupe pt-3">
          <div className="flex items-center justify-between">
            <span className="font-sans text-[13px] font-medium uppercase tracking-[0.08em] text-stone">
              Subtotal
            </span>
            <span className="font-sans text-[18px] text-charcoal tabular-nums">
              ${formatCents(subtotalCents)}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="font-sans text-[13px] font-medium uppercase tracking-[0.08em] text-stone">
              Tax
            </span>
            <span className="font-sans text-[18px] text-charcoal tabular-nums">
              ${formatCents(taxCents)}
            </span>
          </div>
          <div className="flex items-center justify-between border-t border-taupe pt-2">
            <span className="font-sans text-[18px] font-medium text-charcoal">
              Total
            </span>
            <span className="font-sans text-[42px] font-semibold text-amber tabular-nums">
              ${formatCents(totalCents)}
            </span>
          </div>
        </div>
      </div>

      {/* Action bar */}
      <div className="flex items-center gap-3 border-t border-taupe bg-linen px-3 py-3">
        <button
          type="button"
          onClick={handleBack}
          className="h-14 flex-1 rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal active:bg-warm-cream/50 transition-colors duration-100"
          aria-label="Back to menu"
        >
          ← Back to menu
        </button>
        <motion.button
          type="button"
          onClick={handlePay}
          className="h-14 flex-[2] rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)] active:scale-[0.98] active:translate-y-[1px] transition-all duration-100"
          {...(silentAssist
            ? {
                animate: { opacity: [0.8, 1, 0.8] },
                transition: {
                  duration: 2,
                  repeat: Infinity,
                  ease: "easeInOut",
                },
              }
            : {})}
          aria-label={`Pay $${formatCents(totalCents)}`}
        >
          Pay ${formatCents(totalCents)} →
        </motion.button>
      </div>
    </div>
  );

  if (isFullScreen) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: motionTokens.easeOutExpo }}
        className="flex flex-1 flex-col bg-linen"
      >
        {content}
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ y: "100%" }}
      animate={{ y: 0 }}
      exit={{ y: "100%" }}
      transition={{
        duration: 0.3,
        ease: motionTokens.easeOutExpo,
      }}
      className="fixed inset-0 z-30 flex flex-col bg-white/95 backdrop-blur-[12px] rounded-t-[24px] shadow-[0_8px_32px_rgba(45,42,38,0.12)]"
    >
      {/* Handle */}
      <div className="mx-auto mt-3 h-1 w-10 rounded bg-taupe" aria-hidden="true" />
      {content}
    </motion.div>
  );
}

