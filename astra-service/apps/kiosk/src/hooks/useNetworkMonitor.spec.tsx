import { describe, expect, it, vi } from "vitest";
import { render } from "@testing-library/react";
import { useNetworkMonitor } from "./useNetworkMonitor";
import { useSessionStore } from "@astra/kiosk-state";

function TestHarness(): React.JSX.Element {
  useNetworkMonitor();
  return <div />;
}

describe("useNetworkMonitor", () => {
  it("records the browser online state in the session store", () => {
    vi.stubGlobal("navigator", { onLine: true });
    render(<TestHarness />);

    expect(useSessionStore.getState().network.online).toBe(true);

    vi.unstubAllGlobals();
  });
});
