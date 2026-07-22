import { z } from "zod";
import { UuidSchema } from "../ids";

// -----------------------------------------------------------------------------
// Common primitives
// -----------------------------------------------------------------------------

export const IsoTimestampSchema = z.iso.datetime({ offset: true });

export const CurrencySchema = z.string().length(3);

export const EmailSchema = z
  .email()
  .max(255)
  .regex(/^[^\s@]+@[^\s@]+\.[^\s@]+$/, "Expected a valid email address");

export const HexHash64Schema = z.string().regex(/^[a-f0-9]{64}$/i, "Expected 64-character hex hash");

// -----------------------------------------------------------------------------
// Enum schemas
// -----------------------------------------------------------------------------

export const TenantPlanSchema = z.enum(["standard", "enterprise"]);

export const KioskSyncStatusSchema = z.enum([
  "online",
  "offline",
  "degraded",
  "maintenance",
]);

export const ItemTaxCategorySchema = z.enum(["standard", "exempt", "reduced"]);

export const WeightUnitSchema = z.enum(["g", "kg", "lb", "oz"]);

export const CartStatusSchema = z.enum(["active", "finalized", "abandoned", "expired"]);

export const OrderStatusSchema = z.enum(["pending", "paid", "fulfilled", "cancelled", "refunded"]);

export const PaymentMethodSchema = z.enum([
  "credit_debit",
  "nfc_apple_pay",
  "nfc_google_pay",
  "qr_code",
  "cash_recycler",
]);

export const PaymentStatusSchema = z.enum([
  "pending",
  "authorized",
  "captured",
  "queued_offline",
  "declined",
  "voided",
  "refunded",
]);

export const EmployeeRoleSchema = z.enum(["cashier", "supervisor", "manager", "admin"]);

export const AuditEventTypeSchema = z.enum([
  "order_created",
  "order_paid",
  "order_refunded",
  "payment_processed",
  "inventory_adjusted",
  "employee_login",
  "employee_logout",
  "system_boot",
  "system_shutdown",
  "sync_event",
  "security_event",
]);

export const InventoryTransactionTypeSchema = z.enum([
  "sale",
  "restock",
  "adjustment",
  "reserved",
  "released",
  "waste",
  "return",
]);

export const SyncEventTypeSchema = z.enum([
  "inventory_update",
  "cart_merge",
  "transaction_batch",
  "analytics_batch",
]);

export const RefundStatusSchema = z.enum(["pending", "completed", "failed"]);

// -----------------------------------------------------------------------------
// Tenant / Location / Lane hierarchy
// -----------------------------------------------------------------------------

export const TenantSchema = z.object({
  tenantId: UuidSchema,
  slug: z.string().min(1).max(64),
  name: z.string().min(1).max(255),
  billingEmail: EmailSchema,
  plan: TenantPlanSchema,
  isActive: z.boolean(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const LocationSchema = z.object({
  locationId: UuidSchema,
  tenantId: UuidSchema,
  slug: z.string().min(1).max(64),
  name: z.string().min(1).max(255),
  address: z.string().nullable(),
  timezone: z.string().min(1).max(64),
  currency: CurrencySchema,
  taxRate: z.number().min(0).max(1),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const LaneSchema = z.object({
  laneId: UuidSchema,
  locationId: UuidSchema,
  displayName: z.string().min(1).max(64),
  laneNumber: z.number().int(),
  isActive: z.boolean(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

// -----------------------------------------------------------------------------
// Stores / Kiosks
// -----------------------------------------------------------------------------

export const StoreSchema = z.object({
  storeId: UuidSchema,
  tenantId: UuidSchema.nullable(),
  locationId: UuidSchema.nullable(),
  name: z.string().min(1).max(255),
  address: z.string().nullable(),
  timezone: z.string().min(1).max(64),
  currency: CurrencySchema,
  taxRate: z.number().min(0).max(1),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const KioskSchema = z.object({
  kioskId: UuidSchema,
  storeId: UuidSchema,
  laneId: UuidSchema.nullable(),
  tenantId: UuidSchema.nullable(),
  hardwareId: z.string().min(1).max(64),
  displayName: z.string().min(1).max(64),
  ipAddress: z.string().regex(/^(\d{1,3}\.){3}\d{1,3}$/).nullable(),
  lastSeenAt: IsoTimestampSchema.nullable(),
  syncStatus: KioskSyncStatusSchema,
  isLeader: z.boolean(),
  signingKeyHash: HexHash64Schema,
  firmwareVersion: z.string().max(32).nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

// -----------------------------------------------------------------------------
// Menu: Categories / Items / Modifiers
// -----------------------------------------------------------------------------

export const CategorySchema = z.object({
  categoryId: UuidSchema,
  storeId: UuidSchema,
  parentId: UuidSchema.nullable(),
  name: z.string().min(1).max(128),
  description: z.string().nullable(),
  displayOrder: z.number().int(),
  imageUrl: z.url().max(512).nullable(),
  blurhash: z.string().max(32).nullable(),
  isActive: z.boolean(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const ItemSchema = z.object({
  itemId: UuidSchema,
  storeId: UuidSchema,
  categoryId: UuidSchema,
  name: z.string().min(1).max(255),
  description: z.string().nullable(),
  priceCents: z.number().int().nonnegative(),
  costCents: z.number().int().nonnegative().nullable(),
  plu: z.string().max(16).nullable(),
  barcode: z.string().max(32).nullable(),
  sku: z.string().max(64).nullable(),
  imageUrl: z.url().max(512).nullable(),
  blurhash: z.string().max(32).nullable(),
  taxCategory: ItemTaxCategorySchema,
  isWeightBased: z.boolean(),
  weightUnit: WeightUnitSchema.nullable(),
  isActive: z.boolean(),
  metadata: z.record(z.string(), z.unknown()).nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const ModifierGroupSchema = z
  .object({
    modifierGroupId: UuidSchema,
    storeId: UuidSchema,
    name: z.string().min(1).max(128),
    description: z.string().nullable(),
    minSelect: z.number().int().nonnegative(),
    maxSelect: z.number().int().nonnegative(),
    displayOrder: z.number().int(),
    isActive: z.boolean(),
    createdAt: IsoTimestampSchema,
    updatedAt: IsoTimestampSchema,
    deletedAt: IsoTimestampSchema.nullable(),
  })
  .refine((group: { maxSelect: number; minSelect: number }) => group.maxSelect >= group.minSelect, {
    message: "maxSelect must be greater than or equal to minSelect",
    path: ["maxSelect"],
  });

export const ModifierOptionSchema = z.object({
  modifierOptionId: UuidSchema,
  modifierGroupId: UuidSchema,
  name: z.string().min(1).max(128),
  priceDeltaCents: z.number().int().default(0),
  isDefault: z.boolean(),
  displayOrder: z.number().int(),
  isActive: z.boolean(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const ItemModifierGroupSchema = z.object({
  itemId: UuidSchema,
  modifierGroupId: UuidSchema,
  createdAt: IsoTimestampSchema,
});

// -----------------------------------------------------------------------------
// Inventory
// -----------------------------------------------------------------------------

export const InventorySchema = z.object({
  inventoryId: UuidSchema,
  storeId: UuidSchema,
  itemId: UuidSchema,
  quantityAvailable: z.number().int(),
  quantityReserved: z.number().int(),
  quantityOnOrder: z.number().int(),
  reorderPoint: z.number().int().nonnegative(),
  reorderQuantity: z.number().int().nonnegative(),
  location: z.string().max(64).nullable(),
  lastCountedAt: IsoTimestampSchema.nullable(),
  updatedAt: IsoTimestampSchema,
});

export const InventoryTransactionSchema = z.object({
  transactionId: UuidSchema,
  storeId: UuidSchema,
  itemId: UuidSchema,
  transactionType: InventoryTransactionTypeSchema,
  quantityDelta: z.number().int(),
  runningBalance: z.number().int(),
  referenceId: UuidSchema.nullable(),
  referenceType: z.string().max(32).nullable(),
  kioskId: UuidSchema.nullable(),
  employeeId: UuidSchema.nullable(),
  notes: z.string().max(500).nullable(),
  createdAt: IsoTimestampSchema,
});

export const InventoryReservationSchema = z.object({
  reservationId: UuidSchema,
  storeId: UuidSchema,
  kioskId: UuidSchema,
  itemId: UuidSchema,
  cartId: UuidSchema,
  quantity: z.number().int().positive(),
  expiresAtMs: z.number().int().positive(),
  createdAtMs: z.number().int().positive(),
});

// -----------------------------------------------------------------------------
// Carts
// -----------------------------------------------------------------------------

export const CartLineSchema = z.object({
  lineId: UuidSchema,
  cartId: UuidSchema,
  menuItemId: UuidSchema,
  nameSnapshot: z.string().min(1).max(255),
  unitPriceCentsSnapshot: z.number().int().nonnegative(),
  quantity: z.number().int().positive(),
  modifiers: z.array(z.record(z.string(), z.unknown())).readonly(),
  addedAtMs: z.number().int().positive(),
});

export const CartSchema = z
  .object({
    cartId: UuidSchema,
    storeId: UuidSchema,
    kioskId: UuidSchema,
    sessionId: UuidSchema,
    customerPhone: z.string().max(16).nullable(),
    status: CartStatusSchema,
    finalized: z.boolean(),
    version: z.number().int().nonnegative(),
    totalCents: z.number().int().nonnegative(),
    taxCents: z.number().int().nonnegative(),
    discountCents: z.number().int().nonnegative(),
    finalTotalCents: z.number().int().nonnegative(),
    itemsJson: z.array(CartLineSchema).readonly(),
    reservedInventory: z.boolean(),
    expiresAt: IsoTimestampSchema,
    createdAt: IsoTimestampSchema,
    updatedAt: IsoTimestampSchema,
    createdAtMs: z.number().int().positive(),
    updatedAtMs: z.number().int().positive(),
  })
  .refine(
    (cart: { status: string; finalized: boolean; itemsJson: readonly unknown[] }) =>
      !(cart.status === "finalized" || cart.finalized) || cart.itemsJson.length > 0,
    {
      message: "Finalized carts must contain at least one line item",
      path: ["itemsJson"],
    },
  );

// -----------------------------------------------------------------------------
// Orders
// -----------------------------------------------------------------------------

export const OrderItemModifierSchema = z.object({
  modifierGroupId: UuidSchema,
  optionId: UuidSchema,
  name: z.string().min(1).max(128),
  priceDeltaCents: z.number().int(),
});

export const OrderItemSchema = z.object({
  orderItemId: UuidSchema,
  orderId: UuidSchema,
  itemId: UuidSchema,
  nameSnapshot: z.string().min(1).max(255),
  priceCentsSnapshot: z.number().int().nonnegative(),
  quantity: z.number().int().positive(),
  modifiersJson: z.array(OrderItemModifierSchema).readonly(),
  lineTotalCents: z.number().int().nonnegative(),
  createdAt: IsoTimestampSchema,
});

export const TaxBreakdownEntrySchema = z.object({
  label: z.string().min(1),
  amountCents: z.number().int(),
  rate: z.number().min(0).max(1),
});

export const OrderSchema = z.object({
  orderId: UuidSchema,
  storeId: UuidSchema,
  kioskId: UuidSchema,
  cartId: UuidSchema,
  orderNumber: z.string().min(1).max(16),
  status: OrderStatusSchema,
  subtotalCents: z.number().int().nonnegative(),
  taxCents: z.number().int().nonnegative(),
  discountCents: z.number().int().nonnegative(),
  totalCents: z.number().int().nonnegative(),
  itemsJson: z.array(OrderItemSchema).readonly(),
  taxBreakdownJson: z.array(TaxBreakdownEntrySchema).readonly().nullable(),
  metadata: z.record(z.string(), z.unknown()).nullable(),
  paidAt: IsoTimestampSchema.nullable(),
  fulfilledAt: IsoTimestampSchema.nullable(),
  cancelledAt: IsoTimestampSchema.nullable(),
  createdAt: IsoTimestampSchema,
});

// -----------------------------------------------------------------------------
// Payments / Refunds / Offline Tokens
// -----------------------------------------------------------------------------

export const PaymentSchema = z.object({
  paymentId: UuidSchema,
  orderId: UuidSchema,
  kioskId: UuidSchema,
  idempotencyKey: UuidSchema,
  amountCents: z.number().int().positive(),
  currency: CurrencySchema,
  method: PaymentMethodSchema,
  status: PaymentStatusSchema,
  verifoneToken: z.string().max(255).nullable(),
  verifoneAuthCode: z.string().max(16).nullable(),
  cardBrand: z.string().max(16).nullable(),
  cardLastFour: z.string().length(4).nullable(),
  declineReason: z.string().max(255).nullable(),
  receiptText: z.string().nullable(),
  isOfflineToken: z.boolean(),
  offlineTokenHmac: HexHash64Schema.nullable(),
  syncedAt: IsoTimestampSchema.nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
});

export const RefundSchema = z.object({
  refundId: UuidSchema,
  paymentId: UuidSchema,
  orderId: UuidSchema,
  kioskId: UuidSchema,
  amountCents: z.number().int().positive(),
  currency: CurrencySchema,
  reason: z.string().min(1).max(255),
  status: RefundStatusSchema,
  verifoneReference: z.string().max(255).nullable(),
  processedBy: UuidSchema.nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
});

export const OfflineTokenSchema = z.object({
  tokenId: UuidSchema,
  storeId: UuidSchema,
  kioskId: UuidSchema,
  cartId: UuidSchema,
  amountCents: z.number().int().positive(),
  currency: CurrencySchema,
  method: z.string().min(1).max(16),
  verifoneOpaqueToken: z.string().min(1).max(255),
  hmacSignature: HexHash64Schema,
  expiresAt: IsoTimestampSchema,
  settledAt: IsoTimestampSchema.nullable(),
  settlementResult: z.record(z.string(), z.unknown()).nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
});

// -----------------------------------------------------------------------------
// Employees / Users / RBAC
// -----------------------------------------------------------------------------

export const EmployeeSchema = z.object({
  employeeId: UuidSchema,
  storeId: UuidSchema,
  name: z.string().min(1).max(128),
  email: EmailSchema,
  role: EmployeeRoleSchema,
  biometricHash: HexHash64Schema.nullable(),
  webauthnCredentialId: z.string().max(255).nullable(),
  webauthnPublicKey: z.instanceof(Uint8Array).nullable(),
  isActive: z.boolean(),
  lastLoginAt: IsoTimestampSchema.nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

export const RoleSchema = z.object({
  roleId: UuidSchema,
  tenantId: UuidSchema,
  name: z.string().min(1).max(64),
  description: z.string().max(500).nullable(),
  isSystem: z.boolean(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
});

export const PermissionSchema = z.object({
  permissionId: UuidSchema,
  resource: z.string().min(1).max(64),
  action: z.string().min(1).max(64),
  description: z.string().max(500).nullable(),
});

export const RolePermissionSchema = z.object({
  roleId: UuidSchema,
  permissionId: UuidSchema,
  createdAt: IsoTimestampSchema,
});

export const UserSchema = z.object({
  userId: UuidSchema,
  tenantId: UuidSchema,
  email: EmailSchema,
  name: z.string().min(1).max(128),
  roleId: UuidSchema,
  isActive: z.boolean(),
  webauthnCredentialId: z.string().max(255).nullable(),
  webauthnPublicKey: z.instanceof(Uint8Array).nullable(),
  lastLoginAt: IsoTimestampSchema.nullable(),
  createdAt: IsoTimestampSchema,
  updatedAt: IsoTimestampSchema,
  deletedAt: IsoTimestampSchema.nullable(),
});

// -----------------------------------------------------------------------------
// Audit / Event Store / Outbox / Sync / Analytics
// -----------------------------------------------------------------------------

export const AuditLogSchema = z.object({
  auditId: z.number().int().nonnegative(),
  storeId: UuidSchema,
  tenantId: UuidSchema.nullable(),
  laneId: UuidSchema.nullable(),
  kioskId: UuidSchema.nullable(),
  employeeId: UuidSchema.nullable(),
  userId: UuidSchema.nullable(),
  eventType: AuditEventTypeSchema,
  entityType: z.string().min(1).max(32),
  entityId: UuidSchema,
  payloadJson: z.record(z.string(), z.unknown()),
  previousHash: HexHash64Schema,
  currentHash: HexHash64Schema,
  createdAt: IsoTimestampSchema,
});

export const EventStoreRecordSchema = z.object({
  eventId: UuidSchema,
  eventSchema: z.string().min(1).max(64),
  aggregateType: z.string().min(1).max(64),
  aggregateId: UuidSchema,
  sequenceNumber: z.number().int().nonnegative(),
  payload: z.record(z.string(), z.unknown()),
  metadata: z.record(z.string(), z.unknown()),
  occurredAt: IsoTimestampSchema,
  recordedAt: IsoTimestampSchema,
});

export const OutboxEventSchema = z.object({
  eventId: UuidSchema,
  aggregateType: z.string().min(1).max(64),
  aggregateId: UuidSchema,
  eventType: z.string().min(1).max(128),
  payload: z.record(z.string(), z.unknown()),
  occurredAtMs: z.number().int().positive(),
  published: z.boolean(),
  publishedAtMs: z.number().int().positive().nullable(),
  createdAt: IsoTimestampSchema,
});

export const SyncEventSchema = z.object({
  syncEventId: UuidSchema,
  storeId: UuidSchema,
  kioskId: UuidSchema,
  eventType: SyncEventTypeSchema,
  payloadJson: z.record(z.string(), z.unknown()),
  vectorClock: z.record(z.string(), z.number().int().nonnegative()),
  processedAt: IsoTimestampSchema.nullable(),
  createdAt: IsoTimestampSchema,
});

export const AnalyticsEventSchema = z.object({
  analyticsId: z.number().int().nonnegative(),
  storeId: UuidSchema,
  kioskId: UuidSchema.nullable(),
  eventType: z.string().min(1).max(32),
  sessionId: UuidSchema.nullable(),
  customerHash: z.string().max(64).nullable(),
  itemId: UuidSchema.nullable(),
  categoryId: UuidSchema.nullable(),
  quantity: z.number().int().positive().nullable(),
  amountCents: z.number().int().positive().nullable(),
  durationMs: z.number().int().nonnegative().nullable(),
  metadata: z.record(z.string(), z.unknown()).nullable(),
  createdAt: IsoTimestampSchema,
});
