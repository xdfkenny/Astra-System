/*Status bar with P2P sync status, time, and connectivity.
Persistent top bar with background on scroll.
*/
import { useEffect, useState } from "react";
import { useScroll, useSpring } from "framer-motion";
import { useSessionStore } from "@astra/kiosk-state";
import { useApiStatus } from "../hooks/useApiStatus";
import { BottomSheet } from "./BottomSheet";
import { cn } from "@/utils/cn";

export function StatusBar() {
  const network = useSessionStore((s) => s.network);
  const [now, setNow] = useState(() => new Date());
  const [meshOpen, setMeshOpen] = useState(false);
  const apiStatus = useApiStatus();

  useEffect(() => {
    const id = window.setInterval(() => {
      setNow(new Date());
    }, 15_000);
    return () => {
      window.clearInterval(id);
    };
  }, []);

  const { scrollY } = useScroll();
  const scrollProgress = useSpring(scrollY, { stiffness: 300, damping: 30 });
  const [bgVisible, setBgVisible] = useState(false);

  useEffect(() => {
    const unsub = scrollProgress.on("change", (v) => {
      setBgVisible(v > 100);
    });
    return () => { unsub(); };
  }, [scrollProgress]);

  const p2pColor: "moss" | "amber" | "stone" = network.online
    ? "moss"
    : network.meshPeerCount > 0
      ? "amber"
      : "stone";

  const p2pLabel =
    p2pColor === "moss"
      ? "Synced"
      : p2pColor === "amber"
        ? "Syncing"
        : "Offline";

  return (
    <header
      className={cn(
        "sticky top-0 z-20 flex h-12 shrink-0 items-center justify-between px-3 transition-colors duration-300",
        bgVisible
          ? "bg-linen/80 backdrop-blur-[8px]"
          : "bg-transparent"
      )}
      aria-label="Kiosk status"
    >
      <button
        type="button"
        className="flex items-center gap-1.5 touch-target"
        aria-label={`P2P sync status: ${p2pLabel}. Tap for mesh details.`}
        onClick={() => {
          setMeshOpen(true);
        }}
      >
        <span
          className="inline-block h-2 w-2 rounded-full"
          style={{
            backgroundColor:
              p2pColor === "moss"
                ? "var(--color-moss, #5A7A5C)"
                : p2pColor === "amber"
                  ? "var(--color-amber, #B87E6B)"
                  : "var(--color-stone, #6B6862)",
          }}
          aria-hidden="true"
        />
        <span className="font-sans text-[12px] text-stone sr-only">
          {p2pLabel}
        </span>
      </button>

      <time
        className="font-sans text-[14px] text-stone tabular-nums"
        dateTime={now.toISOString()}
      >
        {now.toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        })}
      </time>

      <div className="flex items-center gap-2">
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={
            apiStatus === "online" ? "text-moss" :
            apiStatus === "degraded" ? "text-amber" :
            "text-stone"
          }
          aria-label={
            apiStatus === "online" ? "API online" :
            apiStatus === "degraded" ? "API degraded" :
            "API offline"
          }
          role="img"
        >
          {apiStatus === "online" ? (
            <>
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4" strokeLinecap="round" />
              <path d="M12 16v.01" strokeLinecap="round" />
            </>
          ) : apiStatus === "degraded" ? (
            <>
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4" strokeLinecap="round" />
              <path d="M8 12h8" strokeLinecap="round" />
            </>
          ) : (
            <>
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4" strokeLinecap="round" />
              <path d="M8 8l8 8" strokeLinecap="round" />
              <path d="M16 8l-8 8" strokeLinecap="round" />
            </>
          )}
        </svg>

        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={network.online ? "text-moss" : "text-stone"}
          aria-label={
            network.online ? "Connected to network" : "No network connection"
          }
          role="img"
        >
          {network.online ? (
            <>
              <path d="M22.61 16.95A5 5 0 0 0 18 10h-1.26a8 8 0 0 0-7.05-6M5 5a8 8 0 0 0 4 15h9a5 5 0 0 0 1.7-.3" />
              <line x1="1" y1="1" x2="23" y2="23" />
            </>
          ) : (
            <>
              <path d="M2 20h20" />
              <path d="M5 17a9 9 0 0 1 14 0" />
              <path d="M8 13a5 5 0 0 1 8 0" />
            </>
          )}
        </svg>
      </div>

      <BottomSheet open={meshOpen} onClose={() => { setMeshOpen(false); }}>
        <div className="px-4 pb-6 pt-2">
          <h2 className="font-heading text-[20px] font-semibold text-charcoal">
            P2P mesh sync
          </h2>
          <p className="mt-1 font-sans text-[14px] text-stone">
            This kiosk shares cart and order state with nearby peers over a
            local mesh so orders survive network drops.
          </p>

          <dl className="mt-4 space-y-3">
            <div className="flex items-center justify-between">
              <dt className="font-sans text-[14px] text-stone">Status</dt>
              <dd className="font-sans text-[14px] font-medium text-charcoal">{p2pLabel}</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="font-sans text-[14px] text-stone">Connected peers</dt>
              <dd className="font-sans text-[14px] font-medium text-charcoal tabular-nums">
                {network.meshPeerCount}
              </dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="font-sans text-[14px] text-stone">Sync lag</dt>
              <dd className="font-sans text-[14px] font-medium text-charcoal tabular-nums">
                {network.syncLagMs} ms
              </dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="font-sans text-[14px] text-stone">Role</dt>
              <dd className="font-sans text-[14px] font-medium text-charcoal">
                {network.isLeader ? "Leader" : "Peer"}
              </dd>
            </div>
          </dl>

          <div className="mt-4 flex items-center gap-2" aria-hidden="true">
            <span
              className="inline-block h-3 w-3 rounded-full"
              style={{ backgroundColor: "var(--color-moss, #5A7A5C)" }}
            />
            <span className="text-[12px] text-stone">This kiosk</span>
            {Array.from({ length: network.meshPeerCount }, (_, i) => (
              <span
                key={i}
                className="inline-block h-3 w-3 rounded-full"
                style={{ backgroundColor: "var(--color-amber, #B87E6B)" }}
              />
            ))}
          </div>
        </div>
      </BottomSheet>
    </header>
  );
}
