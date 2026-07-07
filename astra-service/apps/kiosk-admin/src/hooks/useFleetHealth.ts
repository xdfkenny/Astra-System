import { useQuery } from "@tanstack/react-query";

export type NodeHealth = "healthy" | "degraded" | "circuit_open" | "offline";

export interface KioskNode {
  readonly kioskId: string;
  readonly storeId: string;
  readonly health: NodeHealth;
  readonly isLeader: boolean;
  readonly syncLagMs: number;
  readonly paymentSuccessRate: number;
  readonly meshPeers: readonly string[];
}

export interface PaymentLaneHealth {
  readonly laneId: string;
  readonly circuitState: "closed" | "half_open" | "open";
  readonly consecutiveFailures: number;
  readonly lastFailureReason: string | null;
}

export interface FleetHealthSnapshot {
  readonly nodes: readonly KioskNode[];
  readonly paymentLanes: readonly PaymentLaneHealth[];
  readonly generatedAtMs: number;
}

export function useFleetHealth() {
  return useQuery<FleetHealthSnapshot>({
    queryKey: ["fleet-health"],
    queryFn: async ({ signal }) => {
      const apiBase =
        (import.meta.env["VITE_API_GATEWAY_URL"] as string | undefined) ?? "http://localhost:8080";
      const res = await fetch(`${apiBase}/v1/admin/fleet-health`, { signal });
      if (!res.ok) throw new Error(`Fleet health fetch failed: ${String(res.status)}`);
      return (await res.json()) as FleetHealthSnapshot;
    },
    refetchInterval: 3000,
    placeholderData: (prev) => prev,
  });
}
