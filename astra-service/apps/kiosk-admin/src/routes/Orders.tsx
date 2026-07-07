import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_ORDERS } from "../graphql/queries";
import type { AdminListResponse, OrderWithKiosk } from "../graphql/types";
import { formatCents, formatDate } from "../lib/format";

export function Orders(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ orders: AdminListResponse<OrderWithKiosk> }>(LIST_ORDERS);

  return (
    <Layout title="Orders">
      {loading && <Spinner aria-label="Loading orders" />}
      {error && <p className="text-error">Failed to load orders.</p>}
      {data && (
        <DataTable
          rows={data.orders.items}
          keyExtractor={(row) => row.orderId}
          columns={[
            { key: "orderNumber", header: "Order", render: (row) => row.orderNumber },
            { key: "kiosk", header: "Kiosk", render: (row) => row.kioskDisplayName },
            {
              key: "status",
              header: "Status",
              render: (row) => {
                const status =
                  row.status === "paid" ? "healthy" : row.status === "cancelled" ? "offline" : "warning";
                return <StatusBadge status={status}>{row.status}</StatusBadge>;
              },
            },
            { key: "total", header: "Total", render: (row) => formatCents(row.totalCents) },
            { key: "items", header: "Items", render: (row) => row.itemsJson.length },
            { key: "created", header: "Created", render: (row) => formatDate(row.createdAt) },
          ]}
        />
      )}
    </Layout>
  );
}
