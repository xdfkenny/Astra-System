import { useMemo, useEffect, useState, useCallback, useRef } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSnapshot } from "valtio";
import {
  cartProxy,
} from "@astra/kiosk-state";
import type { ReadonlyCartLineItem } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { cartService } from "../state/cartService";
import { BottomSheet } from "../components/BottomSheet";

const SILENT_ASSIST_DELAY_MS = 40_000;

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
  const [editingLineId, setEditingLineId] = useState<string | null>(null);

  const editingLine = editingLineId
    ? cart.lines.find((l) => l.lineId === editingLineId)
    : undefined;

  const itemCount = cart.lines.reduce((sum, l) => sum + l.quantity, 0);
  const isFullScreen = itemCount > 5;

  const subtotalCents = useMemo(
    () => cart.lines.reduce((sum, l) => sum + lineTotalCents(l), 0),
    [cart.lines],
  );
  const taxCents = Math.round(subtotalCents * 0.08);
  const totalCents = subtotalCents + taxCents;

  useEffect(() => {
    const timer = setTimeout(() => { setSilentAssist(true); }, SILENT_ASSIST_DELAY_MS);
    return () => { clearTimeout(timer); };
  }, []);

  const handleQuantityChange = useCallback(
    async (lineId: string, delta: number) => {
      const line = cart.lines.find((l) => l.lineId === lineId);
      if (!line) return;
      const next = line.quantity + delta;
      if (next <= 0) {
        try {
          await cartService.removeItem(lineId);
        } catch (error) {
          console.error("Failed to remove item from cart:", error);
        }
      } else {
        try {
          await cartService.updateQuantity(lineId, next);
        } catch (error) {
          console.error("Failed to update quantity:", error);
        }
      }
    },
    [cart.lines],
  );

  const holdRef = useRef<{ timeout?: number; interval?: number }>({});

  const endHold = useCallback(() => {
    if (holdRef.current.timeout !== undefined) {
      window.clearTimeout(holdRef.current.timeout);
    }
    if (holdRef.current.interval !== undefined) {
      window.clearInterval(holdRef.current.interval);
    }
    holdRef.current = {};
  }, []);

  // Long-press accelerates after 500ms, repeating every 100ms. The initial
  // change is handled by the button's onClick so keyboard activation still works.
  const startHold = useCallback(
    (lineId: string, delta: number) => {
      endHold();
      holdRef.current.timeout = window.setTimeout(() => {
        holdRef.current.interval = window.setInterval(() => {
          void handleQuantityChange(lineId, delta);
        }, 100);
      }, 500);
    },
    [endHold, handleQuantityChange],
  );

  useEffect(() => endHold, [endHold]);

  const holdProps = useCallback(
    (lineId: string, delta: number) => ({
      onPointerDown: () => { startHold(lineId, delta); },
      onPointerUp: endHold,
      onPointerLeave: endHold,
      onPointerCancel: endHold,
    }),
    [startHold, endHold],
  );

  const handleRemove = useCallback(async (lineId: string) => {
    try {
      await cartService.removeItem(lineId);
    } catch (error) {
      console.error("Failed to remove item from cart:", error);
    }
    setEditingLineId(null);
  }, []);

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
        {cart.lines.map((line, idx) => (
          <div key={line.lineId} role="listitem">
            <div className="flex flex-col gap-2 py-3">
              {/* Tappable item summary — opens edit sheet */}
              <button
                type="button"
                onClick={() => { setEditingLineId(line.lineId); }}
                className="flex items-start gap-3 text-left rounded-[12px] transition-colors active:bg-warm-cream/50"
                aria-label={`Edit ${line.nameSnapshot}`}
              >
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
                </div>
              </button>

              {/* Quantity stepper */}
              <div className="ml-[76px] flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => handleQuantityChange(line.lineId, -1)}
                  {...holdProps(line.lineId, -1)}
                  className="h-14 w-14 rounded-full bg-linen border border-taupe flex items-center justify-center"
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
                  onClick={() => handleQuantityChange(line.lineId, 1)}
                  {...holdProps(line.lineId, 1)}
                  className="h-14 w-14 rounded-full bg-linen border border-taupe flex items-center justify-center"
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

            {/* Dashed divider */}
            {idx < cart.lines.length - 1 && (
              <div className="border-t border-dashed border-taupe" />
            )}
          </div>
        ))}

        {/* Tap to edit hint */}
        <p className="mt-4 text-center font-sans text-[14px] text-stone">
          Tap an item to change quantity or remove it
        </p>
      </div>

      {/* Summary */}
      <div className="px-3 pb-3">
        <div className="flex flex-col gap-2 border-t border-taupe pt-3">
          <div className="flex items-center justify-between">
            <span className="font-sans text-caption uppercase tracking-[0.08em] text-stone">
              Subtotal
            </span>
            <span className="font-sans text-[18px] text-charcoal tabular-nums">
              ${formatCents(subtotalCents)}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="font-sans text-caption uppercase tracking-[0.08em] text-stone">
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
          className="h-14 flex-1 rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal"
          aria-label="Back to menu"
        >
          ← Back to menu
        </button>
        <motion.button
          type="button"
          onClick={handlePay}
          className="h-14 flex-[2] rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)]"
          {...(silentAssist
            ? {
                animate: { opacity: [0.8, 1, 0.8] },
                transition: { duration: 2, repeat: Infinity, ease: "easeInOut" },
              }
            : {})}
          aria-label={`Pay $${formatCents(totalCents)}`}
        >
          Pay ${formatCents(totalCents)} →
        </motion.button>
      </div>

      {/* Edit item sheet */}
      <BottomSheet
        open={editingLine !== undefined}
        onClose={() => { setEditingLineId(null); }}
      >
        {editingLine && (
          <div className="flex flex-col gap-4">
            <div className="flex items-start justify-between gap-2">
              <h2 className="font-heading text-[24px] font-semibold text-charcoal">
                {editingLine.nameSnapshot}
              </h2>
              <span className="font-sans text-[18px] font-semibold text-charcoal tabular-nums shrink-0">
                ${formatCents(lineTotalCents(editingLine))}
              </span>
            </div>

            {editingLine.modifiers.length > 0 && (
              <p className="font-sans text-[14px] text-stone">
                {editingLine.modifiers
                  .map((m) => `${m.modifierId}: +$${formatCents(m.priceDeltaCents)}`)
                  .join(", ")}
              </p>
            )}

            <div className="flex items-center justify-center gap-4">
              <button
                type="button"
                onClick={() => handleQuantityChange(editingLine.lineId, -1)}
                {...holdProps(editingLine.lineId, -1)}
                className="h-14 w-14 rounded-full bg-linen border border-taupe flex items-center justify-center"
                aria-label={`Decrease quantity of ${editingLine.nameSnapshot}`}
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
                className="font-sans text-[24px] font-semibold text-charcoal tabular-nums text-center min-w-[56px]"
                aria-label={`Quantity: ${String(editingLine.quantity)}`}
              >
                {editingLine.quantity}
              </span>
              <button
                type="button"
                onClick={() => handleQuantityChange(editingLine.lineId, 1)}
                {...holdProps(editingLine.lineId, 1)}
                className="h-14 w-14 rounded-full bg-linen border border-taupe flex items-center justify-center"
                aria-label={`Increase quantity of ${editingLine.nameSnapshot}`}
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

            <button
              type="button"
              onClick={() => { void handleRemove(editingLine.lineId); }}
              className="h-14 w-full rounded-[16px] border border-soft-rose bg-white/70 font-sans text-[16px] font-medium text-soft-rose"
              aria-label={`Remove ${editingLine.nameSnapshot} from cart`}
            >
              Remove from cart
            </button>
          </div>
        )}
      </BottomSheet>
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
      className="fixed inset-0 z-30 flex flex-col bg-white/95 backdrop-blur-[12px] rounded-t-[24px]"
    >
      {/* Handle */}
      <div className="mx-auto mt-3 h-1 w-10 rounded bg-taupe" aria-hidden="true" />
      {content}
    </motion.div>
  );
}
