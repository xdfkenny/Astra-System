import { useEffect } from "react";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useApiStatus } from "./useApiStatus";

export function useApiNetworkMonitor(): void {
  const { send } = useKioskMachine();
  const apiStatus = useApiStatus(10_000); // Check every 10 seconds

  useEffect(() => {
    // Initial status
    if (apiStatus === "online") {
      send({ type: "NETWORK_ONLINE" });
    } else if (apiStatus === "offline" || apiStatus === "degraded") {
      send({ type: "NETWORK_OFFLINE" });
    }
  }, [apiStatus, send]);

  // Additional network monitoring can be added here
  // For example, listening to online/offline events
  useEffect(() => {
    const handleOnline = () => { send({ type: "NETWORK_ONLINE" }); };
    const handleOffline = () => { send({ type: "NETWORK_OFFLINE" }); };

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [send]);
}