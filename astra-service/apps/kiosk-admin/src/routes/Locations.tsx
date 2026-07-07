import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { LIST_LOCATIONS } from "../graphql/queries";
import type { AdminListResponse, LocationWithLanes } from "../graphql/types";

export function Locations(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ locations: AdminListResponse<LocationWithLanes> }>(LIST_LOCATIONS);

  return (
    <Layout title="Locations">
      {loading && <Spinner aria-label="Loading locations" />}
      {error && <p className="text-error">Failed to load locations.</p>}
      {data && (
        <DataTable
          rows={data.locations.items}
          keyExtractor={(row) => row.locationId}
          columns={[
            { key: "name", header: "Name", render: (row) => row.name },
            { key: "address", header: "Address", render: (row) => row.address ?? "—" },
            { key: "timezone", header: "Timezone", render: (row) => row.timezone },
            { key: "currency", header: "Currency", render: (row) => row.currency },
            { key: "taxRate", header: "Tax Rate", render: (row) => `${(row.taxRate * 100).toFixed(2)}%` },
            { key: "lanes", header: "Lanes", render: (row) => row.lanes.length },
          ]}
        />
      )}
    </Layout>
  );
}
