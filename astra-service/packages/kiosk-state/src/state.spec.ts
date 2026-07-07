import { describe, expect, it } from "vitest";
import { cartProxy, addLineItem, updateLineQuantity, removeLineItem, resetCart, derivedCart } from "./cartProxy";
import { useSessionStore, SESSION_IDLE_TIMEOUT_MS, SILENT_ASSIST_STALL_MS } from "./sessionStore";

const kioskId = "kiosk-test";

function resetSessionStore(): void {
  useSessionStore.setState({
    stage: "attract",
    laneMode: "full",
    sessionId: null,
    lastInteractionAtMs: Date.now(),
    network: { online: true, syncLagMs: 0, meshPeerCount: 0, isLeader: false },
    silentAssistArmed: false,
  });
}

describe("cartProxy", () => {
  it("starts empty", () => {
    resetCart(kioskId);
    expect(cartProxy.lines.length).toBe(0);
    expect(derivedCart.isEmpty).toBe(true);
    expect(derivedCart.itemCount).toBe(0);
  });

  it("adds a line item and updates derived state async", async () => {
    resetCart(kioskId);
    addLineItem({
      menuItemId: "item-1",
      nameSnapshot: "Burrito",
      unitPriceCentsSnapshot: 899,
      quantity: 1,
      modifiers: [],
    });
    await new Promise((resolve) => { setTimeout(resolve, 0); });
    expect(cartProxy.lines.length).toBe(1);
    expect(derivedCart.itemCount).toBe(1);
    expect(derivedCart.isEmpty).toBe(false);
  });

  it("updates quantity and removes zero-quantity lines", () => {
    resetCart(kioskId);
    addLineItem({
      menuItemId: "item-1",
      nameSnapshot: "Burrito",
      unitPriceCentsSnapshot: 899,
      quantity: 2,
      modifiers: [],
    });
    const lineId = cartProxy.lines[0]?.lineId;
    if (!lineId) throw new Error("missing line");
    updateLineQuantity(lineId, 0);
    expect(cartProxy.lines.length).toBe(0);
  });

  it("removes a line by id", () => {
    resetCart(kioskId);
    addLineItem({
      menuItemId: "item-1",
      nameSnapshot: "Burrito",
      unitPriceCentsSnapshot: 899,
      quantity: 1,
      modifiers: [],
    });
    const lineId = cartProxy.lines[0]?.lineId;
    if (!lineId) throw new Error("missing line");
    removeLineItem(lineId);
    expect(cartProxy.lines.length).toBe(0);
  });
});

describe("sessionStore", () => {
  it("starts in attract stage", () => {
    resetSessionStore();
    expect(useSessionStore.getState().stage).toBe("attract");
  });

  it("starts a session and moves to menu", () => {
    resetSessionStore();
    useSessionStore.getState().startSession("session-1");
    expect(useSessionStore.getState().stage).toBe("menu");
    expect(useSessionStore.getState().sessionId).toBe("session-1");
  });

  it("records interaction and resets silent assist", () => {
    resetSessionStore();
    useSessionStore.getState().armSilentAssist(true);
    useSessionStore.getState().recordInteraction();
    expect(useSessionStore.getState().silentAssistArmed).toBe(false);
  });

  it("exports idle timeout constants used by the shell", () => {
    expect(SESSION_IDLE_TIMEOUT_MS).toBeGreaterThan(0);
    expect(SILENT_ASSIST_STALL_MS).toBeGreaterThan(0);
  });
});
