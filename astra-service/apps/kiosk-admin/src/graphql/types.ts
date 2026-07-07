import type {
  AuditLog,
  Category,
  Employee,
  Inventory,
  Item,
  Lane,
  Location,
  ModifierGroup,
  ModifierOption,
  Order,
  Payment,
  Refund,
} from "@astra/shared-types";

export interface DashboardKpis {
  readonly totalRevenueCents: number;
  readonly orderCount: number;
  readonly activeKiosks: number;
  readonly alerts: number;
  readonly revenueTrend: number;
  readonly orderTrend: number;
}

export interface LocationWithLanes extends Location {
  readonly lanes: readonly Lane[];
}

export interface KioskHealth {
  readonly kioskId: string;
  readonly displayName: string;
  readonly syncStatus: string;
  readonly lastSeenAt: string | null;
  readonly isLeader: boolean;
}

export interface MenuCategory extends Category {
  readonly items: readonly Item[];
}

export interface MenuModifierGroup extends ModifierGroup {
  readonly options: readonly ModifierOption[];
}

export interface FullMenu {
  readonly categories: readonly MenuCategory[];
  readonly modifierGroups: readonly MenuModifierGroup[];
}

export interface InventoryWithItem extends Inventory {
  readonly itemName: string;
  readonly itemSku: string | null;
}

export interface OrderWithKiosk extends Order {
  readonly kioskDisplayName: string;
}

export interface PaymentWithOrder extends Payment {
  readonly orderNumber: string;
}

export interface RefundWithPayment extends Refund {
  readonly paymentAmountCents: number;
}

export interface EmployeeWithRole extends Employee {
  readonly roleName: string;
}

export interface AuditLogWithActor extends AuditLog {
  readonly actorName: string | null;
}

export interface AdminListResponse<T> {
  readonly items: readonly T[];
  readonly totalCount: number;
}
