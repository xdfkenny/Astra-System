import { Navigate } from "react-router-dom";
import type { ReactNode } from "react";
import { useAuth } from "./AuthProvider";
import { canAccess, type AdminRole } from "../lib/roles";

interface RouteGuardProps {
  readonly resource: string;
  readonly action?: "view" | "create" | "update" | "delete";
  readonly fallback?: ReactNode;
  readonly children: ReactNode;
}

export function RouteGuard({ resource, action = "view", fallback, children }: RouteGuardProps): React.JSX.Element {
  const { role } = useAuth();
  if (!canAccess(role, resource, action)) {
    return fallback ? (
      <>{fallback}</>
    ) : (
      <Navigate to="/" replace state={{ message: "Access denied" }} />
    );
  }
  return <>{children}</>;
}

interface RoleGuardProps {
  readonly minRole: AdminRole;
  readonly children: ReactNode;
}

export function RoleGuard({ minRole, children }: RoleGuardProps): React.JSX.Element {
  const { role } = useAuth();
  const hierarchy: readonly AdminRole[] = ["readonly", "cashier", "supervisor", "manager", "admin"];
  if (hierarchy.indexOf(role) < hierarchy.indexOf(minRole)) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
