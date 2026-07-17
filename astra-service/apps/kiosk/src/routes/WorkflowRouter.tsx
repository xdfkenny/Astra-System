import { lazy, Suspense, useMemo } from "react";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { LanguageSelectScreen } from "./LanguageSelectScreen";
import { AttractScreen } from "./AttractScreen";
import { MenuScreen } from "./MenuScreen";
import { CartReviewScreen } from "./CartReviewScreen";
import { PaymentAuthScreen } from "./PaymentAuthScreen";
import { ProcessingScreen } from "./ProcessingScreen";
import { ReceiptScreen } from "./ReceiptScreen";
import { AdminScreen } from "./AdminScreen";
import { ItemModal } from "./ItemModal";

const DEV_MODE = import.meta.env.DEV || (import.meta.env as Record<string, string | undefined>)["VITE_ASTRA_DEV_MODE"] === "true";

export function WorkflowRouter(): React.JSX.Element {
  const { state, send } = useKioskMachine();
  const stage = state.value as string;
  const reduceMotion = useReducedMotion();

  const useRemote = !DEV_MODE;

  const baseScreen = useMemo(() => {
    if (stage === "LANGUAGE_SELECT") return <LanguageSelectScreen />;
    if (stage === "ATTRACT") return <AttractScreen />;
    if (stage === "ADMIN") return <AdminScreen />;

    if (stage === "MENU" || stage === "ITEM_DETAIL") {
      if (useRemote) {
        const MenuApp = lazy(() =>
          import("astra_menu/MenuApp").then((m) => ({
            default: () => {
              const C = m.default;
              return (
                <C
                  laneMode={state.context.laneMode}
                  silentAssistArmed={false}
                  onSelectItem={(item) => { send({ type: "SELECT_ITEM", item }); }}
                />
              );
            },
          })),
        );
        return (
          <Suspense fallback={<div className="flex flex-1 items-center justify-center bg-linen" />}>
            <MenuApp />
          </Suspense>
        );
      }
      return <MenuScreen />;
    }

    if (stage === "CART") {
      if (useRemote) {
        const CartApp = lazy(() =>
          import("astra_cart/CartApp").then((m) => ({
            default: () => {
              const C = m.default;
              return (
                <C
                  onBackToMenu={() => { send({ type: "BACK_TO_MENU" }); }}
                  onProceedToPayment={() => { send({ type: "PROCEED_TO_PAYMENT" }); }}
                />
              );
            },
          })),
        );
        return (
          <Suspense fallback={<div className="flex flex-1 items-center justify-center bg-linen" />}>
            <CartApp />
          </Suspense>
        );
      }
      return <CartReviewScreen />;
    }

    if (stage === "PAYMENT") return <PaymentAuthScreen />;
    if (stage === "PROCESSING") return <ProcessingScreen />;
    if (stage === "RECEIPT") return <ReceiptScreen />;

    return <AttractScreen />;
  }, [stage, useRemote, state.context.laneMode, send]);

  return (
    <>
      <AnimatePresence mode="wait" initial={false}>
        <motion.div
          key={stage}
          className="flex h-full w-full flex-1"
          initial={{ opacity: reduceMotion ? 1 : 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: reduceMotion ? 1 : 0 }}
          transition={{ duration: reduceMotion ? 0 : 0.25, ease: "easeOut" }}
        >
          {baseScreen}
        </motion.div>
      </AnimatePresence>
      {stage === "ITEM_DETAIL" && <ItemModal />}
    </>
  );
}

