import { describe, expect, it } from "vitest";
import { canAccess, requiresRole } from "../lib/roles";

describe("canAccess", () => {
  it("grants admins access to every resource", () => {
    expect(canAccess("admin", "audit", "delete")).toBe(true);
    expect(canAccess("admin", "payments", "create")).toBe(true);
  });

  it("denies cashiers access to sensitive resources", () => {
    expect(canAccess("cashier", "employees", "view")).toBe(false);
    expect(canAccess("cashier", "audit", "view")).toBe(false);
  });

  it("allows cashiers to view dashboard and orders", () => {
    expect(canAccess("cashier", "dashboard")).toBe(true);
    expect(canAccess("cashier", "orders", "view")).toBe(true);
  });
});

describe("requiresRole", () => {
  it("returns true when the user meets the minimum role", () => {
    expect(requiresRole("manager", "supervisor")).toBe(true);
    expect(requiresRole("admin", "manager")).toBe(true);
  });

  it("returns false when the user is below the minimum role", () => {
    expect(requiresRole("cashier", "manager")).toBe(false);
  });

  it("returns true for the same role", () => {
    expect(requiresRole("supervisor", "supervisor")).toBe(true);
  });
});
