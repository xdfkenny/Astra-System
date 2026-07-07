import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import type { Item } from "@astra/shared-types";
import { Layout } from "../components/Layout";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import { LIST_MENU } from "../graphql/queries";
import type { FullMenu } from "../graphql/types";
import { formatCents } from "../lib/format";

type MenuItemRow = Item & { readonly categoryName: string };

export function Menu(): React.JSX.Element {
  const { data, loading, error } = useQuery<{ menu: FullMenu }>(LIST_MENU);

  const items: MenuItemRow[] =
    data?.menu.categories.flatMap((category) =>
      category.items.map((item) => ({ ...item, categoryName: category.name })),
    ) ?? [];

  return (
    <Layout title="Menu">
      <div className="flex flex-col gap-6">
        {loading && <Spinner aria-label="Loading menu" />}
        {error && <p className="text-error">Failed to load menu.</p>}

        {data && (
          <>
            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Categories & Items</h2>
              <DataTable
                rows={items}
                keyExtractor={(row) => row.itemId}
                columns={[
                  { key: "name", header: "Item", render: (row) => row.name },
                  { key: "category", header: "Category", render: (row) => row.categoryName },
                  { key: "price", header: "Price", render: (row) => formatCents(row.priceCents) },
                  {
                    key: "status",
                    header: "Status",
                    render: (row) => (
                      <StatusBadge status={row.isActive ? "healthy" : "offline"}>
                        {row.isActive ? "Active" : "Inactive"}
                      </StatusBadge>
                    ),
                  },
                  { key: "sku", header: "SKU / PLU", render: (row) => row.sku ?? row.plu ?? "—" },
                ]}
              />
            </section>

            <section>
              <h2 className="mb-3 font-heading text-lg font-semibold">Modifier Groups</h2>
              <DataTable
                rows={data.menu.modifierGroups}
                keyExtractor={(row) => row.modifierGroupId}
                columns={[
                  { key: "name", header: "Group", render: (row) => row.name },
                  { key: "select", header: "Selection", render: (row) => `${row.minSelect}–${row.maxSelect}` },
                  {
                    key: "options",
                    header: "Options",
                    render: (row) =>
                      row.options.map((o) => `${o.name}${o.priceDeltaCents ? ` (+${formatCents(o.priceDeltaCents)})` : ""}`).join(", "),
                  },
                  {
                    key: "status",
                    header: "Status",
                    render: (row) => (
                      <StatusBadge status={row.isActive ? "healthy" : "offline"}>
                        {row.isActive ? "Active" : "Inactive"}
                      </StatusBadge>
                    ),
                  },
                ]}
              />
            </section>
          </>
        )}
      </div>
    </Layout>
  );
}
