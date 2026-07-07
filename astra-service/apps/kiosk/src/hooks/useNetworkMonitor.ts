import { useEffect } from "react";
import { useSessionStore } from "@astra/kiosk-state";

const HEALTH_POLL_INTERVAL_MS = 5000;

/**
 * Polls the local astra-syncd daemon's HTTP health sidecar and tracks browser
 * online/offline state. These are deliberately distinct: a kiosk can be offline
 * from the internet yet fully healthy within its local mesh (the point of the
 * P2P architecture).
 */
export function useNetworkMonitor(): void {
  useEffect(() => {
    const setNetworkStatus = useSessionStore.getState().setNetworkStatus;

    const updateOnlineState = (): void => {
      setNetworkStatus({ online: navigator.onLine });
    };
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
          readonly syncLagMs: number;
          readonly meshPeerCount: number;
          readonly isLeader: boolean;
        };
        setNetworkStatus(body);
      } catch {
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
