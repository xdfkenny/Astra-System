import { useEffect, useState, useCallback } from "react";
import { motion, useScroll, useSpring } from "framer-motion";
import { useSessionStore } from "@astra/kiosk-state";

export function StatusBar() {
  const network = useSessionStore((s) => s.network);
  const [now, setNow] = useState(() => new Date());

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
    return () => unsub();
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
        onClick={() => {
          /* bottom sheet with mesh details — TBD */
        }}
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
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={`${network.online ? "text-moss" : "text-stone"}`}
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
    </header>
  );
}
