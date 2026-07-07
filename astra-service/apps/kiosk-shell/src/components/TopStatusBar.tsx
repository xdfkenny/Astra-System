import { useEffect, useState } from "react";
import { useSessionStore } from "@astra/kiosk-state";

/** Top status bar: network, sync status, time. Fixed 64px, thumb-zone spec keeps CTAs at bottom. */
export function TopStatusBar(): React.JSX.Element {
  const network = useSessionStore((s) => s.network);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const id = window.setInterval(() => { setNow(new Date()); }, 15_000);
    return () => { window.clearInterval(id); };
  }, []);

  const meshHealthy = network.meshPeerCount > 0 || network.online;
  const statusLabel = network.online
    ? "Online"
    : network.meshPeerCount > 0
      ? `Local mesh (${String(network.meshPeerCount)} lanes)`
      : "Offline — solo mode";

  return (
    <header
      className="hairline flex h-16 shrink-0 items-center justify-between bg-surface px-4 text-ink"
      style={{ borderTop: "none", borderLeft: "none", borderRight: "none" }}
      aria-label="Kiosk status"
    >
      <div className="flex items-center gap-2">
        <span
          className="inline-block h-2.5 w-2.5 rounded-full"
          style={{ background: meshHealthy ? "var(--color-success)" : "var(--color-error)" }}
          aria-hidden="true"
        />
        <span className="text-sm font-medium text-ink-muted">{statusLabel}</span>
        {network.isLeader ? (
          <span className="rounded-pill bg-surface-sunken px-2 py-0.5 text-xs font-semibold text-primary">
            LEADER
          </span>
        ) : null}
      </div>
      <time className="text-sm font-medium tabular-nums text-ink-muted" dateTime={now.toISOString()}>
        {now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
      </time>
    </header>
  );
}
