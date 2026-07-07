import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { EmployeeRole } from "@astra/shared-types";
import type { AdminRole } from "../lib/roles";

interface AuthContextValue {
  readonly role: AdminRole;
  readonly employeeId: string | null;
  readonly setRole: (role: AdminRole) => void;
  readonly setEmployeeId: (id: string | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export interface AuthProviderProps {
  readonly children: ReactNode;
  readonly initialRole?: AdminRole;
}

export function AuthProvider({ children, initialRole = "readonly" }: AuthProviderProps): React.JSX.Element {
  const [role, setRole] = useState<AdminRole>(initialRole);
  const [employeeId, setEmployeeId] = useState<string | null>(null);

  const setRoleWrapped = useCallback((next: AdminRole) => { setRole(next); }, []);
  const setEmployeeIdWrapped = useCallback((id: string | null) => { setEmployeeId(id); }, []);

  const value = useMemo(
    () => ({
      role,
      employeeId,
      setRole: setRoleWrapped,
      setEmployeeId: setEmployeeIdWrapped,
    }),
    [role, employeeId, setRoleWrapped, setEmployeeIdWrapped],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}

export function roleFromEmployeeRole(role: EmployeeRole): AdminRole {
  return role;
}
