import { Suspense, lazy } from "react";
import { FloatingCartSummary } from "../components/FloatingCartSummary";
import { useSessionStore } from "@astra/kiosk-state";

/**
 * Federated module boundary: the actual menu grid/category browser lives in
 * the independently-deployed `astra_menu` remote (apps/kiosk-menu) so the
 * catalog team can ship pricing/layout changes without a shell redeploy.
 * `lazy` + Suspense gives us a skeleton-screen loading state per the
 * "skeleton over spinners" UX mandate.
 */
const MenuRemote = lazy(() => import("astra_menu/MenuApp"));

export function MenuScreen(): React.JSX.Element {
  const laneMode = useSessionStore((s) => s.laneMode);
  const silentAssistArmed = useSessionStore((s) => s.silentAssistArmed);

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden">
      <Suspense fallback={<MenuSkeleton />}>
        <MenuRemote laneMode={laneMode} silentAssistArmed={silentAssistArmed} />
      </Suspense>
      <FloatingCartSummary />
    </div>
  );
}

function MenuSkeleton(): React.JSX.Element {
  return (
    <div className="grid flex-1 grid-cols-2 gap-3 overflow-hidden p-4" aria-busy="true">
      {Array.from({ length: 8 }, (_, i) => (
        <div
          key={i}
          className="hairline h-48 animate-pulse rounded-md bg-surface-sunken"
        />
      ))}
    </div>
  );
}
