import { describe, expect, it } from "vitest";
import { compareHlc, incrementHlc, mergeHlc } from "../hlc";
import type { Hlc } from "../hlc";

describe("hlc", () => {
  const nodeA = "kiosk-a";
  const nodeB = "kiosk-b";

  const makeHlc = (physicalTimeMs: number, logicalCounter: number, nodeId: string): Hlc => ({
    physicalTimeMs,
    logicalCounter,
    nodeId,
  });

  it("compares physical time first", () => {
    const a = makeHlc(100, 0, nodeA);
    const b = makeHlc(200, 0, nodeA);
    expect(compareHlc(a, b)).toBe(-1);
    expect(compareHlc(b, a)).toBe(1);
  });

  it("compares logical counter when physical time is equal", () => {
    const a = makeHlc(100, 0, nodeA);
    const b = makeHlc(100, 5, nodeA);
    expect(compareHlc(a, b)).toBe(-1);
    expect(compareHlc(b, a)).toBe(1);
  });

  it("breaks ties by node id", () => {
    const a = makeHlc(100, 0, nodeA);
    const b = makeHlc(100, 0, nodeB);
    expect(compareHlc(a, b)).toBe(-1);
    expect(compareHlc(b, a)).toBe(1);
  });

  it("returns zero for identical HLCs", () => {
    const a = makeHlc(100, 0, nodeA);
    const b = makeHlc(100, 0, nodeA);
    expect(compareHlc(a, b)).toBe(0);
  });

  it("increments logical counter when physical time has not advanced", () => {
    const future = Date.now() + 10_000;
    const hlc = makeHlc(future, 0, nodeA);
    const next = incrementHlc(hlc);
    expect(next.physicalTimeMs).toBe(future);
    expect(next.logicalCounter).toBe(1);
    expect(next.nodeId).toBe(nodeA);
  });

  it("resets logical counter when wall clock advances", () => {
    const past = 1;
    const hlc = makeHlc(past, 99, nodeA);
    const next = incrementHlc(hlc);
    expect(next.physicalTimeMs).toBeGreaterThan(past);
    expect(next.logicalCounter).toBe(0);
  });

  it("merges remote HLCs deterministically", () => {
    const local = makeHlc(100, 5, nodeA);
    const remote = makeHlc(200, 0, nodeB);
    const merged = mergeHlc(local, remote);
    expect(merged.physicalTimeMs).toBe(200);
    expect(merged.logicalCounter).toBe(1);
    expect(merged.nodeId).toBe(nodeA);
  });

  it("merges equal physical times by taking the higher logical counter", () => {
    const local = makeHlc(100, 5, nodeA);
    const remote = makeHlc(100, 3, nodeB);
    const merged = mergeHlc(local, remote);
    expect(merged.physicalTimeMs).toBe(100);
    expect(merged.logicalCounter).toBe(6);
    expect(merged.nodeId).toBe(nodeA);
  });
});
