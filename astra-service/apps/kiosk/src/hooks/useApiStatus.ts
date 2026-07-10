import { useState, useEffect } from "react";
import { apiClient } from "../state/apiClient";

export type ApiStatus = "online" | "offline" | "degraded" | "unknown";

export function useApiStatus(pollIntervalMs = 30_000): ApiStatus {
  const [status, setStatus] = useState<ApiStatus>("unknown");

  useEffect(() => {
    let isMounted = true;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const checkStatus = async () => {
      try {
        await apiClient.checkHealth();
        if (isMounted) {
          setStatus("online");
        }
      } catch (error) {
        if (isMounted) {
          // If we get any response (even error), consider it degraded
          // If we get no response at all, consider it offline
          if (error instanceof Error && error.message.includes("network")) {
            setStatus("offline");
          } else {
            setStatus("degraded");
          }
        }
      }

      // Schedule next check
      if (isMounted) {
        timeoutId = setTimeout(checkStatus, pollIntervalMs);
      }
    };

     // Initial check
     void checkStatus();

    return () => {
      isMounted = false;
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [pollIntervalMs]);

  return status;
}