import { describe, expect, it } from "vitest";
import { getTableName } from "drizzle-orm/table";
import {
  analyticsEvents,
  auditLogs,
  cartLines,
  carts,
  categories,
  employees,
  eventStore,
  inventory,
  inventoryReservations,
  inventoryTransactions,
  itemModifierGroups,
  items,
  kiosks,
  lanes,
  locations,
  modifierGroups,
  modifierOptions,
  offlineTokens,
  orderItems,
  orders,
  outboxEvents,
  payments,
  permissions,
  refundStatusEnum,
  refunds,
  rolePermissions,
  roles,
  stores,
  syncEvents,
  tenants,
  users,
} from "./drizzle.js";

describe("drizzle schema definitions", () => {
  it("exports all expected tables", () => {
    expect(getTableName(tenants)).toBe("tenants");
    expect(getTableName(locations)).toBe("locations");
    expect(getTableName(lanes)).toBe("lanes");
    expect(getTableName(stores)).toBe("stores");
    expect(getTableName(kiosks)).toBe("kiosks");
    expect(getTableName(categories)).toBe("categories");
    expect(getTableName(items)).toBe("items");
    expect(getTableName(modifierGroups)).toBe("modifier_groups");
    expect(getTableName(modifierOptions)).toBe("modifier_options");
    expect(getTableName(itemModifierGroups)).toBe("item_modifier_groups");
    expect(getTableName(inventory)).toBe("inventory");
    expect(getTableName(inventoryTransactions)).toBe("inventory_transactions");
    expect(getTableName(inventoryReservations)).toBe("inventory_reservations");
    expect(getTableName(carts)).toBe("carts");
    expect(getTableName(cartLines)).toBe("cart_lines");
    expect(getTableName(orders)).toBe("orders");
    expect(getTableName(orderItems)).toBe("order_items");
    expect(getTableName(payments)).toBe("payments");
    expect(getTableName(refunds)).toBe("refunds");
    expect(getTableName(offlineTokens)).toBe("offline_tokens");
    expect(getTableName(employees)).toBe("employees");
    expect(getTableName(users)).toBe("users");
    expect(getTableName(roles)).toBe("roles");
    expect(getTableName(permissions)).toBe("permissions");
    expect(getTableName(rolePermissions)).toBe("role_permissions");
    expect(getTableName(auditLogs)).toBe("audit_logs");
    expect(getTableName(eventStore)).toBe("event_store");
    expect(getTableName(outboxEvents)).toBe("outbox_events");
    expect(getTableName(syncEvents)).toBe("sync_events");
    expect(getTableName(analyticsEvents)).toBe("analytics_events");
  });

  it("defines enum values correctly", () => {
    expect(refundStatusEnum.enumValues).toEqual(["pending", "completed", "failed"]);
  });
});
