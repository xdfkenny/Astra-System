import { useQuery } from "@tanstack/react-query";
import type { MenuCatalogResponse } from "./types";

/**
 * Fetches the menu catalog from the Go API gateway's `/v1/menu` REST
 * endpoint (SSE-updated in production — see gateway router.go). TanStack
 * Query's `offlineFirst` network mode (configured in the host's queryClient)
 * means this resolves instantly from cache during a network outage and the
 * kiosk keeps selling from the last known-good catalog snapshot.
 */
export function useMenuCatalog() {
  return useQuery<MenuCatalogResponse>({
    queryKey: ["menu-catalog"],
    queryFn: async ({ signal }) => {
      const apiBase = (import.meta.env["VITE_API_GATEWAY_URL"] as string | undefined) ?? "http://localhost:8080";
      const res = await fetch(`${apiBase}/v1/menu`, { signal });
      if (!res.ok) {
        throw new Error(`Menu fetch failed: ${String(res.status)}`);
      }
      return (await res.json()) as MenuCatalogResponse;
    },
    // Placeholder catalog so the grid renders immediately on very first boot
    // (before any successful fetch has ever populated the cache).
    placeholderData: { categories: [], items: [] },
  });
}
