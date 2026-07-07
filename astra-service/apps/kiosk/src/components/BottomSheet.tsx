import { type ReactNode } from "react";
import { AnimatePresence, motion } from "framer-motion";

export interface BottomSheetProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

const SHEET_ENTRY = {
  initial: { y: "100%" },
  animate: { y: 0 },
  exit: { y: "100%" },
};

const SHEET_TRANSITION_ENTER = {
  duration: 0.3,
  ease: [0.16, 1, 0.3, 1],
};

const SHEET_TRANSITION_EXIT = {
  duration: 0.2,
  ease: [0.16, 1, 0.3, 1],
};

const BACKDROP_ENTRY = {
  initial: { opacity: 0 },
  animate: { opacity: 1 },
  exit: { opacity: 0 },
};

const BACKDROP_TRANSITION = {
  duration: 0.2,
};

export function BottomSheet({ open, onClose, children }: BottomSheetProps) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            key="bottom-sheet-backdrop"
            className="fixed inset-0 z-30 bg-charcoal/20"
            aria-hidden="true"
            onClick={onClose}
            {...BACKDROP_ENTRY}
            transition={BACKDROP_TRANSITION}
          />
          <motion.section
            key="bottom-sheet"
            role="dialog"
            aria-modal="true"
            aria-label="Bottom sheet"
            className="fixed bottom-0 left-0 right-0 z-30 rounded-t-[24px] bg-white/95 backdrop-blur-[12px]"
            variants={SHEET_ENTRY}
            initial="initial"
            animate="animate"
            exit="exit"
            transition={SHEET_TRANSITION_ENTER}
          >
            <div
              className="mx-auto mb-3 mt-3 h-1 w-10 shrink-0 rounded bg-taupe"
              aria-hidden="true"
            />
            <div className="px-3 pb-6">{children}</div>
          </motion.section>
        </>
      )}
    </AnimatePresence>
  );
}
