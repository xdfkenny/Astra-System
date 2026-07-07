import { z } from "zod";
import { UuidSchema } from "../ids";
import type { Category, CartTotals, MenuItem, PaymentAuthorizationResult } from "../types/kiosk";
import {
  CartLineSchema,
  CurrencySchema,
  IsoTimestampSchema,
  OrderSchema,
  PaymentMethodSchema,
  PaymentStatusSchema,
} from "./zod";

// -----------------------------------------------------------------------------
// Cart API
// -----------------------------------------------------------------------------

export const CreateCartRequestSchema = z.object({
  storeId: UuidSchema,
  kioskId: UuidSchema,
  sessionId: UuidSchema,
  customerPhone: z.string().max(16).optional(),
});

export type CreateCartRequest = z.infer<typeof CreateCartRequestSchema>;

export const AddItemRequestSchema = z.object({
  cartId: UuidSchema,
  menuItemId: UuidSchema,
  nameSnapshot: z.string().min(1).max(255),
  unitPriceCentsSnapshot: z.number().int().nonnegative(),
  quantity: z.number().int().positive().max(99),
  modifiers: z
    .array(
      z.object({
        modifierId: UuidSchema,
        optionId: UuidSchema,
        priceDeltaCents: z.number().int(),
      }),
    )
    .readonly()
    .default([]),
  notes: z.string().max(280).optional(),
  weightGrams: z.number().nonnegative().optional(),
});

export type AddItemRequest = z.infer<typeof AddItemRequestSchema>;

export const UpdateCartRequestSchema = z.object({
  cartId: UuidSchema,
  sessionId: UuidSchema.optional(),
  customerPhone: z.string().max(16).optional(),
  lines: z.array(CartLineSchema).readonly().optional(),
  version: z.number().int().nonnegative().optional(),
});

export type UpdateCartRequest = z.infer<typeof UpdateCartRequestSchema>;

export const CheckoutRequestSchema = z.object({
  cartId: UuidSchema,
  method: PaymentMethodSchema,
  currency: CurrencySchema.default("USD"),
});

export type CheckoutRequest = z.infer<typeof CheckoutRequestSchema>;

export const CartResponseSchema = z.object({
  cartId: UuidSchema,
  storeId: UuidSchema,
  kioskId: UuidSchema,
  sessionId: UuidSchema,
  lines: z.array(CartLineSchema).readonly(),
  totals: z.custom<CartTotals>(),
  version: z.number().int().nonnegative(),
  expiresAt: IsoTimestampSchema,
});

export type CartResponse = z.infer<typeof CartResponseSchema>;

// -----------------------------------------------------------------------------
// Payment API
// -----------------------------------------------------------------------------

export const PaymentResultSchema = z.object({
  paymentId: UuidSchema,
  orderId: UuidSchema,
  cartId: UuidSchema,
  amountCents: z.number().int().positive(),
  currency: CurrencySchema,
  method: PaymentMethodSchema,
  status: PaymentStatusSchema,
  authorization: z.custom<PaymentAuthorizationResult>(),
  receiptUrl: z.string().url().optional(),
});

export type PaymentResult = z.infer<typeof PaymentResultSchema>;

export const RefundRequestSchema = z.object({
  paymentId: UuidSchema,
  amountCents: z.number().int().positive(),
  reason: z.string().min(1).max(255),
});

export type RefundRequest = z.infer<typeof RefundRequestSchema>;

// -----------------------------------------------------------------------------
// Menu API
// -----------------------------------------------------------------------------

export const MenuResponseSchema = z.object({
  storeId: UuidSchema,
  currency: CurrencySchema,
  taxRate: z.number().min(0).max(1),
  categories: z.array(z.custom<Category>()).readonly(),
  items: z.array(z.custom<MenuItem>()).readonly(),
  staleAt: IsoTimestampSchema.optional(),
});

export type MenuResponse = z.infer<typeof MenuResponseSchema>;

// -----------------------------------------------------------------------------
// Order API
// -----------------------------------------------------------------------------

export const OrderResponseSchema = OrderSchema;

export type OrderResponse = z.infer<typeof OrderResponseSchema>;

export const CreateOrderRequestSchema = z.object({
  cartId: UuidSchema,
  paymentId: UuidSchema,
});

export type CreateOrderRequest = z.infer<typeof CreateOrderRequestSchema>;

// -----------------------------------------------------------------------------
// Kiosk heartbeat / health API
// -----------------------------------------------------------------------------

export const KioskHeartbeatRequestSchema = z.object({
  kioskId: UuidSchema,
  storeId: UuidSchema,
  firmwareVersion: z.string().max(32).optional(),
  syncStatus: z.enum(["online", "offline", "degraded", "maintenance"]).optional(),
  peerCount: z.number().int().nonnegative().optional(),
  queueDepth: z.number().int().nonnegative().optional(),
});

export type KioskHeartbeatRequest = z.infer<typeof KioskHeartbeatRequestSchema>;

export const KioskHeartbeatResponseSchema = z.object({
  kioskId: UuidSchema,
  accepted: z.boolean(),
  leaderKioskId: UuidSchema.nullable(),
  configVersion: z.number().int().nonnegative(),
});

export type KioskHeartbeatResponse = z.infer<typeof KioskHeartbeatResponseSchema>;

// -----------------------------------------------------------------------------
// Sync API
// -----------------------------------------------------------------------------

export const SyncBatchRequestSchema = z.object({
  kioskId: UuidSchema,
  storeId: UuidSchema,
  events: z.array(z.record(z.string(), z.unknown())).readonly(),
  vectorClock: z.record(z.string(), z.number().int().nonnegative()),
});

export type SyncBatchRequest = z.infer<typeof SyncBatchRequestSchema>;

export const SyncBatchResponseSchema = z.object({
  accepted: z.boolean(),
  conflicts: z.number().int().nonnegative(),
  vectorClock: z.record(z.string(), z.number().int().nonnegative()),
});

export type SyncBatchResponse = z.infer<typeof SyncBatchResponseSchema>;
