import { useEffect } from "react";
import { HashRouter } from "react-router-dom";
import { ResponsiveProvider } from "../providers/ResponsiveProvider";
import { ViewportLock } from "../components/ViewportLock";
import { OrientationLock } from "../components/OrientationLock";
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
 *
 * ResponsiveProvider sits below HashRouter so every descendant reads from
 * a single ResizeObserver + orientation-change subscription.
 */
export function App(): React.JSX.Element {
  useIdleReclaim();
  useSilentAssist();
  useNetworkMonitor();

  useEffect(() => {
    const preventGesture = (e: Event): void => {
      e.preventDefault();
    };
    document.addEventListener("gesturestart", preventGesture);
    return () => {
      document.removeEventListener("gesturestart", preventGesture);
    };
  }, []);

  return (
    <HashRouter>
      <ResponsiveProvider>
        <OrientationLock>
          <ViewportLock>
            <TopStatusBar />
            <main className="relative flex flex-1 flex-col overflow-hidden">
              <WorkflowRouter />
            </main>
            <div id="astra-live-region" role="status" aria-live="polite" className="sr-only-live" />
          </ViewportLock>
        </OrientationLock>
      </ResponsiveProvider>
    </HashRouter>
  );
}
