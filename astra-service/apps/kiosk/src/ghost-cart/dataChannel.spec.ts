import { describe, expect, it } from "vitest";
import { createGhostCartSession } from "./dataChannel";

describe("ghost-cart dataChannel", () => {
  it("creates a session with send/close handlers", () => {
    if (typeof RTCPeerConnection === "undefined") {
      return;
    }
    const session = createGhostCartSession();
    expect(session.send).toBeTypeOf("function");
    expect(session.close).toBeTypeOf("function");
    expect(session.onMessage).toBeTypeOf("function");
    session.close();
  });
});
