import type { Hlc } from "../hlc";
import type {
  Category,
  Item,
  ItemTaxCategory,
  ModifierGroup,
  ModifierOption,
  Order,
  PaymentMethod,
  PaymentStatus,
  WeightUnit,
} from "./domain";

// -----------------------------------------------------------------------------
// Runtime line item used by kiosk UI and cart-engine.
// -----------------------------------------------------------------------------

export interface ModifierSelection {
  readonly modifierId: string;
  readonly optionId: string;
  readonly priceDeltaCents: number;
}

export interface CartLineItem {
  lineId: string;
  menuItemId: string;
  nameSnapshot: string;
  unitPriceCentsSnapshot: number;
  quantity: number;
  modifiers: ModifierSelection[];
  notes?: string;
  weightGrams?: number;
  addedAtMs: number;
}

export type DeepReadonly<T> = {
  readonly [K in keyof T]: DeepReadonly<T[K]>;
};

export type ReadonlyCartLineItem = DeepReadonly<CartLineItem>;

export interface CartTotals {
  readonly subtotalCents: number;
  readonly discountCents: number;
  readonly taxCents: number;
  readonly environmentalFeeCents: number;
  readonly loyaltyDiscountCents: number;
  readonly totalCents: number;
  readonly breakdown: readonly {
    readonly label: string;
    readonly amountCents: number;
    readonly kind: "tax" | "discount" | "fee" | "loyalty";
  }[];
}

// -----------------------------------------------------------------------------
// Runtime cart state used by kiosk UI and CRDT merge layer.
// -----------------------------------------------------------------------------

export interface CartState {
  cartId: string;
  kioskId: string;
  sessionId: string;
  storeId?: string;
  lines: CartLineItem[];
  version: number;
  currency: string;
  createdAtMs: number;
  updatedAtMs: number;
}

// -----------------------------------------------------------------------------
// Kiosk session and ghost cart
// -----------------------------------------------------------------------------

export interface KioskSession {
  readonly sessionId: string;
  readonly kioskId: string;
  readonly storeId: string;
  readonly laneId?: string;
  readonly startedAtMs: number;
  readonly lastActivityAtMs: number;
  readonly employeeId?: string;
  readonly customerPhone?: string;
}

export interface GhostCart {
  readonly ghostCartId: string;
  readonly ownerKioskId: string;
  readonly ownerSessionId: string;
  readonly lines: readonly CartLineItem[];
  readonly hlc: Hlc;
  readonly expiresAtMs: number;
}

// -----------------------------------------------------------------------------
// Menu response shape used by kiosk UI.
// -----------------------------------------------------------------------------

export type MenuModifierGroup = ModifierGroup & {
  readonly options: readonly ModifierOption[];
};

export type MenuItem = Item & {
  readonly modifierGroups: readonly MenuModifierGroup[];
  readonly category?: Category;
};

// -----------------------------------------------------------------------------
// Payment intent and result used at the kiosk terminal.
// -----------------------------------------------------------------------------

export interface PaymentIntent {
  readonly intentId: string;
  readonly cartId: string;
  readonly orderId?: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly method: PaymentMethod;
  readonly status: PaymentStatus;
  readonly createdAtMs: number;
}

export interface PaymentAuthorizationResult {
  readonly authorizationId: string;
  readonly status: PaymentStatus;
  readonly method: PaymentMethod;
  readonly amountCents: number;
  readonly approvalCode?: string;
  readonly declineReason?: string;
  readonly cardBrand?: string;
  readonly cardLastFour?: string;
  readonly receiptText?: string;
}

export interface OfflinePaymentToken {
  readonly tokenId: string;
  readonly kioskId: string;
  readonly cartId: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly method: PaymentMethod;
  readonly verifoneOpaqueToken: string;
  readonly createdAtMs: number;
  readonly expiresAtMs: number;
  readonly hmacSignature: string;
  readonly synced: boolean;
}

// -----------------------------------------------------------------------------
// Receipt and lane queue.
// -----------------------------------------------------------------------------

export interface Receipt {
  readonly orderId: string;
  readonly orderNumber: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly items: readonly {
    readonly name: string;
    readonly quantity: number;
    readonly unitPriceCents: number;
    readonly lineTotalCents: number;
  }[];
  readonly totals: CartTotals;
  readonly payment: PaymentAuthorizationResult;
  readonly printedAt: string;
}

export interface LaneQueueEstimate {
  readonly laneId: string;
  readonly queueLength: number;
  readonly estimatedWaitSeconds: number;
  readonly mode: "express" | "full";
  readonly sampledAtMs: number;
}

// -----------------------------------------------------------------------------
// Analytics dwell events captured by the kiosk UI shell.
// -----------------------------------------------------------------------------

export interface DwellEvent {
  readonly sessionId: string;
  readonly kioskId: string;
  readonly storeId: string;
  readonly screen: string;
  readonly itemId?: string;
  readonly categoryId?: string;
  readonly durationMs: number;
  readonly stalled: boolean;
  readonly createdAtMs: number;
}

// -----------------------------------------------------------------------------
// Re-export domain types frequently used together with kiosk types.
// -----------------------------------------------------------------------------

export type {
  Category,
  Item,
  ItemTaxCategory,
  ModifierGroup,
  ModifierOption,
  Order,
  PaymentMethod,
  PaymentStatus,
  WeightUnit,
};
