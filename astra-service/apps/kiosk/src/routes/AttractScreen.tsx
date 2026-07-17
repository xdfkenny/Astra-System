import { useCallback, useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { uuidV7 } from "@astra/shared-types";
import { resetCart } from "@astra/kiosk-state";
import { useViewport } from "../components/ViewportLock";
import { queryClient } from "../state/queryClient";
import { useTranslation } from "../i18n";

const ENV: Record<string, string | undefined> = import.meta.env;
const KIOSK_ID = ENV["VITE_KIOSK_ID"] ?? "kiosk-local";
const KIOSK_LANE = ENV["VITE_KIOSK_LANE"] ?? "3";
const IDLE_TIMEOUT_MS = 120_000;
const REVEAL_DURATION_MS = 500;

export function AttractScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const { t } = useTranslation();
  const { logicalPoint } = useViewport();
  const [idle, setIdle] = useState(false);
  const [reveal, setReveal] = useState(false);
  const [clientReady, setClientReady] = useState(false);
  const idleRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const revealTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapPoint = useRef({ x: 0, y: 0 });
  const tapHandledRef = useRef(false);

  useEffect(() => {
    setClientReady(true);
    void queryClient.prefetchQuery({
      queryKey: ["menu-catalog"],
      staleTime: 300_000,
    });
  }, []);

  useEffect(() => {
    idleRef.current = setTimeout(() => {
      setIdle(true);
    }, IDLE_TIMEOUT_MS);
    return () => {
      if (idleRef.current) clearTimeout(idleRef.current);
      if (revealTimerRef.current) clearTimeout(revealTimerRef.current);
    };
  }, []);

  const beginSession = useCallback((): void => {
    if (tapHandledRef.current) return;
    tapHandledRef.current = true;
    resetCart(KIOSK_ID);
    send({ type: "START_SESSION", sessionId: uuidV7() });
  }, [send]);

  const startReveal = useCallback(
    (clientX: number, clientY: number): void => {
      if (tapHandledRef.current) return;
      const logical = logicalPoint(clientX, clientY);
      tapPoint.current = { x: logical.x, y: logical.y };
      setReveal(true);
      if (revealTimerRef.current) clearTimeout(revealTimerRef.current);
      revealTimerRef.current = setTimeout(() => {
        beginSession();
      }, REVEAL_DURATION_MS);
    },
    [beginSession, logicalPoint],
  );

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>): void => {
      if (e.button !== 0 && e.button !== -1) return;
      e.preventDefault();
      startReveal(e.clientX, e.clientY);
    },
    [startReveal],
  );

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>): void => {
      if (tapHandledRef.current) return;
      startReveal(e.clientX, e.clientY);
    },
    [startReveal],
  );

  const handleTouchStart = useCallback(
    (e: React.TouchEvent<HTMLDivElement>): void => {
      if (tapHandledRef.current) return;
      const touch = e.touches[0];
      if (touch) {
        startReveal(touch.clientX, touch.clientY);
      }
    },
    [startReveal],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>): void => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        tapPoint.current = {
          x: window.innerWidth / 2,
          y: window.innerHeight / 2,
        };
        setReveal(true);
        beginSession();
      }
    },
    [beginSession],
  );

  if (!clientReady) {
    return (
      <div
        className="attract-screen relative flex flex-1 flex-col items-center justify-center overflow-hidden bg-linen"
        aria-hidden="true"
      />
    );
  }

  return (
    <div
      className="attract-screen relative flex flex-1 flex-col items-center justify-center overflow-hidden bg-linen"
      style={{ filter: idle ? "brightness(0.3)" : undefined }}
      onPointerDown={handlePointerDown}
      onClick={handleClick}
      onTouchStart={handleTouchStart}
      role="button"
      tabIndex={0}
      aria-label={t("attract.screenReader")}
      onKeyDown={handleKeyDown}
      suppressHydrationWarning
    >
      {/* Animated organic blobs */}
      <div className="absolute inset-0 pointer-events-none" aria-hidden="true">
        <motion.div
          className="absolute left-1/4 top-1/4 h-[40vh] w-[40vh] rounded-full bg-moss opacity-[0.04]"
          animate={{
            borderRadius: [
              "60% 40% 30% 70% / 60% 30% 70% 40%",
              "30% 60% 70% 40% / 50% 60% 30% 60%",
              "50% 60% 30% 60% / 30% 40% 70% 50%",
              "60% 40% 30% 70% / 60% 30% 70% 40%",
            ],
            transform: [
              "translate(0, 0)",
              "translate(-2%, 2%)",
              "translate(2%, -2%)",
              "translate(0, 0)",
            ],
          }}
          transition={{
            duration: idle ? 20 : 12,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
        <motion.div
          className="absolute bottom-1/4 right-1/4 h-[35vh] w-[35vh] rounded-full bg-amber opacity-[0.03]"
          animate={{
            borderRadius: [
              "40% 60% 60% 40% / 50% 40% 60% 50%",
              "60% 30% 40% 70% / 40% 60% 40% 60%",
              "30% 70% 50% 50% / 60% 40% 50% 40%",
              "40% 60% 60% 40% / 50% 40% 60% 50%",
            ],
            transform: [
              "translate(0, 0) scale(1)",
              "translate(3%, -1%) scale(1.05)",
              "translate(-2%, 2%) scale(0.95)",
              "translate(0, 0) scale(1)",
            ],
          }}
          transition={{
            duration: idle ? 20 : 12,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
      </div>

      {/* Center content */}
      <div className="relative z-10 text-center">
        <h1 className="font-heading text-[56px] font-semibold tracking-tight text-charcoal">
          {t("attract.title")}
        </h1>
        <motion.p
          className="mt-3 font-sans text-[18px] text-stone"
          animate={{ opacity: [1, 0.3, 1] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
          role="status"
          aria-label={t("attract.touchToBegin")}
        >
          {t("attract.touchToBegin")}
        </motion.p>
      </div>

      {/* Bottom scrolling lane info */}
      <div className="absolute bottom-10 left-0 right-0 overflow-hidden pointer-events-none">
        <motion.p
          className="font-mono text-[12px] text-stone whitespace-nowrap"
          animate={{ x: ["100%", "-100%"] }}
          transition={{ duration: 15, repeat: Infinity, ease: "linear" }}
          aria-hidden="true"
        >
          {t("attract.laneInfo", { lane: KIOSK_LANE })}
        </motion.p>
      </div>

      {/* Clip-path circle reveal on tap */}
      {reveal && (
        <motion.div
          className="absolute inset-0 z-50 bg-linen"
          initial={{
            clipPath: `circle(0% at ${tapPoint.current.x}px ${tapPoint.current.y}px)`,
          }}
          animate={{
            clipPath: `circle(150% at ${tapPoint.current.x}px ${tapPoint.current.y}px)`,
          }}
          transition={{ duration: 0.5, ease: motionTokens.easeOutExpo }}
        />
      )}

      {/* Screen-reader live region */}
      <div className="sr-only" aria-live="assertive" role="status">
        {t("attract.screenReader")}
      </div>
    </div>
  );
}

