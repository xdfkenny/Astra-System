import { describe, expect, it, vi } from "vitest";
import { useEffect } from "react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useSilentAssist, SILENT_ASSIST_STALL_MS } from "./useSilentAssist";
import { useSessionStore } from "@astra/kiosk-state";

function SessionStarter({ children }: { readonly children: React.ReactNode }): React.JSX.Element {
  const { send } = useKioskMachine();
  useEffect(() => {
    send({ type: "START_SESSION", sessionId: "session-1" });
  }, [send]);
  return <>{children}</>;
}

describe("useSilentAssist", () => {
  it("arms silent assist after the stall threshold on an eligible screen", () => {
    vi.useFakeTimers();
    const TestHarness = (): React.JSX.Element => {
      useSilentAssist();
      return <div />;
    };

    renderWithMachine(
      <SessionStarter>
        <TestHarness />
      </SessionStarter>,
    );

    // Reset interaction time to force stall.
    useSessionStore.setState({ lastInteractionAtMs: Date.now() - SILENT_ASSIST_STALL_MS - 1000 });

    vi.advanceTimersByTime(3000);
    expect(useSessionStore.getState().silentAssistArmed).toBe(true);

    vi.useRealTimers();
  });
});

