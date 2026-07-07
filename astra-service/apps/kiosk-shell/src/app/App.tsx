import { useEffect } from "react";
import { HashRouter } from "react-router-dom";
import { ViewportLock } from "../components/ViewportLock";
import { TopStatusBar } from "../components/TopStatusBar";
import { WorkflowRouter } from "../routes/WorkflowRouter";
import { useIdleReclaim } from "../app/useIdleReclaim";
import { useSilentAssist } from "../app/useSilentAssist";
import { useNetworkMonitor } from "../app/useNetworkMonitor";

/**
 * App root. HashRouter (not BrowserRouter) is deliberate: the kiosk browser
 * shell has no address bar and the OS launches Chromium in kiosk mode
 * pointed at a static `file://`-adjacent origin in some deployments —
 * hash routing avoids any server-side rewrite rule dependency.
 */
export function App(): React.JSX.Element {
  useIdleReclaim();
  useSilentAssist();
  useNetworkMonitor();

  useEffect(() => {
    // Kiosk hardware quirk: prevent pinch-zoom / double-tap-zoom gestures that
    // slip past viewport meta on some Chromium-embedded industrial builds.
    const preventGesture = (e: Event): void => { e.preventDefault(); };
    document.addEventListener("gesturestart", preventGesture);
    return () => { document.removeEventListener("gesturestart", preventGesture); };
  }, []);

  return (
    <HashRouter>
      <ViewportLock>
        <TopStatusBar />
        <main className="relative flex flex-1 flex-col overflow-hidden">
          <WorkflowRouter />
        </main>
        {/* ARIA live region for screen-reader announcements (stage changes, errors) */}
        <div id="astra-live-region" role="status" aria-live="polite" className="sr-only-live" />
      </ViewportLock>
    </HashRouter>
  );
}
