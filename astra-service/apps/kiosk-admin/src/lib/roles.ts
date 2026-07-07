import type { EmployeeRole } from "@astra/shared-types";

export type AdminRole = EmployeeRole | "readonly";

export interface Permission {
  readonly resource: string;
  readonly action: "view" | "create" | "update" | "delete";
}

const ROLE_PERMISSIONS: Record<AdminRole, readonly Permission[]> = {
  admin: [
    { resource: "*", action: "view" },
    { resource: "*", action: "create" },
    { resource: "*", action: "update" },
    { resource: "*", action: "delete" },
  ],
  manager: [
    { resource: "dashboard", action: "view" },
    { resource: "locations", action: "view" },
    { resource: "lanes", action: "view" },
    { resource: "kiosks", action: "view" },
    { resource: "menu", action: "view" },
    { resource: "menu", action: "update" },
    { resource: "inventory", action: "view" },
    { resource: "inventory", action: "update" },
    { resource: "orders", action: "view" },
    { resource: "orders", action: "update" },
    { resource: "payments", action: "view" },
    { resource: "refunds", action: "create" },
    { resource: "employees", action: "view" },
    { resource: "audit", action: "view" },
  ],
  supervisor: [
    { resource: "dashboard", action: "view" },
    { resource: "kiosks", action: "view" },
    { resource: "orders", action: "view" },
    { resource: "orders", action: "update" },
    { resource: "payments", action: "view" },
    { resource: "refunds", action: "create" },
    { resource: "inventory", action: "view" },
  ],
  cashier: [
    { resource: "dashboard", action: "view" },
    { resource: "orders", action: "view" },
    { resource: "payments", action: "view" },
  ],
  readonly: [
    { resource: "dashboard", action: "view" },
    { resource: "kiosks", action: "view" },
    { resource: "orders", action: "view" },
  ],
};

export function canAccess(role: AdminRole, resource: string, action: Permission["action"] = "view"): boolean {
  const perms = ROLE_PERMISSIONS[role];
  return perms.some(
    (p) =>
      (p.resource === "*" || p.resource === resource) &&
      (p.action === action),
  );
}

export function requiresRole(role: AdminRole, minRole: AdminRole): boolean {
  const hierarchy: readonly AdminRole[] = ["readonly", "cashier", "supervisor", "manager", "admin"];
  const roleIndex = hierarchy.indexOf(role);
  const minIndex = hierarchy.indexOf(minRole);
  return roleIndex >= minIndex;
}
