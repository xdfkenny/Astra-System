import { NavLink } from "react-router-dom";
import { cn } from "@astra/design-system/utils";
import { useAuth } from "./AuthProvider";
import { canAccess } from "../lib/roles";

const ITEMS = [
  { to: "/", label: "Dashboard", resource: "dashboard" },
  { to: "/locations", label: "Locations", resource: "locations" },
  { to: "/lanes", label: "Lanes", resource: "lanes" },
  { to: "/kiosks", label: "Kiosks", resource: "kiosks" },
  { to: "/menu", label: "Menu", resource: "menu" },
  { to: "/inventory", label: "Inventory", resource: "inventory" },
  { to: "/orders", label: "Orders", resource: "orders" },
  { to: "/payments", label: "Payments / Refunds", resource: "payments" },
  { to: "/employees", label: "Employees / Roles", resource: "employees" },
  { to: "/audit", label: "Audit Logs", resource: "audit" },
] as const;

export function Nav(): React.JSX.Element {
  const { role } = useAuth();

  return (
    <nav className="flex flex-col gap-1 p-3">
      {ITEMS.filter((item) => canAccess(role, item.resource)).map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.to === "/"}
          className={({ isActive }) =>
            cn(
              "rounded-md px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-primary text-white"
                : "text-ink-muted hover:bg-surface-sunken hover:text-ink",
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
