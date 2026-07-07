import { describe, expect, it } from "vitest";
import { createActor } from "xstate";
import { kioskMachine } from "./kioskMachine";
import type { MenuItem, PaymentAuthorizationResult } from "@astra/shared-types";

const mockItem: MenuItem = {
  itemId: "item-1",
  storeId: "store-1",
  categoryId: "cat-1",
  name: "Test Burrito",
  description: "A test burrito",
  priceCents: 899,
  costCents: 300,
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
      name: "Protein",
      description: null,
      minSelect: 1,
      maxSelect: 1,
      displayOrder: 0,
      isActive: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      deletedAt: null,
      options: [
        {
          modifierOptionId: "opt-1",
          modifierGroupId: "mg-1",
          name: "Chicken",
          priceDeltaCents: 0,
          isDefault: true,
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

describe("kioskMachine", () => {
  it("starts in ATTRACT and transitions to MENU_BROWSE on START_SESSION", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    expect(actor.getSnapshot().value).toBe("ATTRACT");

    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("MENU_BROWSE");
    expect(snapshot.context.sessionId).toBe("session-1");
  });

  it("opens the item modal on SELECT_ITEM", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "SELECT_ITEM", item: mockItem });

    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("ITEM_MODAL");
    expect(snapshot.context.selectedItem).toEqual(mockItem);
  });

  it("returns to MENU_BROWSE and marks cart as having items on ADD_TO_CART", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "SELECT_ITEM", item: mockItem });
    actor.send({ type: "ADD_TO_CART" });

    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("MENU_BROWSE");
    expect(snapshot.context.cartHasItems).toBe(true);
  });

  it("transitions from MENU_BROWSE to CART_REVIEW on GO_TO_CART when cart has items", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });

    expect(actor.getSnapshot().value).toBe("CART_REVIEW");
  });

  it("blocks GO_TO_CART when the cart is empty", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "GO_TO_CART" });

    expect(actor.getSnapshot().value).toBe("MENU_BROWSE");
  });

  it("moves through payment authorization to processing and receipt", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });
    actor.send({ type: "PROCEED_TO_PAYMENT" });

    expect(actor.getSnapshot().value).toBe("PAYMENT_AUTH");

    const result: PaymentAuthorizationResult = {
      authorizationId: "auth-1",
      status: "authorized",
      method: "credit_debit",
      amountCents: 899,
    };
    actor.send({ type: "PAYMENT_AUTHORIZED", result });

    expect(actor.getSnapshot().value).toBe("PROCESSING");
  });

  it("returns to CART_REVIEW on PAYMENT_DECLINED", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });
    actor.send({ type: "PROCEED_TO_PAYMENT" });

    const result: PaymentAuthorizationResult = {
      authorizationId: "auth-2",
      status: "declined",
      method: "credit_debit",
      amountCents: 899,
      declineReason: "Insufficient funds",
    };
    actor.send({ type: "PAYMENT_DECLINED", result });

    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("CART_REVIEW");
    expect(snapshot.context.errorMessage).toBe("Insufficient funds");
  });

  it("enters IDLE_TIMEOUT from MENU_BROWSE and resumes on CONTINUE_SESSION", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "IDLE_TIMEOUT" });

    expect(actor.getSnapshot().value).toBe("IDLE_TIMEOUT");

    actor.send({ type: "CONTINUE_SESSION" });
    expect(actor.getSnapshot().value).toBe("MENU_BROWSE");
  });

  it("resets to ATTRACT from IDLE_TIMEOUT on RESET_SESSION", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "IDLE_TIMEOUT" });
    actor.send({ type: "RESET_SESSION" });

    expect(actor.getSnapshot().value).toBe("ATTRACT");
    expect(actor.getSnapshot().context.sessionId).toBeNull();
  });
});
