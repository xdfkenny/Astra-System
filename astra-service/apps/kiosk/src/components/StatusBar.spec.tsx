import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBar } from "./StatusBar";
import { useSessionStore } from "@astra/kiosk-state";

describe("StatusBar", () => {
  it("renders online status and time", () => {
    useSessionStore.setState({ network: { online: true, syncLagMs: 0, meshPeerCount: 2, isLeader: false } });
    render(<StatusBar />);

    expect(screen.getByText("Online")).toBeDefined();
    expect(screen.getByLabelText("Kiosk status")).toBeDefined();
  });

  it("shows the leader badge when the kiosk is the mesh leader", () => {
    useSessionStore.setState({ network: { online: false, syncLagMs: 0, meshPeerCount: 1, isLeader: true } });
    render(<StatusBar />);

    expect(screen.getByText("LEADER")).toBeDefined();
  });
});
