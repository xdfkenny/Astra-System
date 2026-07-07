import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_LANES } from "../graphql/queries";
import type { AdminListResponse } from "../graphql/types";
import type { Lane } from "@astra/shared-types";

export function Lanes(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ lanes: AdminListResponse<Lane> }>(LIST_LANES);

  return (
    <Layout title="Lanes">
      {loading && <Spinner aria-label="Loading lanes" />}
      {error && <p className="text-error">Failed to load lanes.</p>}
      {data && (
        <DataTable
          rows={data.lanes.items}
          keyExtractor={(row) => row.laneId}
          columns={[
            { key: "displayName", header: "Lane", render: (row) => row.displayName },
            { key: "laneNumber", header: "Number", render: (row) => row.laneNumber },
            {
              key: "status",
              header: "Status",
              render: (row) => (
                <StatusBadge status={row.isActive ? "healthy" : "offline"}>
                  {row.isActive ? "Active" : "Inactive"}
                </StatusBadge>
              ),
            },
            { key: "updated", header: "Last Updated", render: (row) => new Date(row.updatedAt).toLocaleString() },
          ]}
        />
      )}
    </Layout>
  );
}
