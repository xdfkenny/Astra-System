/**
 * Hybrid Logical Clock (HLC) implementation used by the cart CRDT merge layer.
 * HLC combines physical wall-clock time with a logical counter so kiosks can
 * establish happens-before ordering without synchronized clocks.
 */

export interface Hlc {
  readonly physicalTimeMs: number;
  readonly logicalCounter: number;
  readonly nodeId: string;
}

/**
 * Compares two HLC values. Returns -1 when `a` happened before `b`, 1 when `a`
 * happened after `b`, and 0 when they are concurrent. Concurrent HLCs are
 * broken by lexicographic nodeId order so every comparison is total.
 */
export function compareHlc(a: Hlc, b: Hlc): -1 | 0 | 1 {
  if (a.physicalTimeMs !== b.physicalTimeMs) {
    return a.physicalTimeMs < b.physicalTimeMs ? -1 : 1;
  }
  if (a.logicalCounter !== b.logicalCounter) {
    return a.logicalCounter < b.logicalCounter ? -1 : 1;
  }
  if (a.nodeId !== b.nodeId) {
    return a.nodeId < b.nodeId ? -1 : 1;
  }
  return 0;
}

/**
 * Merges a local HLC with a received remote HLC and increments the logical
 * counter to advance the clock.
 */
export function mergeHlc(local: Hlc, remote: Hlc): Hlc {
  const localWins = compareHlc(local, remote) >= 0;
  const base = localWins ? local : remote;
  return {
    physicalTimeMs: base.physicalTimeMs,
    logicalCounter: base.logicalCounter + 1,
    nodeId: local.nodeId,
  };
}

/**
 * Increments an HLC for a local event. If the wall clock has advanced beyond
 * the stored physical time, the logical counter resets to zero; otherwise it
 * is incremented.
 */
export function incrementHlc(hlc: Hlc, nodeId?: string): Hlc {
  const now = Date.now();
  if (now > hlc.physicalTimeMs) {
    return {
      physicalTimeMs: now,
      logicalCounter: 0,
      nodeId: nodeId ?? hlc.nodeId,
    };
  }
  return {
    physicalTimeMs: hlc.physicalTimeMs,
    logicalCounter: hlc.logicalCounter + 1,
    nodeId: nodeId ?? hlc.nodeId,
  };
}
