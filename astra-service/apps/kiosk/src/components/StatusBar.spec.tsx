import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBar } from "./StatusBar";
import { useSessionStore } from "@astra/kiosk-state";

describe("StatusBar", () => {
  it("renders time and kiosk status region", () => {
    useSessionStore.setState({
      network: { online: true, syncLagMs: 0, meshPeerCount: 2, isLeader: false },
    });
    render(<StatusBar />);

    expect(screen.getByLabelText("Kiosk status")).toBeDefined();
    expect(screen.getByLabelText(/P2P sync status/)).toBeDefined();
  });

  it("shows synced status when online", () => {
    useSessionStore.setState({
      network: { online: true, syncLagMs: 0, meshPeerCount: 1, isLeader: false },
    });
    render(<StatusBar />);

    const btn = screen.getByLabelText(/P2P sync status/);
    expect(btn).toBeDefined();
    expect(btn.getAttribute("aria-label")).toContain("Synced");
  });

  it("shows offline status when not connected", () => {
    useSessionStore.setState({
      network: { online: false, syncLagMs: 0, meshPeerCount: 0, isLeader: false },
    });
    render(<StatusBar />);

    const btn = screen.getByLabelText(/P2P sync status/);
    expect(btn.getAttribute("aria-label")).toContain("Offline");
  });
});
