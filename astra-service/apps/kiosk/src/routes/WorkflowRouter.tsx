import { AnimatePresence, motion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { AttractScreen } from "./AttractScreen";
import { MenuScreen } from "./MenuScreen";
import { ItemModal } from "./ItemModal";
import { CartReviewScreen } from "./CartReviewScreen";
import { PaymentAuthScreen } from "./PaymentAuthScreen";
import { ProcessingScreen } from "./ProcessingScreen";
import { ReceiptScreen } from "./ReceiptScreen";
import { IdleTimeoutOverlay } from "../components/IdleTimeoutOverlay";
import { motion as motionTokens } from "@astra/design-tokens";

/**
 * Central workflow router. Renders purely off the XState machine's stage value
 * so the UI can never desync from the authoritative workflow state.
 */
export function WorkflowRouter(): React.JSX.Element {
  const { state } = useKioskMachine();
  const stage = state.value as string;

  const baseScreen =
    stage === "ATTRACT" ? (
      <AttractScreen />
    ) : stage === "MENU_BROWSE" || stage === "ITEM_MODAL" || stage === "IDLE_TIMEOUT" ? (
      <MenuScreen />
    ) : stage === "CART_REVIEW" ? (
      <CartReviewScreen />
    ) : stage === "PAYMENT_AUTH" ? (
      <PaymentAuthScreen />
    ) : stage === "PROCESSING" ? (
      <ProcessingScreen />
    ) : stage === "RECEIPT" || stage === "RESET" ? (
      <ReceiptScreen />
    ) : (
      <AttractScreen />
    );

  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={stage}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{
          duration: motionTokens.durationFast,
          ease: motionTokens.easeStandard,
        }}
        className="relative flex flex-1 flex-col overflow-hidden"
      >
        {baseScreen}
        {stage === "ITEM_MODAL" && <ItemModal />}
        {stage === "IDLE_TIMEOUT" && <IdleTimeoutOverlay />}
      </motion.div>
    </AnimatePresence>
  );
}
