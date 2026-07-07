import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { MeshTopologyGraph } from "../components/MeshTopologyGraph";
import { useFleetHealth } from "../hooks/useFleetHealth";
import { LIST_KIOSK_HEALTH } from "../graphql/queries";
import type { AdminListResponse, KioskHealth } from "../graphql/types";
import { formatDate } from "../lib/format";

export function Kiosks(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ kiosks: AdminListResponse<KioskHealth> }>(LIST_KIOSK_HEALTH);
  const fleet = useFleetHealth();

  return (
    <Layout title="Kiosks">
      <div className="flex flex-col gap-6">
        <section className="rounded-lg border border-border bg-surface p-4">
          <h2 className="mb-2 font-heading text-lg font-semibold">Mesh Topology</h2>
          {fleet.data && fleet.data.nodes.length > 0 ? (
            <MeshTopologyGraph nodes={fleet.data.nodes} />
          ) : fleet.isLoading ? (
            <Spinner aria-label="Loading topology" />
          ) : (
            <p className="text-ink-muted">No topology data.</p>
          )}
        </section>

        {loading && <Spinner aria-label="Loading kiosks" />}
        {error && <p className="text-error">Failed to load kiosk list.</p>}
        {data && (
          <DataTable
            rows={data.kiosks.items}
            keyExtractor={(row) => row.kioskId}
            columns={[
              { key: "displayName", header: "Kiosk", render: (row) => row.displayName },
              {
                key: "health",
                header: "Health",
                render: (row) => {
                  const status =
                    row.syncStatus === "online" ? "healthy" : row.syncStatus === "degraded" ? "degraded" : "offline";
                  return <StatusBadge status={status}>{row.syncStatus}</StatusBadge>;
                },
              },
              { key: "role", header: "Role", render: (row) => (row.isLeader ? "Leader" : "Follower") },
              {
                key: "lastSeen",
                header: "Last Seen",
                render: (row) => (row.lastSeenAt ? formatDate(row.lastSeenAt) : "Never"),
              },
              {
                key: "mesh",
                header: "Mesh Peers",
                render: (row) => {
                  const node = fleet.data?.nodes.find((n) => n.kioskId === row.kioskId);
                  return node?.meshPeers.length ?? 0;
                },
              },
            ]}
          />
        )}
      </div>
    </Layout>
  );
}
