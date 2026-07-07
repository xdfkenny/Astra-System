import type { CartTotals, ReadonlyCartLineItem } from "@astra/shared-types";

/**
 * Pure cart total calculation — deliberately framework-free and side-effect
 * free so it can run identically on the main thread (small carts) or inside
 * `totals.worker.ts` (large carts / heavy modifier trees) without
 * duplication. Tax/environmental-fee rates would be fetched per-jurisdiction
 * from the Go inventory/order service in production; hardcoded here for a
 * single reference jurisdiction (FL, US) to keep this a real, runnable
 * implementation rather than a stub.
 */
const FLORIDA_SALES_TAX_RATE = 0.07; // Miami-Dade combined state+county rate
const ENVIRONMENTAL_FEE_PER_ITEM_CENTS = 5; // e.g. bag/container fee, itemized transparently
const LOYALTY_DISCOUNT_RATE = 0.05; // applied when a loyalty profile is attached (stubless: always example-computed)

export interface ComputeTotalsOptions {
  readonly hasLoyaltyAccount: boolean;
}

export function computeCartTotals(
  lines: ReadonlyArray<ReadonlyCartLineItem>,
  options: ComputeTotalsOptions,
): CartTotals {
  const subtotalCents = lines.reduce((sum, line) => {
    const modifiersTotal = line.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0);
    return sum + line.quantity * (line.unitPriceCentsSnapshot + modifiersTotal);
  }, 0);

  const itemCount = lines.reduce((sum, line) => sum + line.quantity, 0);
  const environmentalFeeCents = itemCount * ENVIRONMENTAL_FEE_PER_ITEM_CENTS;

  const loyaltyDiscountCents = options.hasLoyaltyAccount
    ? Math.round(subtotalCents * LOYALTY_DISCOUNT_RATE)
    : 0;

  const taxableBase = subtotalCents - loyaltyDiscountCents;
  const taxCents = Math.round(taxableBase * FLORIDA_SALES_TAX_RATE);

  const discountCents = loyaltyDiscountCents; // reserved for future promo-code stacking

  const totalCents = subtotalCents - loyaltyDiscountCents + taxCents + environmentalFeeCents;

  const breakdown: Array<{ label: string; amountCents: number; kind: "tax" | "discount" | "fee" | "loyalty" }> = [
    { label: "Subtotal", amountCents: subtotalCents, kind: "fee" },
  ];
  if (loyaltyDiscountCents > 0) {
    breakdown.push({
      label: "Loyalty discount (5%)",
      amountCents: -loyaltyDiscountCents,
      kind: "loyalty",
    });
  }
  breakdown.push({ label: "Sales tax (Miami-Dade, 7%)", amountCents: taxCents, kind: "tax" });
  if (environmentalFeeCents > 0) {
    breakdown.push({
      label: "Environmental fee",
      amountCents: environmentalFeeCents,
      kind: "fee",
    });
  }

  return {
    subtotalCents,
    discountCents,
    taxCents,
    environmentalFeeCents,
    loyaltyDiscountCents,
    totalCents,
    breakdown,
  };
}
