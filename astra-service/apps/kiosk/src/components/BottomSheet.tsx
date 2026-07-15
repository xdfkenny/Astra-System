/*Bottom sheet for expanded content without leaving current screen.
Used for cart review, P2P mesh info, custom modals.
*/
import { type ReactNode } from "react";
import { AnimatePresence, motion, type PanInfo } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cn } from "@/utils/cn";

export interface BottomSheetProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  "aria-label"?: string;
  title?: string;
  className?: string;
}

const DRAG_THRESHOLD = 80;

export function BottomSheet({
  open,
  onClose,
  children,
  "aria-label": ariaLabel = "Bottom sheet",
  title,
  className,
}: BottomSheetProps) {
  const handleDragEnd = (_: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
    if (info.offset.y > DRAG_THRESHOLD) {
      onClose();
    }
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            key="bottom-sheet-backdrop"
            className="fixed inset-0 z-30 bg-charcoal/20 backdrop-blur-[4px]"
            aria-hidden="true"
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          />
          <motion.section
            key="bottom-sheet"
            role="dialog"
            aria-modal="true"
            aria-label={ariaLabel}
            className={cn(
              "fixed bottom-0 left-0 right-0 z-30 rounded-t-[24px] bg-white/95 backdrop-blur-[12px] shadow-[0_8px_32px_rgba(45,42,38,0.12)]",
              "max-h-[80vh] overflow-hidden",
              className
            )}
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{
              duration: 0.3,
              ease: motionTokens.easeOutExpo,
            }}
            drag="y"
            dragConstraints={{ top: 0, bottom: 0 }}
            dragElastic={0.2}
            onDragEnd={handleDragEnd}
          >
            {title && (
              <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-[12px] px-3 py-2 border-b border-taupe/20">
                <div className="mx-auto mb-2 mt-1 h-1 w-10 shrink-0 rounded bg-taupe" />
                <h2 className="font-heading text-[20px] font-medium text-charcoal text-center">
                  {title}
                </h2>
              </div>
            )}
            <div className={cn(
              "px-3 pb-6",
              title ? "mt-2" : "mt-6"
            )}>{children}</div>
          </motion.section>
        </>
      )}
    </AnimatePresence>
  );
}
