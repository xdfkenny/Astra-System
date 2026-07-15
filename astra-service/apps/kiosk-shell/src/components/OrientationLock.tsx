import { type PropsWithChildren } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { useResponsive } from "../providers/ResponsiveProvider";

export function OrientationLock({ children }: PropsWithChildren): React.JSX.Element {
  const { isLandscape } = useResponsive();

  return (
    <>
      {children}
      <AnimatePresence>
        {isLandscape && (
          <motion.div
            key="orientation-warning"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
            className="fixed inset-0 z-[9999] flex flex-col items-center justify-center bg-black"
            role="alert"
            aria-live="assertive"
          >
            <div className="max-w-md px-8 text-center">
              <svg
                className="mx-auto mb-8 h-24 w-12 text-white/60"
                viewBox="0 0 48 96"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <rect x="4" y="4" width="40" height="88" rx="8" ry="8" />
                <rect
                  x="10"
                  y="10"
                  width="28"
                  height="60"
                  rx="2"
                  ry="2"
                  className="fill-white/10"
                />
                <circle cx="24" cy="80" r="3" />
              </svg>

              <h1 className="mb-3 text-2xl font-semibold text-white">
                Vertical orientation required
              </h1>

              <p className="mb-2 text-base leading-relaxed text-gray-400">
                This application is designed for portrait&thinsp;/&thinsp;vertical screen
                orientation.
              </p>

              <p className="text-sm leading-relaxed text-gray-500">
                Please rotate the display or adjust the screen settings to use a vertical layout for
                the best experience.
              </p>

              <div className="mt-8 flex justify-center" aria-hidden="true">
                <svg
                  className="h-8 w-8 animate-pulse text-white/30"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M1 4v6h6" />
                  <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10" />
                </svg>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}
