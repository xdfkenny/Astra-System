import { compareHlc, incrementHlc, mergeHlc } from "./hlc";
import type { Hlc } from "./hlc";

/**
 * TypeScript bridge types for CRDTs. The Rust `astra-syncd` implementation is
 * the canonical source of truth; these types describe the state exchanged
 * across the WASM/JS boundary and the merge rules applied on both sides.
 */

// -----------------------------------------------------------------------------
// LWW-Element-Set
// -----------------------------------------------------------------------------

export interface LwwElement<T> {
  readonly value: T;
  readonly hlc: Hlc;
}

export interface LwwElementSet<T> {
  readonly adds: ReadonlyMap<string, LwwElement<T>>;
  readonly removes: ReadonlyMap<string, Hlc>;
}

/**
 * Returns the live value for an element id, or undefined when the element has
 * been removed at a timestamp greater than or equal to its add timestamp.
 */
export function lookupLwwElement<T>(set: LwwElementSet<T>, id: string): T | undefined {
  const add = set.adds.get(id);
  if (!add) return undefined;
  const remove = set.removes.get(id);
  if (remove && compareHlc(remove, add.hlc) >= 0) return undefined;
  return add.value;
}

/**
 * Adds or updates an element in an LWW-Element-Set. The add wins if it has a
 * higher HLC than any existing remove.
 */
export function addLwwElement<T>(
  set: LwwElementSet<T>,
  id: string,
  value: T,
  localHlc: Hlc,
): LwwElementSet<T> {
  const nextHlc = incrementHlc(localHlc);
  const nextAdds = new Map(set.adds);
  nextAdds.set(id, { value, hlc: nextHlc });
  return { adds: nextAdds, removes: set.removes };
}

/**
 * Marks an element as removed in an LWW-Element-Set. A remove only wins when
 * its HLC is greater than the corresponding add HLC.
 */
export function removeLwwElement<T>(
  set: LwwElementSet<T>,
  id: string,
  localHlc: Hlc,
): LwwElementSet<T> {
  const nextHlc = incrementHlc(localHlc);
  const nextRemoves = new Map(set.removes);
  nextRemoves.set(id, nextHlc);
  return { adds: set.adds, removes: nextRemoves };
}

/**
 * Merges two LWW-Element-Sets deterministically. Add and remove timestamps are
 * compared per element; the operation with the higher HLC wins.
 */
export function mergeLwwElementSet<T>(
  a: LwwElementSet<T>,
  b: LwwElementSet<T>,
): LwwElementSet<T> {
  const adds = new Map<string, LwwElement<T>>(a.adds);
  for (const [id, element] of b.adds) {
    const existing = adds.get(id);
    if (!existing || compareHlc(element.hlc, existing.hlc) > 0) {
      adds.set(id, element);
    }
  }

  const removes = new Map<string, Hlc>(a.removes);
  for (const [id, hlc] of b.removes) {
    const existing = removes.get(id);
    if (!existing || compareHlc(hlc, existing) > 0) {
      removes.set(id, hlc);
    }
  }

  return { adds, removes };
}

// -----------------------------------------------------------------------------
// MV-Register
// -----------------------------------------------------------------------------

export interface MvRegister<T> {
  readonly values: readonly LwwElement<T>[];
}

/**
 * Writes a value into an MV-Register, advancing the local HLC. If the new HLC
 * dominates all existing values, the register collapses to a single value;
 * otherwise concurrent values are retained.
 */
export function writeMvRegister<T>(
  register: MvRegister<T>,
  value: T,
  localHlc: Hlc,
): MvRegister<T> {
  const nextHlc = incrementHlc(localHlc);
  const dominated = register.values.filter(
    (entry) => compareHlc(nextHlc, entry.hlc) > 0,
  );
  return { values: [...dominated, { value, hlc: nextHlc }] };
}

/**
 * Merges two MV-Registers by keeping only values whose HLC is not dominated by
 * another value in either register.
 */
export function mergeMvRegister<T>(a: MvRegister<T>, b: MvRegister<T>): MvRegister<T> {
  const combined = [...a.values, ...b.values];
  const survivors: LwwElement<T>[] = [];
  for (const candidate of combined) {
    const dominated = combined.some(
      (other) =>
        other !== candidate && compareHlc(other.hlc, candidate.hlc) > 0,
    );
    const alreadyIncluded = survivors.some(
      (existing) =>
        compareHlc(existing.hlc, candidate.hlc) === 0 &&
        existing.value === candidate.value,
    );
    if (!dominated && !alreadyIncluded) {
      survivors.push(candidate);
    }
  }
  return { values: survivors };
}

/**
 * Reads the current value(s) from an MV-Register. Returns a single value when
 * the register is converged, or all concurrent values when it is not.
 */
export function readMvRegister<T>(register: MvRegister<T>): readonly T[] {
  return register.values.map((entry) => entry.value);
}

// -----------------------------------------------------------------------------
// Re-export HLC helpers so CRDT consumers can import from one module.
// -----------------------------------------------------------------------------

export { compareHlc, incrementHlc, mergeHlc };
export type { Hlc };
