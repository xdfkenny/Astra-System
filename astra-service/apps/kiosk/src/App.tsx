import { useCallback, useEffect, useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { KioskMachineProvider, useKioskMachine } from "./machines/KioskMachineProvider";
import { ViewportLock } from "./components/ViewportLock";
import { StatusBar } from "./components/StatusBar";
import { OfflineBanner } from "./components/OfflineBanner";
import { IdleTimeoutOverlay } from "./components/IdleTimeoutOverlay";
import { WorkflowRouter } from "./routes/WorkflowRouter";
import { useIdleReclaim, IDLE_TIMEOUT_MS } from "./hooks/useIdleReclaim";
import { useSilentAssist } from "./hooks/useSilentAssist";
import { useNetworkMonitor } from "./hooks/useNetworkMonitor";
import { useApiNetworkMonitor } from "./hooks/useApiNetworkMonitor";
import { queryClient } from "./state/queryClient";

import "./styles/global.css";
import "./styles/touchFixes.css";
import { KioskErrorBoundary } from "./components/KioskErrorBoundary";

export function App(): React.JSX.Element {
  return (
    <QueryClientProvider client={queryClient}>
      <KioskMachineProvider>
        <ViewportLock>
          <KioskShell />
        </ViewportLock>
      </KioskMachineProvider>
    </QueryClientProvider>
  );
}

/** Show "Still shopping?" warning 15s before idle reclaim fires. */
const IDLE_WARNING_BEFORE_MS = 15_000;

function KioskShell(): React.JSX.Element {
  const { state, send } = useKioskMachine();
  const [showIdleWarning, setShowIdleWarning] = useState(false);

  useIdleReclaim();
  useSilentAssist();
  useNetworkMonitor();
  useApiNetworkMonitor();

  useEffect(() => {
    const preventGesture = (e: Event): void => {
      e.preventDefault();
    };
    document.addEventListener("gesturestart", preventGesture);
    return () => {
      document.removeEventListener("gesturestart", preventGesture);
    };
  }, []);

  // Idle timeout warning
  useEffect(() => {
    const isActive = !["ATTRACT", "PAYMENT", "PROCESSING", "RECEIPT"].includes(
      state.value as string,
    );
    if (!isActive) {
      setShowIdleWarning(false);
      return;
    }

    let lastInteractionMs = Date.now();
    const record = () => {
      lastInteractionMs = Date.now();
      setShowIdleWarning(false);
    };
    window.addEventListener("pointerdown", record);
    window.addEventListener("keydown", record);

    const interval = window.setInterval(() => {
      const elapsed = Date.now() - lastInteractionMs;
      const nearingTimeout = elapsed >= IDLE_TIMEOUT_MS - IDLE_WARNING_BEFORE_MS;
      setShowIdleWarning(nearingTimeout);
    }, 1000);

    return () => {
      window.removeEventListener("pointerdown", record);
      window.removeEventListener("keydown", record);
      window.clearInterval(interval);
    };
  }, [state.value]);

  const handleIdleContinue = useCallback(() => {
    setShowIdleWarning(false);
  }, []);

  const handleIdleReset = useCallback(() => {
    setShowIdleWarning(false);
    send({ type: "RETURN_TO_ATTRACT" });
  }, [send]);

  return (
    <>
      <StatusBar />
      <main className="relative flex flex-1 flex-col overflow-hidden bg-linen">
        <KioskErrorBoundary>
          <WorkflowRouter />
        </KioskErrorBoundary>
      </main>
      <OfflineBanner />
      {showIdleWarning && (
        <IdleTimeoutOverlay
          onContinue={handleIdleContinue}
          onReset={handleIdleReset}
        />
      )}
      <div
        id="astra-live-region"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />
    </>
  );
}

