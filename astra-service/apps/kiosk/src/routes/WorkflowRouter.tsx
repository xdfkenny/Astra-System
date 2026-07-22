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

// Remote micro-frontends — lazily loaded only when enabled via env var.
// Defined at module level (not inside useMemo) so React doesn't recreate
// them on every render, which would cause remounting.
const RemoteMenuApp = lazy(() =>
  import("astra_menu/MenuApp").catch(() => ({
    default: MenuScreen,
  })),
);

const RemoteCartApp = lazy(() =>
  import("astra_cart/CartApp").catch(() => ({
    default: CartReviewScreen,
  })),
);

const useRemote =
  (import.meta.env as Record<string, string | undefined>)["VITE_ENABLE_REMOTES"] === "true";

export function WorkflowRouter(): React.JSX.Element {
  const { state, send } = useKioskMachine();
  const stage = String(state.value);
  const reduceMotion = useReducedMotion();

  const baseScreen = useMemo(() => {
    if (state.matches("LANGUAGE_SELECT")) return <LanguageSelectScreen />;
    if (state.matches("ATTRACT")) return <AttractScreen />;
    if (state.matches("ADMIN")) return <AdminScreen />;

    if (state.matches("MENU") || state.matches("ITEM_DETAIL")) {
      if (useRemote) {
        return (
          <Suspense fallback={<div className="flex flex-1 items-center justify-center bg-linen" />}>
            <RemoteMenuApp
              laneMode={state.context.laneMode}
              silentAssistArmed={false}
              onSelectItem={(item) => { send({ type: "SELECT_ITEM", item }); }}
            />
          </Suspense>
        );
      }
      return <MenuScreen />;
    }

    if (state.matches("CART")) {
      if (useRemote) {
        return (
          <Suspense fallback={<div className="flex flex-1 items-center justify-center bg-linen" />}>
            <RemoteCartApp
              onBackToMenu={() => { send({ type: "BACK_TO_MENU" }); }}
              onProceedToPayment={() => { send({ type: "PROCEED_TO_PAYMENT" }); }}
            />
          </Suspense>
        );
      }
      return <CartReviewScreen />;
    }

    if (state.matches("PAYMENT")) return <PaymentAuthScreen />;
    if (state.matches("PROCESSING")) return <ProcessingScreen />;
    if (state.matches("RECEIPT")) return <ReceiptScreen />;

    return <AttractScreen />;
  }, [state, send]);

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
      {state.matches("ITEM_DETAIL") && <ItemModal />}
    </>
  );
}
