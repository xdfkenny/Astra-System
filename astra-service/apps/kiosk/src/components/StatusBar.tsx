import { useEffect, useState } from "react";
import { useScroll, useSpring } from "framer-motion";
import { useSessionStore } from "@astra/kiosk-state";
import { useApiStatus } from "../hooks/useApiStatus";
import { BottomSheet } from "./BottomSheet";

export function StatusBar() {
  const network = useSessionStore((s) => s.network);
  const [now, setNow] = useState(() => new Date());
  const apiStatus = useApiStatus();
  const [meshOpen, setMeshOpen] = useState(false);

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
      className={`sticky top-0 z-20 flex h-12 shrink-0 items-center justify-between px-3 transition-colors duration-300 ${
        bgVisible
          ? "bg-linen/80 backdrop-blur-[8px]"
          : "bg-transparent"
      }`}
      aria-label="Kiosk status"
    >
      <button
        type="button"
        className="flex items-center gap-1.5"
        aria-label={`P2P sync status: ${p2pLabel}. Tap for mesh details.`}
        onClick={() => { setMeshOpen(true); }}
      >
        <span
          className="inline-block h-2 w-2 rounded-full"
          style={{
            backgroundColor:
              p2pColor === "moss"
                ? "var(--color-moss)"
                : p2pColor === "amber"
                  ? "var(--color-amber)"
                  : "var(--color-stone)",
          }}
          aria-hidden="true"
        />
      </button>

      <time
        className="font-ui text-[14px] text-stone tabular-nums"
        dateTime={now.toISOString()}
      >
        {now.toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        })}
      </time>

       <div className="flex items-center gap-2">
         {/* API Status Icon */}
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
         
         {/* Network Icon */}
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
        <div className="flex flex-col gap-4">
          <h2 className="font-heading text-[24px] font-semibold text-charcoal">
            Mesh network
          </h2>

          {/* Node topology */}
          <div className="flex items-center justify-center gap-2 py-2">
            {Array.from({ length: network.meshPeerCount + 1 }, (_, i) => {
              const isSelf = i === 0;
              const nodeColor = network.online
                ? "var(--color-moss)"
                : network.meshPeerCount > 0
                  ? "var(--color-amber)"
                  : "var(--color-stone)";
              return (
                <div key={i} className="flex items-center">
                  <div className="flex flex-col items-center gap-1">
                    <span
                      className="inline-block h-4 w-4 rounded-full"
                      style={{ backgroundColor: nodeColor }}
                      aria-hidden="true"
                    />
                    <span className="font-mono text-[11px] text-stone">
                      {isSelf ? "Lane 3" : `Lane ${String(i)}`}
                    </span>
                  </div>
                  {i < network.meshPeerCount && (
                    <span
                      className="mx-1 mb-4 h-px w-6 bg-taupe"
                      aria-hidden="true"
                    />
                  )}
                </div>
              );
            })}
          </div>

          {/* Details */}
          <dl className="flex flex-col gap-2 border-t border-taupe pt-3 font-mono text-[13px]">
            <div className="flex items-center justify-between">
              <dt className="text-stone">STATUS</dt>
              <dd className="text-charcoal">{p2pLabel.toUpperCase()}</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-stone">PEERS</dt>
              <dd className="text-charcoal tabular-nums">{network.meshPeerCount}</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-stone">SYNC LAG</dt>
              <dd className="text-charcoal tabular-nums">{network.syncLagMs}ms</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-stone">ROLE</dt>
              <dd className="text-charcoal">{network.isLeader ? "LEADER" : "FOLLOWER"}</dd>
            </div>
          </dl>
        </div>
      </BottomSheet>
    </header>
  );
}
