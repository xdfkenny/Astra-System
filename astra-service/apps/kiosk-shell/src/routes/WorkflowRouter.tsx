import { AnimatePresence, motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSessionStore } from "@astra/kiosk-state";
import { AttractScreen } from "./AttractScreen";
import { MenuScreen } from "./MenuScreen";
import { CartReviewScreen } from "./CartReviewScreen";
import { PaymentAuthScreen } from "./PaymentAuthScreen";
import { ReceiptScreen } from "./ReceiptScreen";

/**
 * Central workflow router. Deliberately NOT driven by URL path — the kiosk
 * workflow is a strict finite state machine (see sessionStore.ALLOWED_TRANSITIONS)
 * and using React Router's declarative <Route> tree here would let a stray
 * deep-link or back-button gesture desync the UI from actual session state
 * (e.g. show Payment for a cart that was already reset). We render purely
 * off `stage` and reserve React Router for the Admin micro-frontend, which
 * IS a conventional multi-page app.
 */
export function WorkflowRouter(): React.JSX.Element {
  const stage = useSessionStore((s) => s.stage);

  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={stage}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: motionTokens.durationFast, ease: motionTokens.easeStandard }}
        className="flex flex-1 flex-col overflow-hidden"
      >
        {stage === "attract" && <AttractScreen />}
        {stage === "menu" && <MenuScreen />}
        {stage === "cart_review" && <CartReviewScreen />}
        {stage === "payment_auth" && <PaymentAuthScreen />}
        {stage === "receipt" && <ReceiptScreen />}
      </motion.div>
    </AnimatePresence>
  );
}
