import { describe, expect, it } from "vitest";
import { useEffect } from "react";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { ItemModal } from "../routes/ItemModal";
import type { MenuItem } from "@astra/shared-types";

const mockItem: MenuItem = {
  itemId: "item-1",
  storeId: "store-1",
  categoryId: "cat-1",
  name: "Test Bowl",
  description: "A test bowl",
  priceCents: 999,
  costCents: 400,
  plu: null,
  barcode: null,
  sku: null,
  imageUrl: null,
  blurhash: null,
  taxCategory: "standard",
  isWeightBased: false,
  weightUnit: null,
  isActive: true,
  metadata: {},
  modifierGroups: [
    {
      modifierGroupId: "mg-1",
      storeId: "store-1",
      name: "Add-ons",
      description: null,
      minSelect: 0,
      maxSelect: 2,
      displayOrder: 0,
      isActive: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      deletedAt: null,
      options: [
        {
          modifierOptionId: "opt-1",
          modifierGroupId: "mg-1",
          name: "Guacamole",
          priceDeltaCents: 150,
          isDefault: false,
          displayOrder: 0,
          isActive: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          deletedAt: null,
        },
      ],
    },
  ],
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  deletedAt: null,
};

function ItemSelector({ children }: { readonly children: React.ReactNode }): React.JSX.Element {
  const { send } = useKioskMachine();
  useEffect(() => {
    send({ type: "START_SESSION", sessionId: "session-1" });
    send({ type: "SELECT_ITEM", item: mockItem });
  }, [send]);
  return <>{children}</>;
}

describe("ItemModal", () => {
  it("renders the item name, price, and modifier options", () => {
    renderWithMachine(
      <ItemSelector>
        <ItemModal />
      </ItemSelector>,
    );

    expect(screen.getByText("Test Bowl")).toBeDefined();
    expect(screen.getByText("$9.99")).toBeDefined();
    expect(screen.getByText("Guacamole")).toBeDefined();
  });

  it("closes the modal when the close button is pressed", () => {
    const StateReader = (): React.JSX.Element => {
      const { state } = useKioskMachine();
      return <div data-testid="stage">{state.value as string}</div>;
    };

    const { getByTestId, getByLabelText } = renderWithMachine(
      <ItemSelector>
        <ItemModal />
        <StateReader />
      </ItemSelector>,
    );

    fireEvent.click(getByLabelText("Close item details"));
    expect(getByTestId("stage").textContent).toBe("MENU");
  });
});
