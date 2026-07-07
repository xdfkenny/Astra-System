import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { LIST_AUDIT_LOGS } from "../graphql/queries";
import type { AdminListResponse, AuditLogWithActor } from "../graphql/types";
import { formatDate } from "../lib/format";

export function AuditLogs(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ auditLogs: AdminListResponse<AuditLogWithActor> }>(LIST_AUDIT_LOGS);

  return (
    <Layout title="Audit Logs">
      {loading && <Spinner aria-label="Loading audit logs" />}
      {error && <p className="text-error">Failed to load audit logs.</p>}
      {data && (
        <DataTable
          rows={data.auditLogs.items}
          keyExtractor={(row) => String(row.auditId)}
          columns={[
            { key: "time", header: "Time", render: (row) => formatDate(row.createdAt) },
            { key: "event", header: "Event", render: (row) => row.eventType },
            { key: "entity", header: "Entity", render: (row) => `${row.entityType}:${row.entityId.slice(0, 8)}` },
            { key: "actor", header: "Actor", render: (row) => row.actorName ?? row.employeeId ?? row.userId ?? "system" },
            { key: "kiosk", header: "Kiosk", render: (row) => row.kioskId ?? "—" },
            {
              key: "hash",
              header: "Hash",
              render: (row) => (
                <span className="font-mono text-xs text-ink-muted">{row.currentHash.slice(0, 12)}…</span>
              ),
            },
          ]}
        />
      )}
    </Layout>
  );
}
