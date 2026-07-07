import { QueryClient } from "@tanstack/react-query";

/**
 * TanStack Query client tuned for kiosk operation:
 * - `networkMode: 'offlineFirst'` so cached catalog data renders instantly during outages.
 * - No retries on the kiosk LAN to fail fast when the local gateway is unreachable.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      networkMode: "offlineFirst",
      retry: 1,
      staleTime: 60_000,
      refetchOnWindowFocus: false,
    },
  },
});
