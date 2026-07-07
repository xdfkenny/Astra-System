import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_PAYMENTS_AND_REFUNDS } from "../graphql/queries";
import type { AdminListResponse, PaymentWithOrder, RefundWithPayment } from "../graphql/types";
import { formatCents, formatDate } from "../lib/format";

export function PaymentsRefunds(): React.JSX.Element {
  const { data, loading, error } = useQuery<{
    payments: AdminListResponse<PaymentWithOrder>;
    refunds: AdminListResponse<RefundWithPayment>;
  }>(LIST_PAYMENTS_AND_REFUNDS);

  return (
    <Layout title="Payments / Refunds">
      <div className="flex flex-col gap-6">
        {loading && <Spinner aria-label="Loading payments" />}
        {error && <p className="text-error">Failed to load payments.</p>}

        {data && (
          <>
            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Payments</h2>
              <DataTable
                rows={data.payments.items}
                keyExtractor={(row) => row.paymentId}
                columns={[
                  { key: "order", header: "Order", render: (row) => row.orderNumber },
                  { key: "method", header: "Method", render: (row) => row.method.replace(/_/g, " ") },
                  { key: "amount", header: "Amount", render: (row) => formatCents(row.amountCents, row.currency) },
                  {
                    key: "status",
                    header: "Status",
                    render: (row) => {
                      const status =
                        row.status === "captured"
                          ? "healthy"
                          : row.status === "declined"
                            ? "offline"
                            : "warning";
                      return <StatusBadge status={status}>{row.status}</StatusBadge>;
                    },
                  },
                  { key: "card", header: "Card", render: (row) => `${row.cardBrand ?? ""} ${row.cardLastFour ?? ""}`.trim() || "—" },
                  { key: "created", header: "Created", render: (row) => formatDate(row.createdAt) },
                ]}
              />
            </section>

            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Refunds</h2>
              <DataTable
                rows={data.refunds.items}
                keyExtractor={(row) => row.refundId}
                columns={[
                  { key: "order", header: "Order", render: (row) => row.orderId.slice(0, 8) },
                  { key: "amount", header: "Amount", render: (row) => formatCents(row.amountCents, row.currency) },
                  {
                    key: "status",
                    header: "Status",
                    render: (row) => {
                      const status =
                        row.status === "completed" ? "healthy" : row.status === "failed" ? "offline" : "warning";
                      return <StatusBadge status={status}>{row.status}</StatusBadge>;
                    },
                  },
                  { key: "reason", header: "Reason", render: (row) => row.reason },
                  { key: "processedBy", header: "Processed By", render: (row) => row.processedBy ?? "—" },
                  { key: "created", header: "Created", render: (row) => formatDate(row.createdAt) },
                ]}
              />
            </section>
          </>
        )}
      </div>
    </Layout>
  );
}
