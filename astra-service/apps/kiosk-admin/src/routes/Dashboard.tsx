import { useQuery } from "@apollo/client";
import { Spinner } from "@astra/design-system";
import { Layout } from "../components/Layout";
import { KpiCard } from "../components/KpiCard";
import { MeshTopologyGraph } from "../components/MeshTopologyGraph";
import { CircuitBreakerList } from "../components/CircuitBreakerList";
import { useFleetHealth } from "../hooks/useFleetHealth";
import { DASHBOARD_KPIS } from "../graphql/queries";
import type { DashboardKpis } from "../graphql/types";
import { formatCents, formatPercent } from "../lib/format";

export function Dashboard(): React.JSX.Element {
  const { data: fleet, isLoading: fleetLoading, isError: fleetError } = useFleetHealth();
  const { data: kpiData, loading: kpiLoading, error: kpiError } = useQuery<{ dashboardKpis: DashboardKpis }>(
    DASHBOARD_KPIS,
  );

  const kpis = kpiData?.dashboardKpis;
  const healthyCount = fleet?.nodes.filter((n) => n.health === "healthy").length ?? 0;
  const totalNodes = fleet?.nodes.length ?? 0;

  return (
    <Layout title="Dashboard">
      <div className="flex flex-col gap-6">
        {(kpiLoading || fleetLoading) && <Spinner aria-label="Loading dashboard" />}
        {(kpiError ?? fleetError) && (
          <p className="text-error">Unable to load live dashboard data. Showing cached values where available.</p>
        )}

        <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            title="Revenue"
            value={kpis ? formatCents(kpis.totalRevenueCents) : "—"}
            trend={kpis ? `${kpis.revenueTrend >= 0 ? "+" : ""}${formatPercent(kpis.revenueTrend / 100)} vs last hour` : undefined}
            trendDirection={kpis && kpis.revenueTrend >= 0 ? "up" : "neutral"}
          />
          <KpiCard
            title="Orders"
            value={kpis ? String(kpis.orderCount) : "—"}
            trend={kpis ? `${kpis.orderTrend >= 0 ? "+" : ""}${kpis.orderTrend} vs last hour` : undefined}
            trendDirection={kpis && kpis.orderTrend >= 0 ? "up" : "neutral"}
          />
          <KpiCard title="Active Kiosks" value={kpis ? String(kpis.activeKiosks) : "—"} />
          <KpiCard
            title="Fleet Health"
            value={totalNodes > 0 ? `${healthyCount}/${totalNodes}` : "—"}
            trend={totalNodes > 0 ? `${formatPercent(healthyCount / totalNodes)} healthy` : undefined}
            trendDirection={healthyCount === totalNodes ? "up" : "warning"}
          />
        </section>

        <section className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="rounded-lg border border-border bg-surface p-4">
            <h2 className="mb-2 font-heading text-lg font-semibold">Mesh Topology</h2>
            {fleet && fleet.nodes.length > 0 ? (
              <MeshTopologyGraph nodes={fleet.nodes} />
            ) : (
              <p className="text-ink-muted">No mesh nodes reporting.</p>
            )}
          </div>

          <div className="rounded-lg border border-border bg-surface p-4">
            <h2 className="mb-2 font-heading text-lg font-semibold">Payment Lane Circuit Breakers</h2>
            {fleet && fleet.paymentLanes.length > 0 ? (
              <CircuitBreakerList lanes={fleet.paymentLanes} />
            ) : (
              <p className="text-ink-muted">No payment lane telemetry.</p>
            )}
          </div>
        </section>
      </div>
    </Layout>
  );
}
