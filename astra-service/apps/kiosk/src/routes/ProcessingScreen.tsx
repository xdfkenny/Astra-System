import { motion } from "framer-motion";

/**
 * Processing overlay shown while the order is being finalized after a
 * successful payment authorization. Uses transform/opacity animation so it
 * stays at 60fps on the embedded ARM GPU.
 */
export function ProcessingScreen(): React.JSX.Element {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 bg-surface p-6 text-center">
      <motion.div
        animate={{ rotate: 360 }}
        transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
        className="h-20 w-20 rounded-full border-4 border-border-strong border-t-primary"
        aria-hidden="true"
      />
      <div>
        <h2 className="font-heading text-2xl font-bold text-ink">Finalizing your order…</h2>
        <p className="mt-2 text-ink-muted" role="status" aria-live="polite">
          Please wait while we print your receipt.
        </p>
      </div>
    </div>
  );
}
