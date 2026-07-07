import { cn } from "@astra/design-system/utils";
import type { ReactNode } from "react";

interface DataTableProps<T> {
  readonly rows: readonly T[];
  readonly columns: readonly {
    readonly key: string;
    readonly header: string;
    readonly render: (row: T) => ReactNode;
    readonly className?: string;
  }[];
  readonly emptyText?: string;
  readonly keyExtractor: (row: T) => string;
}

export function DataTable<T>({
  rows,
  columns,
  emptyText = "No data available.",
  keyExtractor,
}: DataTableProps<T>): React.JSX.Element {
  return (
    <div className="hairline overflow-hidden rounded-lg">
      <table className="admin-table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col.key} className={col.className}>{col.header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="py-8 text-center text-ink-muted">
                {emptyText}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={keyExtractor(row)}>
                {columns.map((col) => (
                  <td key={`${keyExtractor(row)}-${col.key}`} className={cn("whitespace-nowrap", col.className)}>
                    {col.render(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
