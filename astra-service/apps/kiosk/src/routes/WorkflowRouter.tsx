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

  const baseScreen =
    stage === "ATTRACT" ? (
      <AttractScreen />
    ) : stage === "MENU" || stage === "ITEM_DETAIL" ? (
      <MenuScreen />
    ) : stage === "CART" ? (
      <CartReviewScreen />
    ) : stage === "PAYMENT" ? (
      <PaymentAuthScreen />
    ) : stage === "PROCESSING" ? (
      <ProcessingScreen />
    ) : stage === "RECEIPT" ? (
      <ReceiptScreen />
    ) : (
      <AttractScreen />
    );

  return (
    <>
      {baseScreen}
      {stage === "ITEM_DETAIL" && <ItemModal />}
    </>
  );
}
