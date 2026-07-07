import { describe, expect, it } from "vitest";
import { useEffect } from "react";
import { screen } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { CartSummary } from "./CartSummary";
import { addLineItem } from "@astra/kiosk-state";

function SessionStarter({ children }: { readonly children: React.ReactNode }): React.JSX.Element {
  const { send } = useKioskMachine();
  useEffect(() => {
    send({ type: "START_SESSION", sessionId: "session-1" });
  }, [send]);
  return <>{children}</>;
}

describe("CartSummary", () => {
  it("shows the item count and total when items are in the cart", () => {
    addLineItem({
      menuItemId: "item-1",
      nameSnapshot: "Test Burrito",
      unitPriceCentsSnapshot: 899,
      quantity: 2,
      modifiers: [],
    });

    renderWithMachine(
      <SessionStarter>
        <CartSummary />
      </SessionStarter>,
    );

    expect(screen.getByText("2")).toBeDefined();
    expect(screen.getByText("$17.98")).toBeDefined();
  });
});
