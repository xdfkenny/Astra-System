import { useCallback } from "react";
import { motion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useTranslation } from "../i18n";

export function AdminScreen(): React.JSX.Element {
  const { t } = useTranslation();
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
          {t("admin.title")}
        </h1>
        <button
          type="button"
          onClick={handleClose}
          className="h-14 rounded-[16px] border border-stone/30 bg-white/10 px-5 font-sans text-[16px] font-medium text-linen"
          aria-label={t("admin.closeLabel")}
        >
          {t("admin.close")}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <section className="mb-8">
          <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-stone">
            {t("admin.kioskStatus")}
          </h2>
          <div className="space-y-2 font-mono text-[14px]">
            <div className="flex justify-between">
              <span className="text-stone">{t("admin.session")}</span>
              <span className="text-linen">{sessionId}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">{t("admin.laneMode")}</span>
              <span className="text-linen">{laneMode}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">{t("admin.apiStatus")}</span>
              <span className="text-linen">{apiStatus}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone">{t("admin.cartHasItems")}</span>
              <span className="text-linen">
                {state.context.cartHasItems ? t("general.yes") : t("general.no")}
              </span>
            </div>
          </div>
        </section>

        <section className="mb-8">
          <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-stone">
            {t("admin.actions")}
          </h2>
          <div className="flex flex-col gap-3">
            <button
              type="button"
              onClick={() => { send({ type: "NETWORK_ONLINE" }); }}
              className="h-12 rounded-[8px] border border-moss/40 bg-moss/10 font-mono text-[14px] text-moss"
            >
              {t("admin.simulateOnline")}
            </button>
            <button
              type="button"
              onClick={() => { send({ type: "NETWORK_OFFLINE" }); }}
              className="h-12 rounded-[8px] border border-soft-rose/40 bg-soft-rose/10 font-mono text-[14px] text-soft-rose"
            >
              {t("admin.simulateOffline")}
            </button>
          </div>
        </section>

        {state.context.errorMessage && (
          <section className="mb-8">
            <h2 className="mb-3 font-mono text-[14px] uppercase tracking-[0.08em] text-soft-rose">
              {t("admin.errors")}
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

