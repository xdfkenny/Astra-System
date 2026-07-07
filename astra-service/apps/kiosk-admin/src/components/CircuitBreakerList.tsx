import type { PaymentLaneHealth } from "../hooks/useFleetHealth";
import { StatusBadge } from "./StatusBadge";

interface CircuitBreakerListProps {
  readonly lanes: readonly PaymentLaneHealth[];
}

export function CircuitBreakerList({ lanes }: CircuitBreakerListProps): React.JSX.Element {
  return (
    <ul className="flex flex-col gap-2">
      {lanes.map((lane) => (
        <li
          key={lane.laneId}
          className="flex items-center justify-between rounded-md bg-surface-sunken px-3 py-2 text-sm"
        >
          <span className="font-medium">{lane.laneId}</span>
          <StatusBadge status={lane.circuitState}>{lane.circuitState.replace("_", " ")}</StatusBadge>
          {lane.consecutiveFailures > 0 ? (
            <span className="text-ink-muted">{lane.consecutiveFailures} failures</span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
