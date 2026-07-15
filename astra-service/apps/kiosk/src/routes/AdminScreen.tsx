import { useCallback } from "react";
import { motion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";

export function AdminScreen(): React.JSX.Element {
  const { state, send } = useKioskMachine();

  const handleClose = useCallback(() => {
    send({ type: "CLOSE_ADMIN" });
  }, [send]);

  const sessionId = state.context.sessionId ?? "N/A";
  const laneMode = state.context.laneMode;
  const apiStatus = state.context.apiStatus;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2 }}
      className="flex flex-1 flex-col bg-charcoal text-linen"
    >
      <div className="flex items-center justify-between border-b border-stone/20 px-6 py-4">
        <h1 className="font-mono text-[24px] font-semibold tracking-tight">
          Admin Panel
        </h1>
        <button
          type="button"
          onClick={handleClose}
          className="h-14 rounded-[16px] border border-stone/30 bg-white/10 px-5 font-sans text-[16px] font-medium text-linen"
          aria-label="Close admin panel"
        >
          Close
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <section className="mb-8">
          <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-stone">
            Kiosk Status
          </h2>
          <div className="space-y-2 font-mono text-[14px]">
            <div className="flex justify-between">
              <span className="text-stone">Session</span>
              <span className="text-linen">{sessionId}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">Lane Mode</span>
              <span className="text-linen">{laneMode}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">API Status</span>
              <span className="text-linen">{apiStatus}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">Cart Has Items</span>
              <span className="text-linen">
                {state.context.cartHasItems ? "Yes" : "No"}
              </span>
            </div>
          </div>
        </section>

        <section className="mb-8">
          <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-stone">
            Actions
          </h2>
          <div className="flex flex-col gap-3">
            <button
              type="button"
              onClick={() => { send({ type: "NETWORK_ONLINE" }); }}
              className="h-12 rounded-[8px] border border-moss/40 bg-moss/10 font-mono text-[14px] text-moss"
            >
              Simulate Online
            </button>
            <button
              type="button"
              onClick={() => { send({ type: "NETWORK_OFFLINE" }); }}
              className="h-12 rounded-[8px] border border-soft-rose/40 bg-soft-rose/10 font-mono text-[14px] text-soft-rose"
            >
              Simulate Offline
            </button>
          </div>
        </section>

        {state.context.errorMessage && (
          <section className="mb-8">
            <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-soft-rose">
              Errors
            </h2>
            <p className="font-mono text-[14px] text-soft-rose">
              {state.context.errorMessage}
            </p>
          </section>
        )}
      </div>
    </motion.div>
  );
}

