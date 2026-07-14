import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { CartSummary } from "./CartSummary";
import { addLineItem } from "@astra/kiosk-state";

describe("CartSummary", () => {
  it("shows the item count and total when items are in the cart", () => {
    addLineItem({
      menuItemId: "item-1",
      nameSnapshot: "Test Burrito",
      unitPriceCentsSnapshot: 899,
      quantity: 2,
      modifiers: [],
    });

    renderWithMachine(<CartSummary />);

    expect(screen.getByText(/2 items/)).toBeDefined();
    expect(screen.getByText(/\$17\.98/)).toBeDefined();
  });
});

