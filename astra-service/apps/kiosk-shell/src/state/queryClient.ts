import { QueryClient } from "@tanstack/react-query";

/**
 * TanStack Query is our "server state" layer for anything backed by the Go
 * API gateway (menu catalog, loyalty lookups). Configured for offline-first:
 * aggressive caching + no refetch-on-focus (a kiosk never loses focus in the
 * browser-tab sense) + a long gcTime so cached menu data survives brief
 * network blips without an empty-state flash.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000, // menu prices don't change mid-shift
      gcTime: 24 * 60 * 60 * 1000, // 24h — survives overnight offline windows
      retry: (failureCount, error) => {
        if (error instanceof Response && error.status >= 400 && error.status < 500) {
          return false; // don't retry client errors (bad request, not-found)
        }
        return failureCount < 3;
      },
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10_000),
      refetchOnWindowFocus: false,
      networkMode: "offlineFirst",
    },
    mutations: {
      networkMode: "offlineFirst",
      retry: 2,
    },
  },
});
