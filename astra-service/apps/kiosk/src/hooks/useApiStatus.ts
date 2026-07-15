import { useState, useEffect, useRef } from "react";
import { apiClient } from "../state/apiClient";

export type ApiStatus = "online" | "offline" | "degraded" | "unknown";

const MIN_POLL_MS = 5_000;
const MAX_POLL_MS = 60_000;
const BACKOFF_FACTOR = 2;

export function useApiStatus(pollIntervalMs = 30_000): ApiStatus {
  const [status, setStatus] = useState<ApiStatus>("unknown");
  const intervalRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentIntervalRef = useRef<number>(pollIntervalMs);
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    currentIntervalRef.current = pollIntervalMs;
    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearTimeout(intervalRef.current);
      }
    };
  }, [pollIntervalMs]);

  useEffect(() => {
    let isMounted = true;

    const checkStatus = async () => {
      try {
        await apiClient.checkHealth();
        if (!isMounted) return;

        setStatus("online");
        currentIntervalRef.current = MIN_POLL_MS;
      } catch (error) {
        if (!isMounted) return;

        if (error instanceof Error && error.message.includes("network")) {
          setStatus("offline");
        } else {
          setStatus("degraded");
        }

        currentIntervalRef.current = Math.min(
          currentIntervalRef.current * BACKOFF_FACTOR,
          MAX_POLL_MS,
        );
      }

      intervalRef.current = setTimeout(checkStatus, currentIntervalRef.current);
    };

    void checkStatus();

    return () => {
      isMounted = false;
      if (intervalRef.current) {
        clearTimeout(intervalRef.current);
      }
    };
  }, []);

  return status;
}

