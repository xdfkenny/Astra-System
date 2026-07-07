import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_INVENTORY } from "../graphql/queries";
import type { AdminListResponse, InventoryWithItem } from "../graphql/types";

export function Inventory(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ inventory: AdminListResponse<InventoryWithItem> }>(LIST_INVENTORY);

  return (
    <Layout title="Inventory">
      {loading && <Spinner aria-label="Loading inventory" />}
      {error && <p className="text-error">Failed to load inventory.</p>}
      {data && (
        <DataTable
          rows={data.inventory.items}
          keyExtractor={(row) => row.inventoryId}
          columns={[
            { key: "item", header: "Item", render: (row) => row.itemName },
            { key: "sku", header: "SKU", render: (row) => row.itemSku ?? "—" },
            { key: "available", header: "Available", render: (row) => row.quantityAvailable },
            { key: "reserved", header: "Reserved", render: (row) => row.quantityReserved },
            { key: "onOrder", header: "On Order", render: (row) => row.quantityOnOrder },
            {
              key: "status",
              header: "Status",
              render: (row) => {
                const status = row.quantityAvailable <= row.reorderPoint ? "degraded" : "healthy";
                return <StatusBadge status={status}>{status === "healthy" ? "OK" : "Low Stock"}</StatusBadge>;
              },
            },
            { key: "location", header: "Location", render: (row) => row.location ?? "—" },
          ]}
        />
      )}
    </Layout>
  );
}
