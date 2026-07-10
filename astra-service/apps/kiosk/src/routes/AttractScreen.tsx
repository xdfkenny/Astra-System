import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { uuidV7 } from "@astra/shared-types";
import { resetCart } from "@astra/kiosk-state";

const KIOSK_ID = import.meta.env.VITE_KIOSK_ID ?? "kiosk-local";
const IDLE_TIMEOUT_MS = 120_000;

export function AttractScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const [idle, setIdle] = useState(false);
  const [reveal, setReveal] = useState(false);
  const idleRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapPoint = useRef({ x: 0, y: 0 });

  useEffect(() => {
    idleRef.current = setTimeout(() => { setIdle(true); }, IDLE_TIMEOUT_MS);
    return () => {
      if (idleRef.current) clearTimeout(idleRef.current);
    };
  }, []);

  const handleTap = (e: React.MouseEvent<HTMLDivElement>) => {
    tapPoint.current = { x: e.clientX, y: e.clientY };
    setReveal(true);
    setTimeout(() => {
      resetCart(KIOSK_ID);
      send({ type: "START_SESSION", sessionId: uuidV7() });
    }, 500);
  };

  return (
    <div
      className="relative flex flex-1 flex-col items-center justify-center overflow-hidden bg-linen"
      style={{ filter: idle ? "brightness(0.3)" : undefined }}
      onClick={handleTap}
      role="button"
      tabIndex={0}
      aria-label="Touch to begin shopping"
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          tapPoint.current = { x: window.innerWidth / 2, y: window.innerHeight / 2 };
          setReveal(true);
          setTimeout(() => {
            resetCart(KIOSK_ID);
            send({ type: "START_SESSION", sessionId: uuidV7() });
          }, 500);
        }
      }}
    >
      {/* Animated organic blobs */}
      <div className="absolute inset-0" aria-hidden="true">
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
            delay: idle ? 0 : -4,
          }}
        />
      </div>

      {/* Center content */}
      <div className="relative z-10 text-center">
        <h1 className="font-heading text-hero font-semibold tracking-tight text-charcoal">
          Astra
        </h1>
        <motion.p
          className="mt-3 font-sans text-body text-stone"
          animate={{ opacity: [1, 0.3, 1] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
          role="status"
          aria-label="Touch to begin"
        >
          Touch to begin
        </motion.p>
      </div>

      {/* Bottom scrolling text */}
      <div className="absolute bottom-10 left-0 right-0 overflow-hidden">
        <motion.p
          className="font-mono text-[12px] text-stone whitespace-nowrap"
          animate={{ x: ["100%", "-100%"] }}
          transition={{ duration: 15, repeat: Infinity, ease: "linear" }}
          aria-hidden="true"
        >
          Self-checkout • Lane 3
        </motion.p>
      </div>

      {/* Idle dim overlay */}
      {idle && (
        <div className="absolute inset-0 bg-black/30 z-20" />
      )}

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
      <div className="sr-only-live" aria-live="assertive" role="status">
        Attract screen. Touch to begin shopping.
      </div>
    </div>
  );
}
