import { useState, useMemo, useCallback, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSnapshot } from "valtio";
import { cartProxy } from "@astra/kiosk-state";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { apiClient } from "../state/apiClient";

type PaymentMethod = "card_nfc" | "cash" | "qr_code";

interface PaymentMethodOption {
  readonly id: PaymentMethod;
  readonly label: string;
  readonly icon: string;
}

const PAYMENT_METHODS: readonly PaymentMethodOption[] = [
  { id: "card_nfc", label: "Card / NFC", icon: "card" },
  { id: "cash", label: "Cash", icon: "cash" },
  { id: "qr_code", label: "QR Code", icon: "qr" },
];

function formatCents(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function PaymentAuthScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const cart = useSnapshot(cartProxy);
  const [selectedMethod, setSelectedMethod] = useState<PaymentMethod | null>(null);
  const [cartExpanded, setCartExpanded] = useState(false);
  const [showBiometric, setShowBiometric] = useState(false);
  const employeeHoldRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [employeeHoldProgress, setEmployeeHoldProgress] = useState(0);

  const itemCount = cart.lines.reduce((sum, l) => sum + l.quantity, 0);
  const subtotalCents = useMemo(
    () =>
      cart.lines.reduce(
        (sum, l) =>
          sum +
          l.quantity *
            (l.unitPriceCentsSnapshot +
              l.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
        0,
      ),
    [cart.lines],
  );
  const taxCents = Math.round(subtotalCents * 0.08);
  const totalCents = subtotalCents + taxCents;

  const handleConfirm = useCallback(() => {
    if (!selectedMethod) return;
    setShowBiometric(true);
  }, [selectedMethod]);

  const handleBiometricComplete = useCallback(async () => {
    setShowBiometric(false);
    try {
      // Map the UI payment method to the API payment method
      const apiMethod = selectedMethod === "card_nfc" ? "nfc_apple_pay" : selectedMethod === "cash" ? "cash_recycler" : "qr_code";
      
      // Create a checkout first
      const checkoutResponse = await apiClient.checkoutCart(
        cartProxy.cartId,
        apiMethod,
      );
      
      // Process the payment
      const paymentResult = await apiClient.processPayment(
        cartProxy.cartId,
        checkoutResponse.paymentIntentId,
        totalCents,
        "USD",
        apiMethod,
      );
      
      send({
        type: "PAYMENT_AUTHORIZED",
        result: {
          authorizationId: paymentResult.paymentId,
          status: paymentResult.status,
          method: paymentResult.method,
          amountCents: paymentResult.amountCents,
             // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
             ...(paymentResult.authorization?.approvalCode && { approvalCode: paymentResult.authorization.approvalCode }),
             // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
             ...(paymentResult.authorization?.cardBrand && { cardBrand: paymentResult.authorization.cardBrand }),
             // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
             ...(paymentResult.authorization?.cardLastFour && { cardLastFour: paymentResult.authorization.cardLastFour }),
        },
      });
    } catch (error) {
      console.error("Payment processing failed:", error);
      send({
        type: "PAYMENT_FAILED",
        message: error instanceof Error ? error.message : "Payment processing failed",
      });
    }
  }, [send, selectedMethod, totalCents]);

  const handleEmployeeHoldStart = useCallback(() => {
    let progress = 0;
    setEmployeeHoldProgress(0);
    employeeHoldRef.current = setInterval(() => {
      progress += 0.1;
      setEmployeeHoldProgress(Math.min(progress, 1));
      if (progress >= 1) {
        if (employeeHoldRef.current) clearInterval(employeeHoldRef.current);
        // Employee override - authorize payment without actual processing
        send({ 
          type: "PAYMENT_AUTHORIZED", 
          result: { 
            authorizationId: crypto.randomUUID(), 
            status: "authorized", 
            method: "nfc_apple_pay", 
            amountCents: totalCents,
          } 
        });
      }
    }, 300);
  }, [send, totalCents]);

  const handleEmployeeHoldEnd = useCallback(() => {
    if (employeeHoldRef.current) {
      clearInterval(employeeHoldRef.current);
      employeeHoldRef.current = null;
    }
    setEmployeeHoldProgress(0);
  }, []);

  return (
    <div className="flex flex-1 flex-col bg-linen">
      {/* Header */}
      <div className="px-3 pb-2 pt-4">
        <h1 className="font-heading text-[28px] font-semibold text-charcoal">
          Ready to pay
        </h1>
      </div>

      {/* Collapsible cart summary */}
      <div className="px-3">
        <button
          type="button"
          onClick={() => { setCartExpanded(!cartExpanded); }}
          className="flex w-full items-center justify-between rounded-[12px] bg-warm-cream/90 px-4 py-3 backdrop-blur-[8px]"
          aria-expanded={cartExpanded}
          aria-label={`Cart summary: ${String(itemCount)} items, total $${formatCents(totalCents)}. Tap to ${cartExpanded ? "collapse" : "expand"}.`}
        >
          <span className="font-sans text-caption uppercase tracking-[0.08em] text-stone">
            {itemCount} {itemCount === 1 ? "item" : "items"}
          </span>
          <span className="font-sans text-[28px] font-semibold text-charcoal tabular-nums">
            ${formatCents(totalCents)}
          </span>
        </button>

        <AnimatePresence>
          {cartExpanded && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.25, ease: motionTokens.easeInOutSoft }}
              className="overflow-hidden"
            >
              <div className="mt-2 rounded-[12px] bg-white/70 px-4 py-3">
                {cart.lines.map((line) => (
                  <div
                    key={line.lineId}
                    className="flex items-center justify-between py-1"
                  >
                    <span className="font-sans text-[16px] text-charcoal truncate max-w-[200px]">
                      {line.nameSnapshot} × {line.quantity}
                    </span>
                    <span className="font-sans text-[16px] text-stone tabular-nums">
                      ${formatCents(
                        line.quantity *
                          (line.unitPriceCentsSnapshot +
                            line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
                      )}
                    </span>
                  </div>
                ))}
                <div className="mt-2 flex items-center justify-between border-t border-taupe pt-2">
                  <span className="font-sans text-caption uppercase tracking-[0.08em] text-stone">
                    Total
                  </span>
                  <span className="font-sans text-[28px] font-semibold text-amber tabular-nums">
                    ${formatCents(totalCents)}
                  </span>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Payment methods */}
      <div className="mt-4 px-3">
        <h2 className="font-sans text-caption uppercase tracking-[0.08em] text-stone mb-2">
          Select payment method
        </h2>
        <div className="flex gap-3 overflow-x-auto snap-x snap-mandatory pb-2">
          {PAYMENT_METHODS.map((method) => {
            const isSelected = selectedMethod === method.id;
            return (
              <button
                key={method.id}
                type="button"
                onClick={() => { setSelectedMethod(method.id); }}
                className={`snap-start flex shrink-0 flex-col items-center justify-center gap-2 rounded-[16px] border-2 transition-colors duration-150 ${
                  isSelected
                    ? "border-moss bg-pale-mint/20"
                    : "border-taupe bg-white/60"
                }`}
                style={{ width: "120px", height: "120px" }}
                aria-pressed={isSelected}
                aria-label={method.label}
              >
                {/* Icon */}
                {method.id === "card_nfc" && (
                  <svg viewBox="0 0 32 32" className="h-8 w-8 text-charcoal" fill="none" stroke="currentColor" strokeWidth={1.5} aria-hidden="true">
                    <rect x="4" y="8" width="24" height="16" rx="3" />
                    <path d="M16 14v4" strokeLinecap="round" />
                    <circle cx="16" cy="18" r="1" fill="currentColor" stroke="none" />
                  </svg>
                )}
                {method.id === "cash" && (
                  <svg viewBox="0 0 32 32" className="h-8 w-8 text-charcoal" fill="none" stroke="currentColor" strokeWidth={1.5} aria-hidden="true">
                    <rect x="2" y="10" width="28" height="14" rx="3" />
                    <circle cx="16" cy="17" r="4" />
                    <path d="M2 14h4M26 14h4" />
                  </svg>
                )}
                {method.id === "qr_code" && (
                  <svg viewBox="0 0 32 32" className="h-8 w-8 text-charcoal" fill="none" stroke="currentColor" strokeWidth={1.5} aria-hidden="true">
                    <rect x="4" y="4" width="10" height="10" rx="1" />
                    <rect x="18" y="4" width="10" height="10" rx="1" />
                    <rect x="4" y="18" width="10" height="10" rx="1" />
                    <path d="M18 22h10M22 18v10" />
                  </svg>
                )}
                <span className="font-sans text-[14px] font-medium text-charcoal">
                  {method.label}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Confirm payment button */}
      <div className="px-3 pb-3">
        <button
          type="button"
          disabled={!selectedMethod}
          onClick={handleConfirm}
          className="h-16 w-full rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)] disabled:opacity-50 disabled:grayscale-[0.5] transition-all duration-100 active:scale-[0.98] active:translate-y-[1px]"
          aria-label={`Pay $${formatCents(totalCents)}`}
        >
          Pay ${formatCents(totalCents)}
        </button>
      </div>

      {/* Employee override — hidden, long-press corner */}
      <button
        type="button"
        onMouseDown={handleEmployeeHoldStart}
        onMouseUp={handleEmployeeHoldEnd}
        onMouseLeave={handleEmployeeHoldEnd}
        onTouchStart={handleEmployeeHoldStart}
        onTouchEnd={handleEmployeeHoldEnd}
        className="absolute bottom-0 right-0 h-16 w-16 opacity-0"
        aria-label="Employee override. Hold for 3 seconds."
      />

      {/* Employee hold progress indicator (hidden until progress) */}
      {employeeHoldProgress > 0 && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-charcoal/40">
          <div className="rounded-[24px] bg-white px-6 py-4 text-center shadow-lg">
            <p className="font-sans text-body text-charcoal">Employee override</p>
            <div className="mt-2 h-2 w-48 overflow-hidden rounded-full bg-taupe">
              <div
                className="h-full rounded-full bg-moss transition-all duration-200"
                style={{ width: `${employeeHoldProgress * 100}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* Biometric auth modal */}
      <AnimatePresence>
        {showBiometric && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-50 flex items-center justify-center bg-charcoal/40"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              transition={{ duration: 0.2, ease: motionTokens.easeOutExpo }}
              className="mx-4 w-full max-w-sm rounded-[24px] bg-white p-6 text-center shadow-lg"
              role="dialog"
              aria-modal="true"
              aria-label="Biometric authentication"
            >
              <h2 className="font-heading text-[28px] font-semibold text-charcoal">
                Verify to complete
              </h2>
              <p className="mt-2 font-sans text-[16px] text-stone">
                Please use the PIN pad or present your card to the terminal.
              </p>

              {/* Animated fingerprint/card icon */}
              <motion.div
                className="mx-auto mt-4 flex h-24 w-24 items-center justify-center rounded-full bg-pale-mint"
                animate={{ scale: [1, 1.05, 1] }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
                aria-hidden="true"
              >
                <svg
                  viewBox="0 0 32 32"
                  className="h-12 w-12 text-moss"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <rect x="4" y="8" width="24" height="16" rx="3" />
                  <path d="M16 14v4" strokeLinecap="round" />
                  <circle cx="16" cy="18" r="1" fill="currentColor" stroke="none" />
                </svg>
              </motion.div>

              {/* Terminal connection status */}
              <p className="mt-3 font-mono text-caption text-moss">
                Terminal: CONNECTED
              </p>

              {/* Simulated auth buttons for demo */}
              <div className="mt-4 flex gap-3">
                <button
                  type="button"
                  onClick={() => { setShowBiometric(false); }}
                  className="h-14 flex-1 rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleBiometricComplete}
                  className="h-14 flex-1 rounded-full bg-moss text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(90,122,92,0.3)]"
                >
                  Authorize
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
