import { describe, expect, it, vi } from "vitest";
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
  it("starts in ATTRACT and transitions to MENU on START_SESSION", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    expect(actor.getSnapshot().value).toBe("ATTRACT");

    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("MENU");
    expect(snapshot.context.sessionId).toBe("session-1");
  });

  it("opens the item detail on SELECT_ITEM", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "SELECT_ITEM", item: mockItem });

    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("ITEM_DETAIL");
    expect(snapshot.context.selectedItem).toEqual(mockItem);
  });

  it("returns to MENU and marks cart as having items on ADD_TO_CART", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "SELECT_ITEM", item: mockItem });
    actor.send({ type: "ADD_TO_CART" });

    const snapshot = actor.getSnapshot();
    expect(snapshot.value).toBe("MENU");
    expect(snapshot.context.cartHasItems).toBe(true);
  });

  it("transitions from MENU to CART on GO_TO_CART when cart has items", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });

    expect(actor.getSnapshot().value).toBe("CART");
  });

  it("blocks GO_TO_CART when the cart is empty", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "GO_TO_CART" });

    expect(actor.getSnapshot().value).toBe("MENU");
  });

  it("moves through payment authorization to processing and receipt", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });
    actor.send({ type: "PROCEED_TO_PAYMENT" });

    expect(actor.getSnapshot().value).toBe("PAYMENT");

    const result: PaymentAuthorizationResult = {
      authorizationId: "auth-1",
      status: "authorized",
      method: "credit_debit",
      amountCents: 899,
    };
    actor.send({ type: "PAYMENT_AUTHORIZED", result });

    expect(actor.getSnapshot().value).toBe("PROCESSING");
  });

  it("returns to CART on PAYMENT_DECLINED", () => {
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
    expect(snapshot.value).toBe("CART");
    expect(snapshot.context.errorMessage).toBe("Insufficient funds");
  });

  it("transitions from RECEIPT to ATTRACT on RECEIPT_ACKNOWLEDGED", async () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "START_SESSION", sessionId: "session-1" });
    actor.send({ type: "CART_UPDATED", cartHasItems: true });
    actor.send({ type: "GO_TO_CART" });
    actor.send({ type: "PROCEED_TO_PAYMENT" });

    const result: PaymentAuthorizationResult = {
      authorizationId: "auth-3",
      status: "authorized",
      method: "credit_debit",
      amountCents: 899,
    };
    actor.send({ type: "PAYMENT_AUTHORIZED", result });

    await vi.waitFor(() => {
      expect(actor.getSnapshot().value).toBe("RECEIPT");
    }, { timeout: 2000 });

    actor.send({ type: "RECEIPT_ACKNOWLEDGED" });
    expect(actor.getSnapshot().value).toBe("ATTRACT");
    expect(actor.getSnapshot().context.sessionId).toBeNull();
  });

  it("transitions to ADMIN on OPEN_ADMIN and back on CLOSE_ADMIN", () => {
    const actor = createActor(kioskMachine);
    actor.start();
    actor.send({ type: "OPEN_ADMIN" });

    expect(actor.getSnapshot().value).toBe("ADMIN");

    actor.send({ type: "CLOSE_ADMIN" });
    expect(actor.getSnapshot().value).toBe("ATTRACT");
  });
});
