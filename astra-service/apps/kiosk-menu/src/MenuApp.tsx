import { useMemo, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { MenuItem } from "@astra/shared-types";
import { useMenuCatalog } from "./useMenuCatalog";
import { MenuItemCard } from "./MenuItemCard";
import { EmptyState } from "@astra/ui-kit";

export interface MenuAppProps {
  readonly laneMode: "express" | "full";
  readonly silentAssistArmed: boolean;
  readonly onSelectItem: (item: MenuItem) => void;
}

const GRID_COLUMNS = 2;
const ROW_HEIGHT_PX = 216; // 128px image + content, matches 8px grid multiples

/**
 * Federated Menu micro-frontend. Virtualizes rows (not individual cards) via
 * TanStack Virtual so a 500-item catalog never mounts more than ~10 rows of
 * DOM nodes at once — critical on the ARM-based kiosk SoC's limited GPU
 * compositing budget. "Express mode" (Lane Intelligence, deep-improvement #3)
 * collapses to a single scrollable list of the top-N best-sellers instead
 * of full category browsing, injected by the host based on queue-length ML.
 */
export default function MenuApp({ laneMode, silentAssistArmed, onSelectItem }: MenuAppProps): React.JSX.Element {
  const { data, isLoading } = useMenuCatalog();
  const [activeCategoryId, setActiveCategoryId] = useState<string | null>(null);
  const scrollParentRef = useRef<HTMLDivElement>(null);

  const items = useMemo(() => {
    const all = data?.items ?? [];
    const filtered = activeCategoryId
      ? all.filter((i) => i.categoryId === activeCategoryId)
      : all;
    return laneMode === "express" ? filtered.slice(0, 12) : filtered;
  }, [data?.items, activeCategoryId, laneMode]);

  const rows = useMemo(() => {
    const chunks: (typeof items)[] = [];
    for (let i = 0; i < items.length; i += GRID_COLUMNS) {
      chunks.push(items.slice(i, i + GRID_COLUMNS));
    }
    return chunks;
  }, [items]);

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollParentRef.current,
    estimateSize: () => ROW_HEIGHT_PX,
    overscan: 3,
  });

  if (isLoading) return <MenuLoadingSkeleton />;

  if (items.length === 0) {
    return (
      <EmptyState
        title="No items available"
        description="This category is temporarily empty. Try another category or check back shortly."
      />
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {laneMode === "full" && data && data.categories.length > 0 ? (
        <nav
          className="flex shrink-0 gap-2 overflow-x-auto p-3"
          aria-label="Menu categories"
        >
          <CategoryChip
            label="All"
            active={activeCategoryId === null}
            onClick={() => { setActiveCategoryId(null); }}
          />
          {data.categories.map((cat) => (
            <CategoryChip
              key={cat.categoryId}
              label={cat.name}
              active={activeCategoryId === cat.categoryId}
              onClick={() => { setActiveCategoryId(cat.categoryId); }}
            />
          ))}
        </nav>
      ) : null}

      <div ref={scrollParentRef} className="flex-1 overflow-y-auto px-3 pb-32">
        <div style={{ height: rowVirtualizer.getTotalSize(), position: "relative" }}>
          {rowVirtualizer.getVirtualItems().map((virtualRow) => (
            <div
              key={virtualRow.key}
              className="absolute left-0 top-0 grid w-full grid-cols-2 gap-3"
              style={{ transform: `translateY(${String(virtualRow.start)}px)` }}
            >
              {rows[virtualRow.index]?.map((item, idx) => (
                <MenuItemCard
                  key={item.itemId}
                  item={item}
                  assistHighlight={silentAssistArmed && virtualRow.index === 0 && idx === 0}
                  onSelectItem={onSelectItem}
                />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function CategoryChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`h-11 shrink-0 rounded-pill px-4 text-sm font-medium transition-colors ${
        active ? "bg-primary text-white" : "hairline bg-surface text-ink-muted"
      }`}
    >
      {label}
    </button>
  );
}

function MenuLoadingSkeleton(): React.JSX.Element {
  return (
    <div className="grid flex-1 grid-cols-2 gap-3 overflow-hidden p-4" aria-busy="true">
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="hairline h-48 animate-pulse rounded-md bg-surface-sunken" />
      ))}
    </div>
  );
}
