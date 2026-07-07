import { cn } from "@astra/design-system/utils";

type Status = "healthy" | "degraded" | "offline" | "circuit_open" | "closed" | "half_open" | "open" | "warning";

const STYLES: Record<Status, string> = {
  healthy: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200",
  degraded: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200",
  offline: "bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-200",
  circuit_open: "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-200",
  closed: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200",
  half_open: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200",
  open: "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-200",
  warning: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200",
};

interface StatusBadgeProps {
  readonly status: Status;
  readonly children?: string;
  readonly className?: string;
}

export function StatusBadge({ status, children, className }: StatusBadgeProps): React.JSX.Element {
  const label = children ?? status.replace("_", " ");
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold capitalize",
        STYLES[status],
        className,
      )}
    >
      {label}
    </span>
  );
}
