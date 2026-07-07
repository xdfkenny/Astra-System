import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_EMPLOYEES_AND_ROLES } from "../graphql/queries";
import type { AdminListResponse, EmployeeWithRole } from "../graphql/types";
import type { Role } from "@astra/shared-types";
import { formatDate } from "../lib/format";

export function EmployeesRoles(): React.JSX.Element {
  const { data, loading, error } = useQuery<{
    employees: AdminListResponse<EmployeeWithRole>;
    roles: AdminListResponse<Role>;
  }>(LIST_EMPLOYEES_AND_ROLES);

  return (
    <Layout title="Employees / Roles">
      <div className="flex flex-col gap-6">
        {loading && <Spinner aria-label="Loading employees" />}
        {error && <p className="text-error">Failed to load employees.</p>}

        {data && (
          <>
            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Employees</h2>
              <DataTable
                rows={data.employees.items}
                keyExtractor={(row) => row.employeeId}
                columns={[
                  { key: "name", header: "Name", render: (row) => row.name },
                  { key: "email", header: "Email", render: (row) => row.email },
                  { key: "role", header: "Role", render: (row) => row.roleName },
                  {
                    key: "status",
                    header: "Status",
                    render: (row) => (
                      <StatusBadge status={row.isActive ? "healthy" : "offline"}>
                        {row.isActive ? "Active" : "Inactive"}
                      </StatusBadge>
                    ),
                  },
                  {
                    key: "lastLogin",
                    header: "Last Login",
                    render: (row) => (row.lastLoginAt ? formatDate(row.lastLoginAt) : "Never"),
                  },
                ]}
              />
            </section>

            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Roles</h2>
              <DataTable
                rows={data.roles.items}
                keyExtractor={(row) => row.roleId}
                columns={[
                  { key: "name", header: "Role", render: (row) => row.name },
                  { key: "description", header: "Description", render: (row) => row.description ?? "—" },
                  {
                    key: "system",
                    header: "System",
                    render: (row) => (
                      <StatusBadge status={row.isSystem ? "healthy" : "offline"}>
                        {row.isSystem ? "System" : "Custom"}
                      </StatusBadge>
                    ),
                  },
                  { key: "updated", header: "Updated", render: (row) => formatDate(row.updatedAt) },
                ]}
              />
            </section>
          </>
        )}
      </div>
    </Layout>
  );
}
