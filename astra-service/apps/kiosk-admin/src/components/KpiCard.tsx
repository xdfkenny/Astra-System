import { Card } from "@astra/design-system";

interface KpiCardProps {
  readonly title: string;
  readonly value: string;
  readonly trend?: string | undefined;
  readonly trendDirection?: "up" | "down" | "neutral" | "warning" | undefined;
}

export function KpiCard({ title, value, trend, trendDirection = "neutral" }: KpiCardProps): React.JSX.Element {
  const trendColor =
    trendDirection === "up" ? "text-success" : trendDirection === "down" ? "text-error" : "text-ink-muted";
  return (
    <Card className="flex flex-col gap-1">
      <h3 className="text-sm font-medium text-ink-muted">{title}</h3>
      <span className="font-heading text-3xl font-bold text-ink">{value}</span>
      {trend ? <span className={`text-sm ${trendColor}`}>{trend}</span> : null}
    </Card>
  );
}
