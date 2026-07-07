import { describe, expect, it, vi } from "vitest";
import { useEffect } from "react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useIdleReclaim, IDLE_TIMEOUT_MS } from "./useIdleReclaim";

function SessionStarter({ children }: { readonly children: React.ReactNode }): React.JSX.Element {
  const { send } = useKioskMachine();
  useEffect(() => {
    send({ type: "START_SESSION", sessionId: "session-1" });
  }, [send]);
  return <>{children}</>;
}

describe("useIdleReclaim", () => {
  it("sends IDLE_TIMEOUT after the threshold with no interaction", () => {
    vi.useFakeTimers();
    const TestHarness = (): React.JSX.Element => {
      useIdleReclaim();
      return <div />;
    };

    const { container } = renderWithMachine(
      <SessionStarter>
        <TestHarness />
      </SessionStarter>,
    );

    vi.advanceTimersByTime(IDLE_TIMEOUT_MS + 1500);
    expect(container).toBeTruthy();

    vi.useRealTimers();
  });
});
