import { useEffect, useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { useSessionStore } from "@astra/kiosk-state";
import { KioskMachineProvider } from "./machines/KioskMachineProvider";
import { ResponsiveProvider } from "./providers/ResponsiveProvider";
import { ViewportLock } from "./components/ViewportLock";
import { OrientationLock } from "./components/OrientationLock";
import { StatusBar } from "./components/StatusBar";
import { OfflineBanner } from "./components/OfflineBanner";
import { WorkflowRouter } from "./routes/WorkflowRouter";
import { useIdleReclaim } from "./hooks/useIdleReclaim";
import { useSilentAssist } from "./hooks/useSilentAssist";
import { useNetworkMonitor } from "./hooks/useNetworkMonitor";
import { useApiNetworkMonitor } from "./hooks/useApiNetworkMonitor";
import { queryClient } from "./state/queryClient";

import "./styles/global.css";
import "./styles/touchFixes.css";
import { KioskErrorBoundary } from "./components/KioskErrorBoundary";

/**
 * App root. The workflow is driven by the XState kiosk machine, which is the
 * single source of truth for customer stage. HashRouter is kept for the rare
 * drive-thru preview mode that launches from a file-adjacent origin.
 *
 * ResponsiveProvider sits outermost so every descendant (including
 * OrientationLock and ViewportLock) reads from a single ResizeObserver.
 */
export function App(): React.JSX.Element {
  return (
    <QueryClientProvider client={queryClient}>
      <KioskMachineProvider>
        <ResponsiveProvider>
          <OrientationLock>
            <ViewportLock>
              <KioskShell />
            </ViewportLock>
          </OrientationLock>
        </ResponsiveProvider>
      </KioskMachineProvider>
    </QueryClientProvider>
  );
}

const OFFLINE_AMBIENT_BORDER_DELAY_MS = 5 * 60 * 1000;

function KioskShell(): React.JSX.Element {
  useIdleReclaim();
  useSilentAssist();
  useNetworkMonitor();
  useApiNetworkMonitor();

  const online = useSessionStore((s) => s.network.online);
  const [offlineTooLong, setOfflineTooLong] = useState(false);

  useEffect(() => {
    if (online) {
      setOfflineTooLong(false);
      return;
    }
    const timer = window.setTimeout(() => {
      setOfflineTooLong(true);
    }, OFFLINE_AMBIENT_BORDER_DELAY_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [online]);

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
      <main
        className={`bg-linen relative flex flex-1 flex-col overflow-hidden transition-[border-color] duration-500 ${
          offlineTooLong ? "border-2 border-offline/30" : "border-2 border-transparent"
        }`}
      >
        <KioskErrorBoundary>
          <WorkflowRouter />
        </KioskErrorBoundary>
      </main>
      <OfflineBanner />
      <div id="astra-live-region" role="status" aria-live="polite" className="sr-only-live" />
    </>
  );
}
