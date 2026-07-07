// -----------------------------------------------------------------------------
// Domain types mirroring database/schemas/drizzle.ts and database/schemas/go_structs.go.
// Identifiers are UUID v7 strings; timestamps are ISO 8601 strings with timezone.
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Enums (aligned with Drizzle pgEnum values)
// -----------------------------------------------------------------------------

export type TenantPlan = "standard" | "enterprise";

export type KioskSyncStatus = "online" | "offline" | "degraded" | "maintenance";

export type ItemTaxCategory = "standard" | "exempt" | "reduced";

export type WeightUnit = "g" | "kg" | "lb" | "oz";

export type CartStatus = "active" | "finalized" | "abandoned" | "expired";

export type OrderStatus = "pending" | "paid" | "fulfilled" | "cancelled" | "refunded";

export type PaymentMethod =
  | "credit_debit"
  | "nfc_apple_pay"
  | "nfc_google_pay"
  | "qr_code"
  | "cash_recycler";

export type PaymentStatus =
  | "pending"
  | "authorized"
  | "captured"
  | "queued_offline"
  | "declined"
  | "voided"
  | "refunded";

export type EmployeeRole = "cashier" | "supervisor" | "manager" | "admin";

export type AuditEventType =
  | "order_created"
  | "order_paid"
  | "order_refunded"
  | "payment_processed"
  | "inventory_adjusted"
  | "employee_login"
  | "employee_logout"
  | "system_boot"
  | "system_shutdown"
  | "sync_event"
  | "security_event";

export type InventoryTransactionType =
  | "sale"
  | "restock"
  | "adjustment"
  | "reserved"
  | "released"
  | "waste"
  | "return";

export type SyncEventType =
  | "inventory_update"
  | "cart_merge"
  | "transaction_batch"
  | "analytics_batch";

export type RefundStatus = "pending" | "completed" | "failed";

// -----------------------------------------------------------------------------
// Tenant / Location / Lane hierarchy
// -----------------------------------------------------------------------------

export interface Tenant {
  readonly tenantId: string;
  readonly slug: string;
  readonly name: string;
  readonly billingEmail: string;
  readonly plan: TenantPlan;
  readonly isActive: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface Location {
  readonly locationId: string;
  readonly tenantId: string;
  readonly slug: string;
  readonly name: string;
  readonly address: string | null;
  readonly timezone: string;
  readonly currency: string;
  readonly taxRate: number;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface Lane {
  readonly laneId: string;
  readonly locationId: string;
  readonly displayName: string;
  readonly laneNumber: number;
  readonly isActive: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

// -----------------------------------------------------------------------------
// Stores / Kiosks
// -----------------------------------------------------------------------------

export interface Store {
  readonly storeId: string;
  readonly tenantId: string | null;
  readonly locationId: string | null;
  readonly name: string;
  readonly address: string | null;
  readonly timezone: string;
  readonly currency: string;
  readonly taxRate: number;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface Kiosk {
  readonly kioskId: string;
  readonly storeId: string;
  readonly laneId: string | null;
  readonly tenantId: string | null;
  readonly hardwareId: string;
  readonly displayName: string;
  readonly ipAddress: string | null;
  readonly lastSeenAt: string | null;
  readonly syncStatus: KioskSyncStatus;
  readonly isLeader: boolean;
  readonly signingKeyHash: string;
  readonly firmwareVersion: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

// -----------------------------------------------------------------------------
// Menu: Categories / Items / Modifiers
// -----------------------------------------------------------------------------

export interface Category {
  readonly categoryId: string;
  readonly storeId: string;
  readonly parentId: string | null;
  readonly name: string;
  readonly description: string | null;
  readonly displayOrder: number;
  readonly imageUrl: string | null;
  readonly blurhash: string | null;
  readonly isActive: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface Item {
  readonly itemId: string;
  readonly storeId: string;
  readonly categoryId: string;
  readonly name: string;
  readonly description: string | null;
  readonly priceCents: number;
  readonly costCents: number | null;
  readonly plu: string | null;
  readonly barcode: string | null;
  readonly sku: string | null;
  readonly imageUrl: string | null;
  readonly blurhash: string | null;
  readonly taxCategory: ItemTaxCategory;
  readonly isWeightBased: boolean;
  readonly weightUnit: WeightUnit | null;
  readonly isActive: boolean;
  readonly metadata: Readonly<Record<string, unknown>> | null;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface ModifierGroup {
  readonly modifierGroupId: string;
  readonly storeId: string;
  readonly name: string;
  readonly description: string | null;
  readonly minSelect: number;
  readonly maxSelect: number;
  readonly displayOrder: number;
  readonly isActive: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface ModifierOption {
  readonly modifierOptionId: string;
  readonly modifierGroupId: string;
  readonly name: string;
  readonly priceDeltaCents: number;
  readonly isDefault: boolean;
  readonly displayOrder: number;
  readonly isActive: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface ItemModifierGroup {
  readonly itemId: string;
  readonly modifierGroupId: string;
  readonly createdAt: string;
}

// -----------------------------------------------------------------------------
// Inventory
// -----------------------------------------------------------------------------

export interface Inventory {
  readonly inventoryId: string;
  readonly storeId: string;
  readonly itemId: string;
  readonly quantityAvailable: number;
  readonly quantityReserved: number;
  readonly quantityOnOrder: number;
  readonly reorderPoint: number;
  readonly reorderQuantity: number;
  readonly location: string | null;
  readonly lastCountedAt: string | null;
  readonly updatedAt: string;
}

export interface InventoryTransaction {
  readonly transactionId: string;
  readonly storeId: string;
  readonly itemId: string;
  readonly transactionType: InventoryTransactionType;
  readonly quantityDelta: number;
  readonly runningBalance: number;
  readonly referenceId: string | null;
  readonly referenceType: string | null;
  readonly kioskId: string | null;
  readonly employeeId: string | null;
  readonly notes: string | null;
  readonly createdAt: string;
}

export interface InventoryReservation {
  readonly reservationId: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly itemId: string;
  readonly cartId: string;
  readonly quantity: number;
  readonly expiresAtMs: number;
  readonly createdAtMs: number;
}

// -----------------------------------------------------------------------------
// Carts
// -----------------------------------------------------------------------------

export interface CartLine {
  readonly lineId: string;
  readonly cartId: string;
  readonly menuItemId: string;
  readonly nameSnapshot: string;
  readonly unitPriceCentsSnapshot: number;
  readonly quantity: number;
  readonly modifiers: readonly unknown[];
  readonly addedAtMs: number;
}

export interface Cart {
  readonly cartId: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly sessionId: string;
  readonly customerPhone: string | null;
  readonly status: CartStatus;
  readonly finalized: boolean;
  readonly version: number;
  readonly totalCents: number;
  readonly taxCents: number;
  readonly discountCents: number;
  readonly finalTotalCents: number;
  readonly itemsJson: readonly CartLine[];
  readonly reservedInventory: boolean;
  readonly expiresAt: string;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly createdAtMs: number;
  readonly updatedAtMs: number;
}

// -----------------------------------------------------------------------------
// Orders
// -----------------------------------------------------------------------------

export interface OrderItemModifier {
  readonly modifierGroupId: string;
  readonly optionId: string;
  readonly name: string;
  readonly priceDeltaCents: number;
}

export interface OrderItem {
  readonly orderItemId: string;
  readonly orderId: string;
  readonly itemId: string;
  readonly nameSnapshot: string;
  readonly priceCentsSnapshot: number;
  readonly quantity: number;
  readonly modifiersJson: readonly OrderItemModifier[];
  readonly lineTotalCents: number;
  readonly createdAt: string;
}

export interface TaxBreakdownEntry {
  readonly label: string;
  readonly amountCents: number;
  readonly rate: number;
}

export interface Order {
  readonly orderId: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly cartId: string;
  readonly orderNumber: string;
  readonly status: OrderStatus;
  readonly subtotalCents: number;
  readonly taxCents: number;
  readonly discountCents: number;
  readonly totalCents: number;
  readonly itemsJson: readonly OrderItem[];
  readonly taxBreakdownJson: readonly TaxBreakdownEntry[] | null;
  readonly metadata: Readonly<Record<string, unknown>> | null;
  readonly paidAt: string | null;
  readonly fulfilledAt: string | null;
  readonly cancelledAt: string | null;
  readonly createdAt: string;
}

// -----------------------------------------------------------------------------
// Payments / Refunds / Offline Tokens
// -----------------------------------------------------------------------------

export interface Payment {
  readonly paymentId: string;
  readonly orderId: string;
  readonly kioskId: string;
  readonly idempotencyKey: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly method: PaymentMethod;
  readonly status: PaymentStatus;
  readonly verifoneToken: string | null;
  readonly verifoneAuthCode: string | null;
  readonly cardBrand: string | null;
  readonly cardLastFour: string | null;
  readonly declineReason: string | null;
  readonly receiptText: string | null;
  readonly isOfflineToken: boolean;
  readonly offlineTokenHmac: string | null;
  readonly syncedAt: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface Refund {
  readonly refundId: string;
  readonly paymentId: string;
  readonly orderId: string;
  readonly kioskId: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly reason: string;
  readonly status: RefundStatus;
  readonly verifoneReference: string | null;
  readonly processedBy: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface OfflineToken {
  readonly tokenId: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly cartId: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly method: string;
  readonly verifoneOpaqueToken: string;
  readonly hmacSignature: string;
  readonly expiresAt: string;
  readonly settledAt: string | null;
  readonly settlementResult: Readonly<Record<string, unknown>> | null;
  readonly createdAt: string;
  readonly updatedAt: string;
}

// -----------------------------------------------------------------------------
// Employees / Users / RBAC
// -----------------------------------------------------------------------------

export interface Employee {
  readonly employeeId: string;
  readonly storeId: string;
  readonly name: string;
  readonly email: string;
  readonly role: EmployeeRole;
  readonly biometricHash: string | null;
  readonly webauthnCredentialId: string | null;
  readonly webauthnPublicKey: Uint8Array | null;
  readonly isActive: boolean;
  readonly lastLoginAt: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

export interface Role {
  readonly roleId: string;
  readonly tenantId: string;
  readonly name: string;
  readonly description: string | null;
  readonly isSystem: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface Permission {
  readonly permissionId: string;
  readonly resource: string;
  readonly action: string;
  readonly description: string | null;
}

export interface RolePermission {
  readonly roleId: string;
  readonly permissionId: string;
  readonly createdAt: string;
}

export interface User {
  readonly userId: string;
  readonly tenantId: string;
  readonly email: string;
  readonly name: string;
  readonly roleId: string;
  readonly isActive: boolean;
  readonly webauthnCredentialId: string | null;
  readonly webauthnPublicKey: Uint8Array | null;
  readonly lastLoginAt: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly deletedAt: string | null;
}

// -----------------------------------------------------------------------------
// Audit / Event Store / Outbox / Sync / Analytics
// -----------------------------------------------------------------------------

export interface AuditLog {
  readonly auditId: number;
  readonly storeId: string;
  readonly tenantId: string | null;
  readonly laneId: string | null;
  readonly kioskId: string | null;
  readonly employeeId: string | null;
  readonly userId: string | null;
  readonly eventType: AuditEventType;
  readonly entityType: string;
  readonly entityId: string;
  readonly payloadJson: Readonly<Record<string, unknown>>;
  readonly previousHash: string;
  readonly currentHash: string;
  readonly createdAt: string;
}

export interface EventStoreRecord {
  readonly eventId: string;
  readonly eventSchema: string;
  readonly aggregateType: string;
  readonly aggregateId: string;
  readonly sequenceNumber: number;
  readonly payload: Readonly<Record<string, unknown>>;
  readonly metadata: Readonly<Record<string, unknown>>;
  readonly occurredAt: string;
  readonly recordedAt: string;
}

export interface OutboxEvent {
  readonly eventId: string;
  readonly aggregateType: string;
  readonly aggregateId: string;
  readonly eventType: string;
  readonly payload: Readonly<Record<string, unknown>>;
  readonly occurredAtMs: number;
  readonly published: boolean;
  readonly publishedAtMs: number | null;
  readonly createdAt: string;
}

export interface SyncEvent {
  readonly syncEventId: string;
  readonly storeId: string;
  readonly kioskId: string;
  readonly eventType: SyncEventType;
  readonly payloadJson: Readonly<Record<string, unknown>>;
  readonly vectorClock: Readonly<Record<string, number>>;
  readonly processedAt: string | null;
  readonly createdAt: string;
}

export interface AnalyticsEvent {
  readonly analyticsId: number;
  readonly storeId: string;
  readonly kioskId: string | null;
  readonly eventType: string;
  readonly sessionId: string | null;
  readonly customerHash: string | null;
  readonly itemId: string | null;
  readonly categoryId: string | null;
  readonly quantity: number | null;
  readonly amountCents: number | null;
  readonly durationMs: number | null;
  readonly metadata: Readonly<Record<string, unknown>> | null;
  readonly createdAt: string;
}
