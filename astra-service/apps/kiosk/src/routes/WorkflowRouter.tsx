import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { AttractScreen } from "./AttractScreen";
import { MenuScreen } from "./MenuScreen";
import { ItemModal } from "./ItemModal";
import { CartReviewScreen } from "./CartReviewScreen";
import { PaymentAuthScreen } from "./PaymentAuthScreen";
import { ProcessingScreen } from "./ProcessingScreen";
import { ReceiptScreen } from "./ReceiptScreen";

export function WorkflowRouter(): React.JSX.Element {
  const { state } = useKioskMachine();
  const stage = state.value as string;
  const reducedMotion = useReducedMotion();

  // ITEM_DETAIL renders the MenuScreen with an overlay modal, so it shares the
  // MENU base to avoid remounting the menu list on modal open/close.
  const baseStage = stage === "ITEM_DETAIL" ? "MENU" : stage;

  const baseScreen =
    baseStage === "ATTRACT" ? (
      <AttractScreen />
    ) : baseStage === "MENU" ? (
      <MenuScreen />
    ) : baseStage === "CART" ? (
      <CartReviewScreen />
    ) : baseStage === "PAYMENT" ? (
      <PaymentAuthScreen />
    ) : baseStage === "PROCESSING" ? (
      <ProcessingScreen />
    ) : baseStage === "RECEIPT" ? (
      <ReceiptScreen />
    ) : (
      <AttractScreen />
    );

  return (
    <>
      <AnimatePresence mode="wait">
        <motion.div
          key={baseStage}
          className="flex flex-1 flex-col"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{
            duration: reducedMotion ? 0.05 : 0.3,
            ease: motionTokens.easeOutExpo,
          }}
        >
          {baseScreen}
        </motion.div>
      </AnimatePresence>
      {stage === "ITEM_DETAIL" && <ItemModal />}
    </>
  );
}
