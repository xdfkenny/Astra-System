import { useEffect } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { KioskMachineProvider } from "./machines/KioskMachineProvider";
import { ViewportLock } from "./components/ViewportLock";
import { StatusBar } from "./components/StatusBar";
import { WorkflowRouter } from "./routes/WorkflowRouter";
import { useIdleReclaim } from "./hooks/useIdleReclaim";
import { useSilentAssist } from "./hooks/useSilentAssist";
import { useNetworkMonitor } from "./hooks/useNetworkMonitor";
import { queryClient } from "./state/queryClient";
import "./styles/global.css";

/**
 * App root. The workflow is driven by the XState kiosk machine, which is the
 * single source of truth for customer stage. HashRouter is kept for the rare
 * drive-thru preview mode that launches from a file-adjacent origin.
 */
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

function KioskShell(): React.JSX.Element {
  useIdleReclaim();
  useSilentAssist();
  useNetworkMonitor();

  useEffect(() => {
    // Kiosk hardware quirk: prevent pinch-zoom / double-tap-zoom gestures.
    const preventGesture = (e: Event): void => {
      e.preventDefault();
    };
    document.addEventListener("gesturestart", preventGesture);
    return () => {
      document.removeEventListener("gesturestart", preventGesture);
    };
  }, []);

  return (
    <>
      <StatusBar />
      <main className="relative flex flex-1 flex-col overflow-hidden bg-linen">
        <WorkflowRouter />
      </main>
      <div id="astra-live-region" role="status" aria-live="polite" className="sr-only-live" />
    </>
  );
}
