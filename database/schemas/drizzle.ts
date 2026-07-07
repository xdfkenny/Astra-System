import { relations } from "drizzle-orm/relations";
import { sql } from "drizzle-orm/sql/sql";
import type { AnyPgColumn } from "drizzle-orm/pg-core";
import {
  bigint,
  boolean,
  customType,
  decimal,
  index,
  inet,
  integer,
  jsonb,
  pgEnum,
  pgTable,
  primaryKey,
  text,
  timestamp,
  unique,
  uniqueIndex,
  uuid,
  varchar,
} from "drizzle-orm/pg-core";

// ---------------------------------------------------------------------------
// Custom column types
// ---------------------------------------------------------------------------

export const bytea = customType<{ data: Buffer }>({
  dataType() {
    return "bytea";
  },
});

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export const tenantPlanEnum = pgEnum("tenant_plan", ["standard", "enterprise"]);

export const kioskSyncStatusEnum = pgEnum("kiosk_sync_status", [
  "online",
  "offline",
  "degraded",
  "maintenance",
]);

export const itemTaxCategoryEnum = pgEnum("item_tax_category", [
  "standard",
  "exempt",
  "reduced",
]);

export const weightUnitEnum = pgEnum("weight_unit", ["g", "kg", "lb", "oz"]);

export const cartStatusEnum = pgEnum("cart_status", [
  "active",
  "finalized",
  "abandoned",
  "expired",
]);

export const orderStatusEnum = pgEnum("order_status", [
  "pending",
  "paid",
  "fulfilled",
  "cancelled",
  "refunded",
]);

export const paymentMethodEnum = pgEnum("payment_method", [
  "credit_debit",
  "nfc_apple_pay",
  "nfc_google_pay",
  "qr_code",
  "cash_recycler",
]);

export const paymentStatusEnum = pgEnum("payment_status", [
  "pending",
  "authorized",
  "captured",
  "declined",
  "voided",
  "refunded",
]);

export const employeeRoleEnum = pgEnum("employee_role", [
  "cashier",
  "supervisor",
  "manager",
  "admin",
]);

export const auditEventTypeEnum = pgEnum("audit_event_type", [
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

export const inventoryTransactionTypeEnum = pgEnum("inventory_transaction_type", [
  "sale",
  "restock",
  "adjustment",
  "reserved",
  "released",
  "waste",
  "return",
]);

export const syncEventTypeEnum = pgEnum("sync_event_type", [
  "inventory_update",
  "cart_merge",
  "transaction_batch",
  "analytics_batch",
]);

export const refundStatusEnum = pgEnum("refund_status", [
  "pending",
  "completed",
  "failed",
]);

// ---------------------------------------------------------------------------
// Tenant / Location / Lane hierarchy
// ---------------------------------------------------------------------------

export const tenants = pgTable(
  "tenants",
  {
    tenantId: uuid("tenant_id").primaryKey().defaultRandom(),
    slug: varchar("slug", { length: 64 }).notNull().unique(),
    name: varchar("name", { length: 255 }).notNull(),
    billingEmail: varchar("billing_email", { length: 255 }).notNull(),
    plan: tenantPlanEnum("plan").notNull().default("standard"),
    isActive: boolean("is_active").notNull().default(true),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_tenants_slug")
      .on(table.slug)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

export const locations = pgTable(
  "locations",
  {
    locationId: uuid("location_id").primaryKey().defaultRandom(),
    tenantId: uuid("tenant_id")
      .notNull()
      .references(() => tenants.tenantId),
    slug: varchar("slug", { length: 64 }).notNull(),
    name: varchar("name", { length: 255 }).notNull(),
    address: text("address"),
    timezone: varchar("timezone", { length: 64 }).notNull().default("UTC"),
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    taxRate: decimal("tax_rate", { precision: 5, scale: 4 }).notNull().default("0.0000"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    unique("locations_tenant_slug_unique").on(table.tenantId, table.slug),
    index("idx_locations_tenant")
      .on(table.tenantId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

export const lanes = pgTable(
  "lanes",
  {
    laneId: uuid("lane_id").primaryKey().defaultRandom(),
    locationId: uuid("location_id")
      .notNull()
      .references(() => locations.locationId),
    displayName: varchar("display_name", { length: 64 }).notNull(),
    laneNumber: integer("lane_number").notNull(),
    isActive: boolean("is_active").notNull().default(true),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    unique("lanes_location_number_unique").on(table.locationId, table.laneNumber),
    index("idx_lanes_location")
      .on(table.locationId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

// ---------------------------------------------------------------------------
// Stores / Kiosks
// ---------------------------------------------------------------------------

export const stores = pgTable(
  "stores",
  {
    storeId: uuid("store_id").primaryKey().defaultRandom(),
    tenantId: uuid("tenant_id").references(() => tenants.tenantId),
    locationId: uuid("location_id").references(() => locations.locationId),
    name: varchar("name", { length: 255 }).notNull(),
    address: text("address"),
    timezone: varchar("timezone", { length: 64 }).notNull().default("UTC"),
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    taxRate: decimal("tax_rate", { precision: 5, scale: 4 }).notNull().default("0.0000"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_stores_deleted_at")
      .on(table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_stores_tenant")
      .on(table.tenantId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_stores_location")
      .on(table.locationId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

export const kiosks = pgTable(
  "kiosks",
  {
    kioskId: uuid("kiosk_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    laneId: uuid("lane_id").references(() => lanes.laneId),
    tenantId: uuid("tenant_id").references(() => tenants.tenantId),
    hardwareId: varchar("hardware_id", { length: 64 }).notNull().unique(),
    displayName: varchar("display_name", { length: 64 }).notNull(),
    ipAddress: inet("ip_address"),
    lastSeenAt: timestamp("last_seen_at", { withTimezone: true }),
    syncStatus: kioskSyncStatusEnum("sync_status").notNull().default("online"),
    isLeader: boolean("is_leader").notNull().default(false),
    signingKeyHash: varchar("signing_key_hash", { length: 64 }).notNull(),
    firmwareVersion: varchar("firmware_version", { length: 32 }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_kiosks_store")
      .on(table.storeId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_kiosks_leader")
      .on(table.storeId, table.isLeader)
      .where(sql`${table.isLeader} = TRUE AND ${table.deletedAt} IS NULL`),
    uniqueIndex("idx_kiosks_one_leader_per_store")
      .on(table.storeId)
      .where(sql`${table.isLeader} = TRUE AND ${table.deletedAt} IS NULL`),
    index("idx_kiosks_lane")
      .on(table.laneId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_kiosks_tenant")
      .on(table.tenantId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

// ---------------------------------------------------------------------------
// Menu: Categories / Items / Modifiers
// ---------------------------------------------------------------------------

export const categories = pgTable(
  "categories",
  {
    categoryId: uuid("category_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    parentId: uuid("parent_id").references((): AnyPgColumn => categories.categoryId),
    name: varchar("name", { length: 128 }).notNull(),
    description: text("description"),
    displayOrder: integer("display_order").notNull().default(0),
    imageUrl: varchar("image_url", { length: 512 }),
    blurhash: varchar("blurhash", { length: 32 }),
    isActive: boolean("is_active").notNull().default(true),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_categories_store")
      .on(table.storeId, table.displayOrder, table.isActive)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

export const items = pgTable(
  "items",
  {
    itemId: uuid("item_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    categoryId: uuid("category_id")
      .notNull()
      .references(() => categories.categoryId),
    name: varchar("name", { length: 255 }).notNull(),
    description: text("description"),
    priceCents: integer("price_cents").notNull(),
    costCents: integer("cost_cents"),
    plu: varchar("plu", { length: 16 }),
    barcode: varchar("barcode", { length: 32 }),
    sku: varchar("sku", { length: 64 }),
    imageUrl: varchar("image_url", { length: 512 }),
    blurhash: varchar("blurhash", { length: 32 }),
    taxCategory: itemTaxCategoryEnum("tax_category").notNull().default("standard"),
    isWeightBased: boolean("is_weight_based").notNull().default(false),
    weightUnit: weightUnitEnum("weight_unit"),
    isActive: boolean("is_active").notNull().default(true),
    metadata: jsonb("metadata"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_items_store_category")
      .on(table.storeId, table.categoryId, table.isActive)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_items_barcode")
      .on(table.barcode)
      .where(sql`${table.barcode} IS NOT NULL AND ${table.deletedAt} IS NULL`),
    index("idx_items_plu")
      .on(table.plu)
      .where(sql`${table.plu} IS NOT NULL AND ${table.deletedAt} IS NULL`),
    index("idx_items_name_trgm").using("gin", table.name),
  ]
);

export const modifierGroups = pgTable(
  "modifier_groups",
  {
    modifierGroupId: uuid("modifier_group_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    name: varchar("name", { length: 128 }).notNull(),
    description: text("description"),
    minSelect: integer("min_select").notNull().default(0),
    maxSelect: integer("max_select").notNull().default(1),
    displayOrder: integer("display_order").notNull().default(0),
    isActive: boolean("is_active").notNull().default(true),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  () => []
);

export const modifierOptions = pgTable(
  "modifier_options",
  {
    modifierOptionId: uuid("modifier_option_id").primaryKey().defaultRandom(),
    modifierGroupId: uuid("modifier_group_id")
      .notNull()
      .references(() => modifierGroups.modifierGroupId),
    name: varchar("name", { length: 128 }).notNull(),
    priceDeltaCents: integer("price_delta_cents").notNull().default(0),
    isDefault: boolean("is_default").notNull().default(false),
    displayOrder: integer("display_order").notNull().default(0),
    isActive: boolean("is_active").notNull().default(true),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  () => []
);

export const itemModifierGroups = pgTable(
  "item_modifier_groups",
  {
    itemId: uuid("item_id")
      .notNull()
      .references(() => items.itemId),
    modifierGroupId: uuid("modifier_group_id")
      .notNull()
      .references(() => modifierGroups.modifierGroupId),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [primaryKey({ columns: [table.itemId, table.modifierGroupId] })]
);

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

export const inventory = pgTable(
  "inventory",
  {
    inventoryId: uuid("inventory_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    itemId: uuid("item_id")
      .notNull()
      .references(() => items.itemId),
    quantityAvailable: integer("quantity_available").notNull().default(0),
    quantityReserved: integer("quantity_reserved").notNull().default(0),
    quantityOnOrder: integer("quantity_on_order").notNull().default(0),
    reorderPoint: integer("reorder_point").notNull().default(0),
    reorderQuantity: integer("reorder_quantity").notNull().default(0),
    location: varchar("location", { length: 64 }),
    lastCountedAt: timestamp("last_counted_at", { withTimezone: true }),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    unique("inventory_store_item_unique").on(table.storeId, table.itemId),
    index("idx_inventory_store").on(table.storeId),
    index("idx_inventory_low_stock")
      .on(table.storeId, table.quantityAvailable)
      .where(sql`${table.quantityAvailable} <= ${table.reorderPoint}`),
  ]
);

export const inventoryTransactions = pgTable(
  "inventory_transactions",
  {
    transactionId: uuid("transaction_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    itemId: uuid("item_id")
      .notNull()
      .references(() => items.itemId),
    transactionType: inventoryTransactionTypeEnum("transaction_type").notNull(),
    quantityDelta: integer("quantity_delta").notNull(),
    runningBalance: integer("running_balance").notNull(),
    referenceId: uuid("reference_id"),
    referenceType: varchar("reference_type", { length: 32 }),
    kioskId: uuid("kiosk_id").references(() => kiosks.kioskId),
    employeeId: uuid("employee_id").references(() => employees.employeeId),
    notes: varchar("notes", { length: 500 }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_inventory_transactions_store_item").on(
      table.storeId,
      table.itemId,
      table.createdAt
    ),
    index("idx_inventory_transactions_reference").on(table.referenceType, table.referenceId),
  ]
);

export const inventoryReservations = pgTable(
  "inventory_reservations",
  {
    reservationId: uuid("reservation_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    itemId: uuid("item_id")
      .notNull()
      .references(() => items.itemId),
    cartId: uuid("cart_id")
      .notNull()
      .references(() => carts.cartId),
    quantity: integer("quantity").notNull(),
    expiresAtMs: bigint("expires_at_ms", { mode: "number" }).notNull(),
    createdAtMs: bigint("created_at_ms", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_inventory_reservations_item").on(table.itemId, table.expiresAtMs),
    index("idx_inventory_reservations_cart").on(table.cartId),
    index("idx_inventory_reservations_expires").on(table.expiresAtMs),
  ]
);

// ---------------------------------------------------------------------------
// Carts
// ---------------------------------------------------------------------------

export const carts = pgTable(
  "carts",
  {
    cartId: uuid("cart_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    sessionId: uuid("session_id").notNull(),
    customerPhone: varchar("customer_phone", { length: 16 }),
    status: cartStatusEnum("status").notNull().default("active"),
    finalized: boolean("finalized").notNull().default(false),
    version: integer("version").notNull().default(0),
    totalCents: integer("total_cents").notNull().default(0),
    taxCents: integer("tax_cents").notNull().default(0),
    discountCents: integer("discount_cents").notNull().default(0),
    finalTotalCents: integer("final_total_cents").notNull().default(0),
    itemsJson: jsonb("items_json").notNull().default("[]"),
    reservedInventory: boolean("reserved_inventory").notNull().default(false),
    expiresAt: timestamp("expires_at", { withTimezone: true })
      .notNull()
      .default(sql`NOW() + INTERVAL '10 minutes'`),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    createdAtMs: bigint("created_at_ms", { mode: "number" }).notNull().default(
      sql`(EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT`
    ),
    updatedAtMs: bigint("updated_at_ms", { mode: "number" }).notNull().default(
      sql`(EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT`
    ),
  },
  (table) => [
    index("idx_carts_store_kiosk")
      .on(table.storeId, table.kioskId, table.status)
      .where(sql`${table.status} = 'active'`),
    index("idx_carts_session")
      .on(table.sessionId, table.status)
      .where(sql`${table.status} = 'active'`),
    index("idx_carts_expires")
      .on(table.expiresAt)
      .where(sql`${table.status} = 'active'`),
  ]
);

export const cartLines = pgTable(
  "cart_lines",
  {
    lineId: uuid("line_id").primaryKey().defaultRandom(),
    cartId: uuid("cart_id")
      .notNull()
      .references(() => carts.cartId, { onDelete: "cascade" }),
    menuItemId: uuid("menu_item_id")
      .notNull()
      .references(() => items.itemId),
    nameSnapshot: varchar("name_snapshot", { length: 255 }).notNull(),
    unitPriceCentsSnapshot: integer("unit_price_cents_snapshot").notNull(),
    quantity: integer("quantity").notNull(),
    modifiers: jsonb("modifiers").notNull().default("[]"),
    addedAtMs: bigint("added_at_ms", { mode: "number" }).notNull(),
  },
  (table) => [index("idx_cart_lines_cart").on(table.cartId)]
);

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

export const orders = pgTable(
  "orders",
  {
    orderId: uuid("order_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    cartId: uuid("cart_id")
      .notNull()
      .references(() => carts.cartId),
    orderNumber: varchar("order_number", { length: 16 }).notNull().unique(),
    status: orderStatusEnum("status").notNull().default("pending"),
    subtotalCents: integer("subtotal_cents").notNull().default(0),
    taxCents: integer("tax_cents").notNull().default(0),
    discountCents: integer("discount_cents").notNull().default(0),
    totalCents: integer("total_cents").notNull().default(0),
    itemsJson: jsonb("items_json").notNull().default("[]"),
    taxBreakdownJson: jsonb("tax_breakdown_json"),
    metadata: jsonb("metadata"),
    paidAt: timestamp("paid_at", { withTimezone: true }),
    fulfilledAt: timestamp("fulfilled_at", { withTimezone: true }),
    cancelledAt: timestamp("cancelled_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_orders_store").on(table.storeId, table.createdAt),
    index("idx_orders_kiosk").on(table.kioskId, table.createdAt),
    index("idx_orders_number").on(table.orderNumber),
    index("idx_orders_status").on(table.storeId, table.status, table.createdAt),
  ]
);

export const orderItems = pgTable(
  "order_items",
  {
    orderItemId: uuid("order_item_id").primaryKey().defaultRandom(),
    orderId: uuid("order_id")
      .notNull()
      .references(() => orders.orderId),
    itemId: uuid("item_id")
      .notNull()
      .references(() => items.itemId),
    nameSnapshot: varchar("name_snapshot", { length: 255 }).notNull(),
    priceCentsSnapshot: integer("price_cents_snapshot").notNull(),
    quantity: integer("quantity").notNull(),
    modifiersJson: jsonb("modifiers_json").notNull().default("[]"),
    lineTotalCents: integer("line_total_cents").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [index("idx_order_items_order").on(table.orderId)]
);

// ---------------------------------------------------------------------------
// Payments / Refunds / Offline Tokens
// ---------------------------------------------------------------------------

export const payments = pgTable(
  "payments",
  {
    paymentId: uuid("payment_id").primaryKey().defaultRandom(),
    orderId: uuid("order_id")
      .notNull()
      .references(() => orders.orderId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    idempotencyKey: uuid("idempotency_key").notNull().unique(),
    amountCents: integer("amount_cents").notNull(),
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    method: paymentMethodEnum("method").notNull(),
    status: paymentStatusEnum("status").notNull().default("pending"),
    verifoneToken: varchar("verifone_token", { length: 255 }),
    verifoneAuthCode: varchar("verifone_auth_code", { length: 16 }),
    cardBrand: varchar("card_brand", { length: 16 }),
    cardLastFour: varchar("card_last_four", { length: 4 }),
    declineReason: varchar("decline_reason", { length: 255 }),
    receiptText: text("receipt_text"),
    isOfflineToken: boolean("is_offline_token").notNull().default(false),
    offlineTokenHmac: varchar("offline_token_hmac", { length: 64 }),
    syncedAt: timestamp("synced_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_payments_order").on(table.orderId),
    index("idx_payments_kiosk").on(table.kioskId, table.createdAt),
    index("idx_payments_offline")
      .on(table.isOfflineToken, table.syncedAt)
      .where(sql`${table.isOfflineToken} = TRUE AND ${table.syncedAt} IS NULL`),
    index("idx_payments_idempotency").on(table.idempotencyKey),
  ]
);

export const refunds = pgTable(
  "refunds",
  {
    refundId: uuid("refund_id").primaryKey().defaultRandom(),
    paymentId: uuid("payment_id")
      .notNull()
      .references(() => payments.paymentId),
    orderId: uuid("order_id")
      .notNull()
      .references(() => orders.orderId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    amountCents: integer("amount_cents").notNull(),
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    reason: varchar("reason", { length: 255 }).notNull(),
    status: refundStatusEnum("status").notNull().default("pending"),
    verifoneReference: varchar("verifone_reference", { length: 255 }),
    processedBy: uuid("processed_by").references(() => employees.employeeId),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_refunds_payment").on(table.paymentId),
    index("idx_refunds_order").on(table.orderId, table.createdAt),
  ]
);

export const offlineTokens = pgTable(
  "offline_tokens",
  {
    tokenId: uuid("token_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    cartId: uuid("cart_id").notNull(),
    amountCents: integer("amount_cents").notNull(),
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    method: varchar("method", { length: 16 }).notNull(),
    verifoneOpaqueToken: varchar("verifone_opaque_token", { length: 255 }).notNull(),
    hmacSignature: varchar("hmac_signature", { length: 64 }).notNull(),
    expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
    settledAt: timestamp("settled_at", { withTimezone: true }),
    settlementResult: jsonb("settlement_result"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_offline_tokens_store")
      .on(table.storeId, table.settledAt)
      .where(sql`${table.settledAt} IS NULL`),
    index("idx_offline_tokens_expires")
      .on(table.expiresAt)
      .where(sql`${table.settledAt} IS NULL`),
  ]
);

// ---------------------------------------------------------------------------
// Employees / Users / RBAC
// ---------------------------------------------------------------------------

export const employees = pgTable(
  "employees",
  {
    employeeId: uuid("employee_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    name: varchar("name", { length: 128 }).notNull(),
    email: varchar("email", { length: 255 }).notNull(),
    role: employeeRoleEnum("role").notNull().default("cashier"),
    biometricHash: varchar("biometric_hash", { length: 64 }),
    webauthnCredentialId: varchar("webauthn_credential_id", { length: 255 }),
    webauthnPublicKey: bytea("webauthn_public_key"),
    isActive: boolean("is_active").notNull().default(true),
    lastLoginAt: timestamp("last_login_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    index("idx_employees_store")
      .on(table.storeId, table.isActive)
      .where(sql`${table.deletedAt} IS NULL`),
    uniqueIndex("idx_employees_email")
      .on(table.email)
      .where(sql`${table.deletedAt} IS NULL`),
  ]
);

export const roles = pgTable(
  "roles",
  {
    roleId: uuid("role_id").primaryKey().defaultRandom(),
    tenantId: uuid("tenant_id")
      .notNull()
      .references(() => tenants.tenantId),
    name: varchar("name", { length: 64 }).notNull(),
    description: text("description"),
    isSystem: boolean("is_system").notNull().default(false),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    unique("roles_tenant_name_unique").on(table.tenantId, table.name),
    index("idx_roles_tenant").on(table.tenantId),
  ]
);

export const permissions = pgTable(
  "permissions",
  {
    permissionId: uuid("permission_id").primaryKey().defaultRandom(),
    resource: varchar("resource", { length: 64 }).notNull(),
    action: varchar("action", { length: 64 }).notNull(),
    description: text("description"),
  },
  (table) => [unique("permissions_resource_action_unique").on(table.resource, table.action)]
);

export const rolePermissions = pgTable(
  "role_permissions",
  {
    roleId: uuid("role_id")
      .notNull()
      .references(() => roles.roleId, { onDelete: "cascade" }),
    permissionId: uuid("permission_id")
      .notNull()
      .references(() => permissions.permissionId, { onDelete: "cascade" }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [primaryKey({ columns: [table.roleId, table.permissionId] })]
);

export const users = pgTable(
  "users",
  {
    userId: uuid("user_id").primaryKey().defaultRandom(),
    tenantId: uuid("tenant_id")
      .notNull()
      .references(() => tenants.tenantId),
    email: varchar("email", { length: 255 }).notNull(),
    name: varchar("name", { length: 128 }).notNull(),
    roleId: uuid("role_id")
      .notNull()
      .references(() => roles.roleId),
    isActive: boolean("is_active").notNull().default(true),
    webauthnCredentialId: varchar("webauthn_credential_id", { length: 255 }),
    webauthnPublicKey: bytea("webauthn_public_key"),
    lastLoginAt: timestamp("last_login_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
    deletedAt: timestamp("deleted_at", { withTimezone: true }),
  },
  (table) => [
    unique("users_tenant_email_unique").on(table.tenantId, table.email),
    index("idx_users_tenant")
      .on(table.tenantId, table.deletedAt)
      .where(sql`${table.deletedAt} IS NULL`),
    index("idx_users_role").on(table.roleId),
  ]
);

// ---------------------------------------------------------------------------
// Audit / Event Store / Outbox / Sync / Analytics
// ---------------------------------------------------------------------------

export const auditLogs = pgTable(
  "audit_logs",
  {
    auditId: bigint("audit_id", { mode: "number" }).notNull(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    tenantId: uuid("tenant_id").references(() => tenants.tenantId),
    laneId: uuid("lane_id").references(() => lanes.laneId),
    kioskId: uuid("kiosk_id").references(() => kiosks.kioskId),
    employeeId: uuid("employee_id").references(() => employees.employeeId),
    userId: uuid("user_id").references(() => users.userId),
    eventType: auditEventTypeEnum("event_type").notNull(),
    entityType: varchar("entity_type", { length: 32 }).notNull(),
    entityId: uuid("entity_id").notNull(),
    payloadJson: jsonb("payload_json").notNull(),
    previousHash: varchar("previous_hash", { length: 64 }).notNull(),
    currentHash: varchar("current_hash", { length: 64 }).notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    primaryKey({ columns: [table.auditId, table.createdAt] }),
    index("idx_audit_logs_store").on(table.storeId, table.createdAt),
    index("idx_audit_logs_entity").on(table.entityType, table.entityId, table.createdAt),
    index("idx_audit_logs_event").on(table.eventType, table.createdAt),
  ]
);

export const eventStore = pgTable(
  "event_store",
  {
    eventId: uuid("event_id").primaryKey().defaultRandom(),
    eventSchema: varchar("event_schema", { length: 64 }).notNull(),
    aggregateType: varchar("aggregate_type", { length: 64 }).notNull(),
    aggregateId: uuid("aggregate_id").notNull(),
    sequenceNumber: bigint("sequence_number", { mode: "number" }).notNull(),
    payload: jsonb("payload").notNull(),
    metadata: jsonb("metadata").notNull().default("{}"),
    occurredAt: timestamp("occurred_at", { withTimezone: true }).notNull().defaultNow(),
    recordedAt: timestamp("recorded_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    unique("event_store_aggregate_unique").on(
      table.aggregateType,
      table.aggregateId,
      table.sequenceNumber
    ),
    index("idx_event_store_aggregate").on(
      table.aggregateType,
      table.aggregateId,
      table.sequenceNumber
    ),
    index("idx_event_store_occurred").on(table.occurredAt),
  ]
);

export const outboxEvents = pgTable(
  "outbox_events",
  {
    eventId: uuid("event_id").primaryKey().defaultRandom(),
    aggregateType: varchar("aggregate_type", { length: 64 }).notNull(),
    aggregateId: uuid("aggregate_id").notNull(),
    eventType: varchar("event_type", { length: 128 }).notNull(),
    payload: jsonb("payload").notNull(),
    occurredAtMs: bigint("occurred_at_ms", { mode: "number" }).notNull(),
    published: boolean("published").notNull().default(false),
    publishedAtMs: bigint("published_at_ms", { mode: "number" }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_outbox_unpublished")
      .on(table.published, table.occurredAtMs)
      .where(sql`${table.published} = FALSE`),
    index("idx_outbox_aggregate").on(table.aggregateType, table.aggregateId, table.occurredAtMs),
  ]
);

export const syncEvents = pgTable(
  "sync_events",
  {
    syncEventId: uuid("sync_event_id").primaryKey().defaultRandom(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id")
      .notNull()
      .references(() => kiosks.kioskId),
    eventType: syncEventTypeEnum("event_type").notNull(),
    payloadJson: jsonb("payload_json").notNull(),
    vectorClock: jsonb("vector_clock").notNull(),
    processedAt: timestamp("processed_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    index("idx_sync_events_store").on(table.storeId, table.createdAt),
    index("idx_sync_events_unprocessed")
      .on(table.storeId, table.processedAt)
      .where(sql`${table.processedAt} IS NULL`),
  ]
);

export const analyticsEvents = pgTable(
  "analytics_events",
  {
    analyticsId: bigint("analytics_id", { mode: "number" }).notNull(),
    storeId: uuid("store_id")
      .notNull()
      .references(() => stores.storeId),
    kioskId: uuid("kiosk_id").references(() => kiosks.kioskId),
    eventType: varchar("event_type", { length: 32 }).notNull(),
    sessionId: uuid("session_id"),
    customerHash: varchar("customer_hash", { length: 64 }),
    itemId: uuid("item_id").references(() => items.itemId),
    categoryId: uuid("category_id").references(() => categories.categoryId),
    quantity: integer("quantity"),
    amountCents: integer("amount_cents"),
    durationMs: integer("duration_ms"),
    metadata: jsonb("metadata"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => [
    primaryKey({ columns: [table.analyticsId, table.createdAt] }),
    index("idx_analytics_store").on(table.storeId, table.eventType, table.createdAt),
    index("idx_analytics_item")
      .on(table.itemId, table.createdAt)
      .where(sql`${table.itemId} IS NOT NULL`),
  ]
);

// ---------------------------------------------------------------------------
// Relations
// ---------------------------------------------------------------------------

export const tenantsRelations = relations(tenants, ({ many }) => ({
  locations: many(locations),
  stores: many(stores),
  kiosks: many(kiosks),
  roles: many(roles),
  users: many(users),
}));

export const locationsRelations = relations(locations, ({ one, many }) => ({
  tenant: one(tenants, {
    fields: [locations.tenantId],
    references: [tenants.tenantId],
  }),
  lanes: many(lanes),
  stores: many(stores),
}));

export const lanesRelations = relations(lanes, ({ one, many }) => ({
  location: one(locations, {
    fields: [lanes.locationId],
    references: [locations.locationId],
  }),
  kiosks: many(kiosks),
}));

export const storesRelations = relations(stores, ({ one, many }) => ({
  tenant: one(tenants, {
    fields: [stores.tenantId],
    references: [tenants.tenantId],
  }),
  location: one(locations, {
    fields: [stores.locationId],
    references: [locations.locationId],
  }),
  kiosks: many(kiosks),
  categories: many(categories),
  items: many(items),
  inventory: many(inventory),
  carts: many(carts),
  orders: many(orders),
  employees: many(employees),
  offlineTokens: many(offlineTokens),
  syncEvents: many(syncEvents),
  analyticsEvents: many(analyticsEvents),
}));

export const kiosksRelations = relations(kiosks, ({ one, many }) => ({
  store: one(stores, {
    fields: [kiosks.storeId],
    references: [stores.storeId],
  }),
  lane: one(lanes, {
    fields: [kiosks.laneId],
    references: [lanes.laneId],
  }),
  tenant: one(tenants, {
    fields: [kiosks.tenantId],
    references: [tenants.tenantId],
  }),
  carts: many(carts),
  orders: many(orders),
  payments: many(payments),
  refunds: many(refunds),
  inventoryTransactions: many(inventoryTransactions),
  inventoryReservations: many(inventoryReservations),
  syncEvents: many(syncEvents),
  analyticsEvents: many(analyticsEvents),
  offlineTokens: many(offlineTokens),
}));

export const categoriesRelations = relations(categories, ({ one, many }) => ({
  store: one(stores, {
    fields: [categories.storeId],
    references: [stores.storeId],
  }),
  parent: one(categories, {
    fields: [categories.parentId],
    references: [categories.categoryId],
    relationName: "category_parent",
  }),
  children: many(categories, {
    relationName: "category_parent",
  }),
  items: many(items),
  analyticsEvents: many(analyticsEvents),
}));

export const itemsRelations = relations(items, ({ one, many }) => ({
  store: one(stores, {
    fields: [items.storeId],
    references: [stores.storeId],
  }),
  category: one(categories, {
    fields: [items.categoryId],
    references: [categories.categoryId],
  }),
  inventory: many(inventory),
  inventoryTransactions: many(inventoryTransactions),
  inventoryReservations: many(inventoryReservations),
  orderItems: many(orderItems),
  cartLines: many(cartLines),
  modifierGroups: many(itemModifierGroups),
  analyticsEvents: many(analyticsEvents),
}));

export const modifierGroupsRelations = relations(modifierGroups, ({ one, many }) => ({
  store: one(stores, {
    fields: [modifierGroups.storeId],
    references: [stores.storeId],
  }),
  options: many(modifierOptions),
  items: many(itemModifierGroups),
}));

export const modifierOptionsRelations = relations(modifierOptions, ({ one }) => ({
  modifierGroup: one(modifierGroups, {
    fields: [modifierOptions.modifierGroupId],
    references: [modifierGroups.modifierGroupId],
  }),
}));

export const itemModifierGroupsRelations = relations(itemModifierGroups, ({ one }) => ({
  item: one(items, {
    fields: [itemModifierGroups.itemId],
    references: [items.itemId],
  }),
  modifierGroup: one(modifierGroups, {
    fields: [itemModifierGroups.modifierGroupId],
    references: [modifierGroups.modifierGroupId],
  }),
}));

export const inventoryRelations = relations(inventory, ({ one, many }) => ({
  store: one(stores, {
    fields: [inventory.storeId],
    references: [stores.storeId],
  }),
  item: one(items, {
    fields: [inventory.itemId],
    references: [items.itemId],
  }),
  transactions: many(inventoryTransactions),
  reservations: many(inventoryReservations),
}));

export const inventoryTransactionsRelations = relations(inventoryTransactions, ({ one }) => ({
  store: one(stores, {
    fields: [inventoryTransactions.storeId],
    references: [stores.storeId],
  }),
  item: one(items, {
    fields: [inventoryTransactions.itemId],
    references: [items.itemId],
  }),
  kiosk: one(kiosks, {
    fields: [inventoryTransactions.kioskId],
    references: [kiosks.kioskId],
  }),
  employee: one(employees, {
    fields: [inventoryTransactions.employeeId],
    references: [employees.employeeId],
  }),
}));

export const inventoryReservationsRelations = relations(inventoryReservations, ({ one }) => ({
  store: one(stores, {
    fields: [inventoryReservations.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [inventoryReservations.kioskId],
    references: [kiosks.kioskId],
  }),
  item: one(items, {
    fields: [inventoryReservations.itemId],
    references: [items.itemId],
  }),
  cart: one(carts, {
    fields: [inventoryReservations.cartId],
    references: [carts.cartId],
  }),
}));

export const cartsRelations = relations(carts, ({ one, many }) => ({
  store: one(stores, {
    fields: [carts.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [carts.kioskId],
    references: [kiosks.kioskId],
  }),
  lines: many(cartLines),
  orders: many(orders),
  inventoryReservations: many(inventoryReservations),
}));

export const cartLinesRelations = relations(cartLines, ({ one }) => ({
  cart: one(carts, {
    fields: [cartLines.cartId],
    references: [carts.cartId],
  }),
  menuItem: one(items, {
    fields: [cartLines.menuItemId],
    references: [items.itemId],
  }),
}));

export const ordersRelations = relations(orders, ({ one, many }) => ({
  store: one(stores, {
    fields: [orders.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [orders.kioskId],
    references: [kiosks.kioskId],
  }),
  cart: one(carts, {
    fields: [orders.cartId],
    references: [carts.cartId],
  }),
  items: many(orderItems),
  payments: many(payments),
  refunds: many(refunds),
}));

export const orderItemsRelations = relations(orderItems, ({ one }) => ({
  order: one(orders, {
    fields: [orderItems.orderId],
    references: [orders.orderId],
  }),
  item: one(items, {
    fields: [orderItems.itemId],
    references: [items.itemId],
  }),
}));

export const paymentsRelations = relations(payments, ({ one, many }) => ({
  order: one(orders, {
    fields: [payments.orderId],
    references: [orders.orderId],
  }),
  kiosk: one(kiosks, {
    fields: [payments.kioskId],
    references: [kiosks.kioskId],
  }),
  refunds: many(refunds),
}));

export const refundsRelations = relations(refunds, ({ one }) => ({
  payment: one(payments, {
    fields: [refunds.paymentId],
    references: [payments.paymentId],
  }),
  order: one(orders, {
    fields: [refunds.orderId],
    references: [orders.orderId],
  }),
  kiosk: one(kiosks, {
    fields: [refunds.kioskId],
    references: [kiosks.kioskId],
  }),
  processor: one(employees, {
    fields: [refunds.processedBy],
    references: [employees.employeeId],
  }),
}));

export const offlineTokensRelations = relations(offlineTokens, ({ one }) => ({
  store: one(stores, {
    fields: [offlineTokens.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [offlineTokens.kioskId],
    references: [kiosks.kioskId],
  }),
}));

export const employeesRelations = relations(employees, ({ one, many }) => ({
  store: one(stores, {
    fields: [employees.storeId],
    references: [stores.storeId],
  }),
  inventoryTransactions: many(inventoryTransactions),
  refunds: many(refunds),
  auditLogs: many(auditLogs),
}));

export const rolesRelations = relations(roles, ({ one, many }) => ({
  tenant: one(tenants, {
    fields: [roles.tenantId],
    references: [tenants.tenantId],
  }),
  permissions: many(rolePermissions),
  users: many(users),
}));

export const permissionsRelations = relations(permissions, ({ many }) => ({
  roles: many(rolePermissions),
}));

export const rolePermissionsRelations = relations(rolePermissions, ({ one }) => ({
  role: one(roles, {
    fields: [rolePermissions.roleId],
    references: [roles.roleId],
  }),
  permission: one(permissions, {
    fields: [rolePermissions.permissionId],
    references: [permissions.permissionId],
  }),
}));

export const usersRelations = relations(users, ({ one, many }) => ({
  tenant: one(tenants, {
    fields: [users.tenantId],
    references: [tenants.tenantId],
  }),
  role: one(roles, {
    fields: [users.roleId],
    references: [roles.roleId],
  }),
  auditLogs: many(auditLogs),
}));

export const auditLogsRelations = relations(auditLogs, ({ one }) => ({
  store: one(stores, {
    fields: [auditLogs.storeId],
    references: [stores.storeId],
  }),
  tenant: one(tenants, {
    fields: [auditLogs.tenantId],
    references: [tenants.tenantId],
  }),
  lane: one(lanes, {
    fields: [auditLogs.laneId],
    references: [lanes.laneId],
  }),
  kiosk: one(kiosks, {
    fields: [auditLogs.kioskId],
    references: [kiosks.kioskId],
  }),
  employee: one(employees, {
    fields: [auditLogs.employeeId],
    references: [employees.employeeId],
  }),
  user: one(users, {
    fields: [auditLogs.userId],
    references: [users.userId],
  }),
}));

export const eventStoreRelations = relations(eventStore, () => ({}));

export const outboxEventsRelations = relations(outboxEvents, () => ({}));

export const syncEventsRelations = relations(syncEvents, ({ one }) => ({
  store: one(stores, {
    fields: [syncEvents.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [syncEvents.kioskId],
    references: [kiosks.kioskId],
  }),
}));

export const analyticsEventsRelations = relations(analyticsEvents, ({ one }) => ({
  store: one(stores, {
    fields: [analyticsEvents.storeId],
    references: [stores.storeId],
  }),
  kiosk: one(kiosks, {
    fields: [analyticsEvents.kioskId],
    references: [kiosks.kioskId],
  }),
  item: one(items, {
    fields: [analyticsEvents.itemId],
    references: [items.itemId],
  }),
  category: one(categories, {
    fields: [analyticsEvents.categoryId],
    references: [categories.categoryId],
  }),
}));

// ---------------------------------------------------------------------------
// Inferred types
// ---------------------------------------------------------------------------

export type Tenant = typeof tenants.$inferSelect;
export type NewTenant = typeof tenants.$inferInsert;
export type Location = typeof locations.$inferSelect;
export type NewLocation = typeof locations.$inferInsert;
export type Lane = typeof lanes.$inferSelect;
export type NewLane = typeof lanes.$inferInsert;

export type Store = typeof stores.$inferSelect;
export type NewStore = typeof stores.$inferInsert;
export type Kiosk = typeof kiosks.$inferSelect;
export type NewKiosk = typeof kiosks.$inferInsert;

export type Category = typeof categories.$inferSelect;
export type NewCategory = typeof categories.$inferInsert;
export type Item = typeof items.$inferSelect;
export type NewItem = typeof items.$inferInsert;
export type ModifierGroup = typeof modifierGroups.$inferSelect;
export type NewModifierGroup = typeof modifierGroups.$inferInsert;
export type ModifierOption = typeof modifierOptions.$inferSelect;
export type NewModifierOption = typeof modifierOptions.$inferInsert;
export type ItemModifierGroup = typeof itemModifierGroups.$inferSelect;
export type NewItemModifierGroup = typeof itemModifierGroups.$inferInsert;

export type Inventory = typeof inventory.$inferSelect;
export type NewInventory = typeof inventory.$inferInsert;
export type InventoryTransaction = typeof inventoryTransactions.$inferSelect;
export type NewInventoryTransaction = typeof inventoryTransactions.$inferInsert;
export type InventoryReservation = typeof inventoryReservations.$inferSelect;
export type NewInventoryReservation = typeof inventoryReservations.$inferInsert;

export type Cart = typeof carts.$inferSelect;
export type NewCart = typeof carts.$inferInsert;
export type CartLine = typeof cartLines.$inferSelect;
export type NewCartLine = typeof cartLines.$inferInsert;

export type Order = typeof orders.$inferSelect;
export type NewOrder = typeof orders.$inferInsert;
export type OrderItem = typeof orderItems.$inferSelect;
export type NewOrderItem = typeof orderItems.$inferInsert;

export type Payment = typeof payments.$inferSelect;
export type NewPayment = typeof payments.$inferInsert;
export type Refund = typeof refunds.$inferSelect;
export type NewRefund = typeof refunds.$inferInsert;
export type OfflineToken = typeof offlineTokens.$inferSelect;
export type NewOfflineToken = typeof offlineTokens.$inferInsert;

export type Employee = typeof employees.$inferSelect;
export type NewEmployee = typeof employees.$inferInsert;
export type Role = typeof roles.$inferSelect;
export type NewRole = typeof roles.$inferInsert;
export type Permission = typeof permissions.$inferSelect;
export type NewPermission = typeof permissions.$inferInsert;
export type RolePermission = typeof rolePermissions.$inferSelect;
export type NewRolePermission = typeof rolePermissions.$inferInsert;
export type User = typeof users.$inferSelect;
export type NewUser = typeof users.$inferInsert;

export type AuditLog = typeof auditLogs.$inferSelect;
export type NewAuditLog = typeof auditLogs.$inferInsert;
export type EventStoreRecord = typeof eventStore.$inferSelect;
export type NewEventStoreRecord = typeof eventStore.$inferInsert;
export type OutboxEvent = typeof outboxEvents.$inferSelect;
export type NewOutboxEvent = typeof outboxEvents.$inferInsert;
export type SyncEvent = typeof syncEvents.$inferSelect;
export type NewSyncEvent = typeof syncEvents.$inferInsert;
export type AnalyticsEvent = typeof analyticsEvents.$inferSelect;
export type NewAnalyticsEvent = typeof analyticsEvents.$inferInsert;
