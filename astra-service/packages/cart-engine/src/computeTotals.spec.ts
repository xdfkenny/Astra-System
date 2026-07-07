import { describe, expect, it } from "vitest";
import { computeCartTotals } from "./computeTotals";
import type { CartLineItem } from "@astra/shared-types";

const baseLine: CartLineItem = {
  lineId: "line-1",
  menuItemId: "item-1",
  nameSnapshot: "Test Burrito",
  unitPriceCentsSnapshot: 899,
  quantity: 1,
  modifiers: [],
  addedAtMs: Date.now(),
};

describe("computeCartTotals", () => {
  it("calculates subtotal, tax, environmental fee, and total for a single item", () => {
    const totals = computeCartTotals([baseLine], { hasLoyaltyAccount: false });
    expect(totals.subtotalCents).toBe(899);
    expect(totals.taxCents).toBe(Math.round(899 * 0.07));
    expect(totals.environmentalFeeCents).toBe(5);
    expect(totals.totalCents).toBe(899 + Math.round(899 * 0.07) + 5);
  });

  it("applies loyalty discount when account is attached", () => {
    const totals = computeCartTotals([baseLine], { hasLoyaltyAccount: true });
    const expectedDiscount = Math.round(899 * 0.05);
    expect(totals.loyaltyDiscountCents).toBe(expectedDiscount);
    expect(totals.discountCents).toBe(expectedDiscount);
    expect(totals.taxCents).toBe(Math.round((899 - expectedDiscount) * 0.07));
  });

  it("sums modifiers into the line price", () => {
    const line: CartLineItem = {
      ...baseLine,
      modifiers: [
        { modifierId: "mg-1", optionId: "opt-1", priceDeltaCents: 100 },
        { modifierId: "mg-1", optionId: "opt-2", priceDeltaCents: 150 },
      ],
    };
    const totals = computeCartTotals([line], { hasLoyaltyAccount: false });
    expect(totals.subtotalCents).toBe(899 + 100 + 150);
  });

  it("multiplies line totals by quantity", () => {
    const line: CartLineItem = { ...baseLine, quantity: 3 };
    const totals = computeCartTotals([line], { hasLoyaltyAccount: false });
    expect(totals.subtotalCents).toBe(899 * 3);
    expect(totals.environmentalFeeCents).toBe(5 * 3);
  });

  it("returns zero totals for an empty cart", () => {
    const totals = computeCartTotals([], { hasLoyaltyAccount: false });
    expect(totals.subtotalCents).toBe(0);
    expect(totals.taxCents).toBe(0);
    expect(totals.totalCents).toBe(0);
  });
});
