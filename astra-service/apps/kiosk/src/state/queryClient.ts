import { QueryClient } from "@tanstack/react-query";

function shouldRetry(failureCount: number, error: Error): boolean {
  if (failureCount >= 3) return false;
  if (error.name === "ApiError") {
    const apiErr = error as { statusCode?: number; retryAfterMs?: number };
    if (apiErr.statusCode === 429 && apiErr.retryAfterMs) return true;
    if (apiErr.statusCode === 0) return true;
    if (apiErr.statusCode != null && apiErr.statusCode < 500) return false;
  }
  return true;
}

function retryDelay(failureCount: number): number {
  return Math.min(1_000 * 2 ** failureCount, 15_000);
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      networkMode: "offlineFirst",
      staleTime: 60_000,
      gcTime: 30 * 60 * 1000,
      retry: shouldRetry,
      retryDelay,
      refetchOnWindowFocus: false,
      refetchOnReconnect: true,
    },
    mutations: {
      retry: shouldRetry,
      retryDelay,
    },
  },
});
