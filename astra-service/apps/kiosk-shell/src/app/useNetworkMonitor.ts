import { useEffect } from "react";
import { useSessionStore } from "@astra/kiosk-state";

const HEALTH_POLL_INTERVAL_MS = 5000;

/**
 * Polls the local astra-syncd daemon's HTTP health sidecar (loopback only —
 * never exposed off-device) for mesh peer count / leader status / sync lag,
 * and separately tracks browser online/offline for the "internet reachable"
 * signal shown in the top status bar. These are deliberately distinct
 * signals: a kiosk can be offline from the internet yet fully healthy within
 * its local mesh (the entire point of the P2P architecture).
 */
export function useNetworkMonitor(): void {
  useEffect(() => {
    const setNetworkStatus = useSessionStore.getState().setNetworkStatus;

    const updateOnlineState = (): void => { setNetworkStatus({ online: navigator.onLine }); };
    window.addEventListener("online", updateOnlineState);
    window.addEventListener("offline", updateOnlineState);
    updateOnlineState();

    const pollMeshHealth = async (): Promise<void> => {
      try {
        const res = await fetch("http://127.0.0.1:4499/healthz", {
          signal: AbortSignal.timeout(2000),
        });
        if (!res.ok) throw new Error(`syncd health ${String(res.status)}`);
        const body = (await res.json()) as {
          syncLagMs: number;
          meshPeerCount: number;
          isLeader: boolean;
        };
        setNetworkStatus(body);
      } catch {
        // syncd unreachable — treat mesh as degraded but don't crash the UI.
        setNetworkStatus({ meshPeerCount: 0, isLeader: false });
      }
    };

    void pollMeshHealth();
    const interval = window.setInterval(() => {
      void pollMeshHealth();
    }, HEALTH_POLL_INTERVAL_MS);

    return () => {
      window.removeEventListener("online", updateOnlineState);
      window.removeEventListener("offline", updateOnlineState);
      window.clearInterval(interval);
    };
  }, []);
}
