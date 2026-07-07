import { Suspense, lazy, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSessionStore } from "@astra/kiosk-state";
import type { MenuItem } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { CartSummary } from "../components/CartSummary";
import { ProduceScanner } from "../produce/ProduceScanner";

const MenuRemote = lazy(() => import("astra_menu/MenuApp"));

/**
 * Federated menu boundary: the catalog grid/category browser lives in the
 * independently-deployed `astra_menu` remote. `lazy` + `Suspense` gives a
 * skeleton-screen loading state per the "skeleton over spinners" UX mandate.
 */
export function MenuScreen(): React.JSX.Element {
  const { state, send } = useKioskMachine();
  const silentAssistArmed = useSessionStore((s) => s.silentAssistArmed);
  const [showScanner, setShowScanner] = useState(false);
  const { data: catalog } = useMenuCatalogForLookup();

  const handleSelectItem = (item: MenuItem): void => {
    setShowScanner(false);
    send({ type: "SELECT_ITEM", item });
  };

  const lookupPlu = (plu: string): MenuItem | null => {
    return catalog?.items.find((item) => item.plu === plu.trim()) ?? null;
  };

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden">
      <div className="flex justify-end p-3">
        <button
          type="button"
          onClick={() => { setShowScanner(true); }}
          className="hairline flex h-14 items-center gap-2 rounded-md bg-surface px-4 font-medium text-ink"
          aria-label="Scan produce by camera or PLU"
        >
          <span aria-hidden="true">📷</span> Scan Produce
        </button>
      </div>
      <Suspense fallback={<MenuSkeleton />}>
        <MenuRemote
          laneMode={state.context.laneMode}
          silentAssistArmed={silentAssistArmed}
          onSelectItem={handleSelectItem}
        />
      </Suspense>
      <CartSummary />
      {showScanner ? (
        <ProduceScanner
          onLookupPlu={lookupPlu}
          onSelectItem={handleSelectItem}
          onClose={() => { setShowScanner(false); }}
        />
      ) : null}
    </div>
  );
}

function useMenuCatalogForLookup() {
  return useQuery<{ readonly items: readonly MenuItem[] }>({
    queryKey: ["menu-catalog"],
    queryFn: async ({ signal }) => {
      const apiBase = import.meta.env.VITE_API_GATEWAY_URL ?? "http://localhost:8080";
      const res = await fetch(`${apiBase}/v1/menu`, { signal });
      if (!res.ok) {
        throw new Error(`Menu fetch failed: ${String(res.status)}`);
      }
      return (await res.json()) as { readonly items: readonly MenuItem[] };
    },
    placeholderData: { items: [] },
  });
}

function MenuSkeleton(): React.JSX.Element {
  return (
    <div className="grid flex-1 grid-cols-2 gap-3 overflow-hidden p-4" aria-busy="true">
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="hairline h-48 animate-pulse rounded-md bg-surface-sunken" />
      ))}
    </div>
  );
}
