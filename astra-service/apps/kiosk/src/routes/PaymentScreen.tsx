// Payment screen with method selection and biometric auth.
// Horizontal scrollable payment cards.
// Biometric auth modal appears on confirm.
import { useState } from "react";
import { motion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { PrimaryButton } from "../components/PrimaryButton";
import { SecondaryButton } from "../components/SecondaryButton";
import { BiometricAuthPrompt } from "../components/BiometricAuthPrompt";
import { cn } from "@/utils/cn";
import type { PaymentMethod } from "@astra/shared-types";

interface PaymentMethodOption {
  readonly id: PaymentMethod;
  readonly name: string;
  readonly icon: string;
  readonly description: string;
  selected: boolean;
}

const PAYMENT_METHODS: PaymentMethodOption[] = [
  {
    id: "credit_debit",
    name: "Card / NFC",
    icon: "💳",
    description: "Visa, Mastercard, Amex",
    selected: true,
  },
  {
    id: "cash_recycler",
    name: "Cash",
    icon: "💵",
    description: "Exact change accepted",
    selected: false,
  },
  {
    id: "qr_code",
    name: "QR Code",
    icon: "📱",
    description: "Scan to pay",
    selected: false,
  },
];

export function PaymentScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  // PAYMENT_METHODS is a non-empty literal array, so the first element is always defined.
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const [selectedMethod, setSelectedMethod] = useState<PaymentMethodOption>(PAYMENT_METHODS[0]!);
  const [showBiometricAuth, setShowBiometricAuth] = useState(false);
  const [isProcessingTerminal, setIsProcessingTerminal] = useState(false);

  const handleMethodSelect = (method: PaymentMethodOption) => {
    setSelectedMethod(method);
    PAYMENT_METHODS.forEach((m) => {
      m.selected = m.id === method.id;
    });
  };

  const handleConfirmPayment = () => {
    setIsProcessingTerminal(true);
    setTimeout(() => {
      setShowBiometricAuth(true);
    }, 1000);
  };

  const handleBiometricAuth = () => {
    setShowBiometricAuth(false);
    setIsProcessingTerminal(true);
    setTimeout(() => {
      send({
        type: "PAYMENT_AUTHORIZED",
        result: {
          authorizationId: "auth-terminal-123",
          status: "authorized",
          method: selectedMethod.id,
          amountCents: 1299,
        },
      });
    }, 2000);
  };

  const handleCancel = () => {
    send({ type: "CANCEL_PAYMENT" });
  };

  const handleBiometricAuthClose = () => {
    setShowBiometricAuth(false);
    setIsProcessingTerminal(false);
    handleCancel();
  };

  const handleEmployeeOverride = () => {
    send({ type: "OPEN_ADMIN" });
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-linen safe-top safe-bottom">
      <div className="flex-shrink-0 p-4 pb-3">
        <h1 className="font-heading text-[28px] font-semibold text-charcoal">
          Ready to pay
        </h1>

        <div className="mt-4 rounded-[16px] bg-white/85 backdrop-blur-[8px] border border-taupe/20 p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="font-sans text-[14px] uppercase tracking-wider text-stone">
              Cart summary
            </span>
            <button
              type="button"
              className="font-sans text-[14px] font-medium text-moss hover:text-moss/80"
              onClick={() => { send({ type: "GO_TO_CART" }); }}
            >
              Edit
            </button>
          </div>

          <div className="space-y-2">
            <div className="flex justify-between text-[16px] text-charcoal">
              <span>2 items</span>
              <span>$24.50</span>
            </div>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-4 pb-24">
        <h2 className="font-sans text-[14px] uppercase tracking-wider text-stone mb-3">
          Payment method
        </h2>

        <div className="flex gap-3 overflow-x-auto pb-4 no-scrollbar">
          {PAYMENT_METHODS.map((method) => (
            <motion.button
              key={method.id}
              type="button"
              className={cn(
                "flex-shrink-0 w-32 rounded-[16px] p-4 text-center transition-all duration-200",
                "border-2",
                selectedMethod.id === method.id
                  ? "border-moss bg-pale-mint/30"
                  : "border-taupe bg-white/50 hover:border-moss/40 hover:bg-moss/5"
              )}
              onClick={() => { handleMethodSelect(method); }}
              whileTap={{ scale: 0.95 }}
            >
              <div className="text-3xl mb-2">{method.icon}</div>
              <h3 className="font-sans text-[16px] font-medium text-charcoal mb-1">
                {method.name}
              </h3>
              <p className="font-sans text-[12px] text-stone">
                {method.description}
              </p>
              {selectedMethod.id === method.id && (
                <motion.div
                  layoutId="selected-payment"
                  className="h-1 w-8 rounded-full bg-moss mx-auto mt-2"
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                />
              )}
            </motion.button>
          ))}
        </div>

        <div className="mt-6">
          <h3 className="font-sans text-[14px] uppercase tracking-wider text-stone mb-3">
            Payment terminal
          </h3>

          <div className="rounded-[16px] bg-white/85 backdrop-blur-[8px] border border-taupe/20 p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-full bg-denim/10 flex items-center justify-center">
                  <span className="text-lg">🅿</span>
                </div>
                <div>
                  <h4 className="font-sans text-[16px] font-medium text-charcoal">
                    Verifone Terminal
                  </h4>
                  <p className="font-sans text-[12px] text-stone">
                    Terminal: KIOSK-3
                  </p>
                </div>
              </div>
              <div
                className={cn(
                  "h-3 w-3 rounded-full",
                  isProcessingTerminal ? "bg-amber animate-pulse" : "bg-moss"
                )}
              />
            </div>

            {isProcessingTerminal && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="rounded-[12px] bg-amber/10 border border-amber/20 p-4"
              >
                <p className="font-sans text-[14px] text-amber text-center">
                  Connecting to payment terminal...
                </p>
              </motion.div>
            )}
          </div>
        </div>

        <div className="h-20" />
      </div>

      <div className="absolute bottom-0 left-0 right-0 z-20 bg-linen border-t border-taupe/20 px-4 py-4 safe-bottom">
        <div className="mx-auto max-w-md space-y-3">
          <PrimaryButton
            onClick={handleConfirmPayment}
            className="w-full"
            disabled={isProcessingTerminal}
            isLoading={isProcessingTerminal}
            loadingText="Connecting terminal..."
          >
            Confirm Payment
          </PrimaryButton>

          <SecondaryButton
            onClick={handleCancel}
            className="w-full"
            variant="ghost"
          >
            Cancel
          </SecondaryButton>

          <div className="flex items-center justify-center">
            <button
              type="button"
              className="group flex items-center gap-1 font-sans text-[12px] text-stone/60 hover:text-stone"
              onClick={handleEmployeeOverride}
            >
              <span className="h-1 w-8 rounded-full bg-stone/20 group-hover:bg-stone/40" />
              Hold for 3 seconds (staff only)
            </button>
          </div>
        </div>
      </div>

      <BiometricAuthPrompt
        isOpen={showBiometricAuth}
        onAuthenticate={handleBiometricAuth}
        onCancel={handleBiometricAuthClose}
      />
    </div>
  );
}
