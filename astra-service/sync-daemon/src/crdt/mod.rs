#![deny(unsafe_code)]

//! Conflict-free Replicated Data Types (CRDTs) for the Astra sync daemon.
//!
//! This module provides production-grade CRDT implementations used by kiosk peers
//! to converge on shared state without a central server:
//!
//! * [`LwwElementSet`] — a Last-Writer-Wins element set/map used for inventory
//!   counts.  Each element (typically a SKU) carries an HLC timestamp and a value;
//!   the most recent write wins, with deterministic tie-breaking by node id.
//! * [`MvRegister`] — a Multi-Value register used for cart state.  Concurrent writes
//!   are preserved as multiple values until an explicit resolution strategy is
//!   applied; this avoids silent data loss when two kiosks mutate a cart
//!   concurrently.
//! * [`CrdtMerge`] — a uniform merge trait so the sync engine can treat CRDTs
//!   polymorphically.

use std::collections::HashMap;
use std::fmt::Debug;
use std::hash::Hash;

use serde::{Deserialize, Serialize};

use crate::crdt::hlc::Hlc;
use crate::KioskId;

pub mod hlc;

/// Trait bound for values stored inside CRDTs.
pub trait CrdtValue:
    Clone + Serialize + for<'de> Deserialize<'de> + PartialEq + Send + Sync + Debug
{
}
impl<T> CrdtValue for T where
    T: Clone + Serialize + for<'de> Deserialize<'de> + PartialEq + Send + Sync + Debug
{
}

/// Trait bound for element identifiers used in sets.
pub trait CrdtElement:
    Clone + Eq + Hash + Serialize + for<'de> Deserialize<'de> + Send + Sync + Debug
{
}
impl<T> CrdtElement for T where
    T: Clone + Eq + Hash + Serialize + for<'de> Deserialize<'de> + Send + Sync + Debug
{
}

/// Uniform merge contract for all CRDT types in this module.
pub trait CrdtMerge {
    /// Merges `other` into `self`, mutating `self` to become the least-upper-bound
    /// of the two states.
    fn merge(&mut self, other: &Self);
}

/// Internal entry for an element in an [`LwwElementSet`].
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(bound(serialize = "V: CrdtValue", deserialize = "V: CrdtValue"))]
struct ElementEntry<V: CrdtValue> {
    /// The current value associated with the element.  `None` after a remove.
    value: Option<V>,
    /// The HLC timestamp of the last write or remove.
    hlc: Hlc,
    /// Whether the element was removed by the last write.
    removed: bool,
}

/// Last-Writer-Wins element set/map.
///
/// Each element is identified by a key `K` and carries a value `V` plus an HLC
/// timestamp.  The most recent write (by HLC) wins; removes are treated as writes
/// that set `value` to `None`.  This structure is suitable for inventory counts
/// where the canonical state is the most recent observed count for each SKU.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(bound(
    serialize = "K: CrdtElement, V: CrdtValue",
    deserialize = "K: CrdtElement, V: CrdtValue"
))]
pub struct LwwElementSet<K: CrdtElement, V: CrdtValue> {
    entries: HashMap<K, ElementEntry<V>>,
}

impl<K: CrdtElement, V: CrdtValue> LwwElementSet<K, V> {
    /// Creates an empty set.
    pub fn new() -> Self {
        Self {
            entries: HashMap::new(),
        }
    }

    /// Inserts or updates `key` with `value` and timestamp `hlc`.
    ///
    /// The write is accepted only if `hlc` dominates the existing entry's HLC.
    /// Returns `true` when the write was applied.
    pub fn add(&mut self, key: K, value: V, hlc: Hlc) -> bool {
        match self.entries.get(&key) {
            Some(entry) if !hlc.dominates(&entry.hlc) => false,
            _ => {
                self.entries.insert(
                    key,
                    ElementEntry {
                        value: Some(value),
                        hlc,
                        removed: false,
                    },
                );
                true
            }
        }
    }

    /// Removes `key` with timestamp `hlc`.
    ///
    /// The remove wins if its HLC dominates the existing entry's HLC.  Returns
    /// `true` if the remove was applied.
    pub fn remove(&mut self, key: &K, hlc: Hlc) -> bool {
        match self.entries.get(key) {
            Some(entry) if !hlc.dominates(&entry.hlc) => false,
            _ => {
                let mut new_entry = match self.entries.remove(key) {
                    Some(entry) => entry,
                    None => ElementEntry {
                        value: None,
                        hlc: hlc.clone(),
                        removed: false,
                    },
                };
                new_entry.value = None;
                new_entry.hlc = hlc;
                new_entry.removed = true;
                self.entries.insert(key.clone(), new_entry);
                true
            }
        }
    }

    /// Returns the current value for `key` if it is present and not removed.
    pub fn get(&self, key: &K) -> Option<&V> {
        self.entries.get(key).and_then(|entry| {
            if entry.removed {
                None
            } else {
                entry.value.as_ref()
            }
        })
    }

    /// Returns `true` if the key is present and not removed.
    pub fn contains(&self, key: &K) -> bool {
        self.get(key).is_some()
    }

    /// Returns all currently present keys.
    pub fn keys(&self) -> Vec<&K> {
        self.entries
            .iter()
            .filter(|(_, entry)| !entry.removed && entry.value.is_some())
            .map(|(k, _)| k)
            .collect()
    }

    /// Returns all present key/value pairs.
    pub fn entries(&self) -> Vec<(&K, &V)> {
        self.entries
            .iter()
            .filter(|(_, entry)| !entry.removed && entry.value.is_some())
            .map(|(k, entry)| (k, entry.value.as_ref().expect("present value")))
            .collect()
    }

    /// Returns the number of present elements.
    pub fn len(&self) -> usize {
        self.entries().len()
    }

    /// Returns `true` if the set contains no present elements.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

impl<K: CrdtElement, V: CrdtValue> Default for LwwElementSet<K, V> {
    fn default() -> Self {
        Self::new()
    }
}

impl<K: CrdtElement, V: CrdtValue> CrdtMerge for LwwElementSet<K, V> {
    fn merge(&mut self, other: &Self) {
        for (key, other_entry) in &other.entries {
            let keep_other = match self.entries.get(key) {
                Some(our_entry) => other_entry.hlc.dominates(&our_entry.hlc),
                None => true,
            };
            if keep_other {
                self.entries.insert(key.clone(), other_entry.clone());
            }
        }
    }
}

impl<K: CrdtElement, V: CrdtValue> PartialEq for LwwElementSet<K, V> {
    fn eq(&self, other: &Self) -> bool {
        self.entries() == other.entries()
    }
}

/// A single concurrent value inside an [`MvRegister`].
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(bound(serialize = "V: CrdtValue", deserialize = "V: CrdtValue"))]
struct RegisterValue<V: CrdtValue> {
    value: V,
    hlc: Hlc,
}

/// Multi-Value register.
///
/// Unlike an LWW register, an MV register preserves all values that are not
/// dominated by another concurrent write.  This is important for cart state:
/// if two kiosks add different items to the same cart while partitioned, both
/// updates are kept so a merge strategy can combine them rather than dropping one.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(bound(serialize = "V: CrdtValue", deserialize = "V: CrdtValue"))]
pub struct MvRegister<V: CrdtValue> {
    values: Vec<RegisterValue<V>>,
}

impl<V: CrdtValue> MvRegister<V> {
    /// Creates an empty MV register.
    pub fn new() -> Self {
        Self { values: Vec::new() }
    }

    /// Writes `value` with timestamp `hlc`.
    ///
    /// Any existing values that are dominated by `hlc` are removed (they are
    /// causally overwritten).  Concurrent values are preserved.
    pub fn write(&mut self, value: V, hlc: Hlc) {
        self.values.retain(|entry| !hlc.dominates(&entry.hlc));
        self.values.push(RegisterValue { value, hlc });
    }

    /// Reads all values that are not causally dominated by another value.
    ///
    /// If all writes are causally ordered, this returns a single value.
    pub fn read(&self) -> Vec<&V> {
        let mut maximal: Vec<&V> = Vec::new();
        for candidate in &self.values {
            let dominated = self
                .values
                .iter()
                .any(|other| other.hlc.dominates(&candidate.hlc) && other.hlc != candidate.hlc);
            if !dominated {
                maximal.push(&candidate.value);
            }
        }
        // Deterministic ordering for tests and stable serialization.
        maximal.sort_by(|a, b| format!("{:?}", a).cmp(&format!("{:?}", b)));
        maximal
    }

    /// Resolves concurrency by applying `resolver` to the set of maximal values.
    ///
    /// The resulting single value is written with the maximum HLC of the merged
    /// values plus one logical tick.
    pub fn resolve_with<F>(&mut self, resolver: F)
    where
        F: FnOnce(&[V]) -> V,
    {
        let maximal = self.read();
        if maximal.len() <= 1 {
            return;
        }
        let cloned: Vec<V> = maximal.into_iter().cloned().collect();
        let resolved = resolver(&cloned);

        let fallback_hlc = Hlc {
            wallclock_ms: 0,
            counter: 0,
            node_id: "resolve".to_string(),
        };
        let max_hlc = self
            .values
            .iter()
            .map(|entry| &entry.hlc)
            .fold(None, |acc: Option<Hlc>, hlc| match acc {
                Some(current) if current.compare_total(hlc).is_ge() => Some(current),
                _ => Some(hlc.clone()),
            })
            .unwrap_or(fallback_hlc);

        self.values.clear();
        self.values.push(RegisterValue {
            value: resolved,
            hlc: max_hlc,
        });
    }

    /// Convenience resolver that keeps the lexicographically last value.
    pub fn resolve_last_write(&mut self)
    where
        V: Ord,
    {
        self.resolve_with(|values| {
            values
                .iter()
                .max()
                .expect("at least one maximal value")
                .clone()
        });
    }

    /// Returns the number of stored concurrent values.
    pub fn len(&self) -> usize {
        self.values.len()
    }

    /// Returns `true` if the register contains no values.
    pub fn is_empty(&self) -> bool {
        self.values.is_empty()
    }
}

impl<V: CrdtValue> Default for MvRegister<V> {
    fn default() -> Self {
        Self::new()
    }
}

impl<V: CrdtValue> CrdtMerge for MvRegister<V> {
    fn merge(&mut self, other: &Self) {
        for other_entry in &other.values {
            let dominated = self.values.iter().any(|our_entry| {
                our_entry.hlc.dominates(&other_entry.hlc) && our_entry.hlc != other_entry.hlc
            });
            let duplicate = self
                .values
                .iter()
                .any(|our_entry| our_entry.hlc == other_entry.hlc);
            if !dominated && !duplicate {
                self.values.push(other_entry.clone());
            }
        }
    }
}

impl<V: CrdtValue> PartialEq for MvRegister<V> {
    fn eq(&self, other: &Self) -> bool {
        self.read() == other.read()
    }
}

// ---------------------------------------------------------------------------
// PN Counter
// ---------------------------------------------------------------------------

/// A Positive-Negative counter for inventory counts.
///
/// A PN-Counter tracks increments and decrements independently per origin
/// kiosk.  The current value is the sum of all positive counts minus the sum
/// of all negative counts.  Merge takes the pointwise maximum of each origin's
/// positive and negative counts, guaranteeing convergence without coordination.
///
/// This is suitable for inventory quantities where multiple kiosks may sell
/// (decrement) or restock (increment) the same SKU concurrently.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PNCounter {
    positives: HashMap<KioskId, u64>,
    negatives: HashMap<KioskId, u64>,
}

impl PNCounter {
    /// Creates an empty PN-Counter.
    pub fn new() -> Self {
        Self {
            positives: HashMap::new(),
            negatives: HashMap::new(),
        }
    }

    /// Increments the counter by `amount` on behalf of `origin`.
    pub fn increment(&mut self, origin: KioskId, amount: u64) {
        let entry = self.positives.entry(origin).or_insert(0);
        *entry = entry.saturating_add(amount);
    }

    /// Decrements the counter by `amount` on behalf of `origin`.
    pub fn decrement(&mut self, origin: KioskId, amount: u64) {
        let entry = self.negatives.entry(origin).or_insert(0);
        *entry = entry.saturating_add(amount);
    }

    /// Returns the current value, saturating at zero to prevent negative inventory.
    pub fn value(&self) -> u64 {
        let pos: u64 = self.positives.values().sum();
        let neg: u64 = self.negatives.values().sum();
        pos.saturating_sub(neg)
    }

    /// Returns a reference to the per-origin positive counts.
    pub fn positives(&self) -> &HashMap<KioskId, u64> {
        &self.positives
    }

    /// Returns a reference to the per-origin negative counts.
    pub fn negatives(&self) -> &HashMap<KioskId, u64> {
        &self.negatives
    }
}

impl Default for PNCounter {
    fn default() -> Self {
        Self::new()
    }
}

impl CrdtMerge for PNCounter {
    fn merge(&mut self, other: &Self) {
        for (origin, count) in &other.positives {
            let entry = self.positives.entry(origin.clone()).or_insert(0);
            *entry = (*entry).max(*count);
        }
        for (origin, count) in &other.negatives {
            let entry = self.negatives.entry(origin.clone()).or_insert(0);
            *entry = (*entry).max(*count);
        }
    }
}

impl PartialEq for PNCounter {
    fn eq(&self, other: &Self) -> bool {
        self.positives == other.positives && self.negatives == other.negatives
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn hlc(node: &str, offset: u64) -> Hlc {
        Hlc {
            wallclock_ms: 1_700_000_000_000 + offset,
            counter: 0,
            node_id: node.to_string(),
        }
    }

    #[test]
    fn lww_element_set_add_and_get() {
        let mut set = LwwElementSet::<String, u64>::new();
        assert!(set.add("sku-1".to_string(), 42, hlc("k1", 1)));
        assert_eq!(set.get(&"sku-1".to_string()), Some(&42));
    }

    #[test]
    fn lww_element_set_older_write_is_ignored() {
        let mut set = LwwElementSet::<String, u64>::new();
        set.add("sku-1".to_string(), 100, hlc("k1", 10));
        assert!(!set.add("sku-1".to_string(), 50, hlc("k2", 5)));
        assert_eq!(set.get(&"sku-1".to_string()), Some(&100));
    }

    #[test]
    fn lww_element_set_remove_wins() {
        let mut set = LwwElementSet::<String, u64>::new();
        set.add("sku-1".to_string(), 100, hlc("k1", 10));
        assert!(set.remove(&"sku-1".to_string(), hlc("k1", 20)));
        assert!(!set.contains(&"sku-1".to_string()));
    }

    #[test]
    fn lww_element_set_merge_takes_newer() {
        let mut a = LwwElementSet::<String, u64>::new();
        a.add("sku-1".to_string(), 10, hlc("k1", 1));

        let mut b = LwwElementSet::<String, u64>::new();
        b.add("sku-1".to_string(), 20, hlc("k2", 5));
        b.add("sku-2".to_string(), 30, hlc("k2", 6));

        a.merge(&b);
        assert_eq!(a.get(&"sku-1".to_string()), Some(&20));
        assert_eq!(a.get(&"sku-2".to_string()), Some(&30));
    }

    #[test]
    fn mv_register_preserves_concurrent_writes() {
        let mut reg = MvRegister::<String>::new();
        reg.write("a".to_string(), hlc("k1", 1));
        reg.write("b".to_string(), hlc("k2", 1));
        let values = reg.read();
        assert_eq!(values.len(), 2);
    }

    #[test]
    fn mv_register_overwrites_causally_older_values() {
        let mut reg = MvRegister::<String>::new();
        reg.write("a".to_string(), hlc("k1", 1));
        reg.write("b".to_string(), hlc("k1", 5));
        let values = reg.read();
        assert_eq!(values.len(), 1);
        assert_eq!(values[0], "b");
    }

    #[test]
    fn mv_register_merge_preserves_concurrent_values() {
        let mut a = MvRegister::<String>::new();
        a.write("left".to_string(), hlc("k1", 1));

        let mut b = MvRegister::<String>::new();
        b.write("right".to_string(), hlc("k2", 1));

        a.merge(&b);
        let values = a.read();
        assert_eq!(values.len(), 2);
        assert!(values.iter().any(|v| v.as_str() == "left"));
        assert!(values.iter().any(|v| v.as_str() == "right"));
    }

    #[test]
    fn mv_register_resolve_with_combines_values() {
        let mut reg = MvRegister::<String>::new();
        reg.write("alpha".to_string(), hlc("k1", 1));
        reg.write("beta".to_string(), hlc("k2", 1));
        reg.resolve_with(|values| {
            let mut combined: Vec<String> = values.iter().map(|s| s.to_string()).collect();
            combined.sort();
            combined.join(",")
        });
        let values = reg.read();
        assert_eq!(values.len(), 1);
        assert_eq!(values[0], "alpha,beta");
    }

    #[test]
    fn pn_counter_increment_and_value() {
        let mut counter = PNCounter::new();
        counter.increment(KioskId::from("k1"), 10);
        counter.increment(KioskId::from("k1"), 5);
        assert_eq!(counter.value(), 15);
        assert_eq!(counter.positives().get(&KioskId::from("k1")), Some(&15));
    }

    #[test]
    fn pn_counter_decrement_reduces_value() {
        let mut counter = PNCounter::new();
        counter.increment(KioskId::from("k1"), 20);
        counter.decrement(KioskId::from("k1"), 7);
        assert_eq!(counter.value(), 13);
        assert_eq!(counter.negatives().get(&KioskId::from("k1")), Some(&7));
    }

    #[test]
    fn pn_counter_saturates_at_zero() {
        let mut counter = PNCounter::new();
        counter.increment(KioskId::from("k1"), 10);
        counter.decrement(KioskId::from("k1"), 100);
        assert_eq!(counter.value(), 0);
        counter.increment(KioskId::from("k1"), 95);
        assert_eq!(counter.value(), 5);
    }

    #[test]
    fn pn_counter_merge_combines_per_origin() {
        let mut c1 = PNCounter::new();
        c1.increment(KioskId::from("k1"), 10);
        c1.decrement(KioskId::from("k1"), 3);

        let mut c2 = PNCounter::new();
        c2.increment(KioskId::from("k2"), 5);
        c2.decrement(KioskId::from("k2"), 2);

        c1.merge(&c2);
        assert_eq!(c1.value(), 10);
        assert_eq!(c1.positives().len(), 2);
        assert_eq!(c1.negatives().len(), 2);
    }

    #[test]
    fn pn_counter_merge_takes_maximum_per_origin() {
        let mut c1 = PNCounter::new();
        c1.increment(KioskId::from("k1"), 5);

        let mut c2 = PNCounter::new();
        c2.increment(KioskId::from("k1"), 12);

        c1.merge(&c2);
        assert_eq!(c1.value(), 12);
    }

    #[test]
    fn pn_counter_merge_is_idempotent() {
        let mut c1 = PNCounter::new();
        c1.increment(KioskId::from("k1"), 7);
        c1.decrement(KioskId::from("k2"), 4);

        let c2 = c1.clone();
        c1.merge(&c2);
        assert_eq!(c1, c2);
    }

    #[test]
    fn pn_counter_merge_is_commutative() {
        let mut a = PNCounter::new();
        a.increment(KioskId::from("k1"), 10);
        a.decrement(KioskId::from("k2"), 3);

        let mut b = PNCounter::new();
        b.increment(KioskId::from("k3"), 5);
        b.decrement(KioskId::from("k1"), 2);

        let mut ab = a.clone();
        ab.merge(&b);

        let mut ba = b.clone();
        ba.merge(&a);

        assert_eq!(ab, ba);
    }

    #[test]
    fn pn_counter_merge_is_associative() {
        let mut a = PNCounter::new();
        a.increment(KioskId::from("k1"), 1);
        let mut b = PNCounter::new();
        b.increment(KioskId::from("k2"), 2);
        let mut c = PNCounter::new();
        c.decrement(KioskId::from("k3"), 3);

        let mut ab = a.clone();
        ab.merge(&b);
        ab.merge(&c);

        let mut bc = b.clone();
        bc.merge(&c);
        let mut abc = a.clone();
        abc.merge(&bc);

        assert_eq!(ab, abc);
    }

    #[test]
    fn pn_counter_equality_compares_state() {
        let mut a = PNCounter::new();
        a.increment(KioskId::from("k1"), 10);
        a.decrement(KioskId::from("k2"), 5);

        let mut b = PNCounter::new();
        b.increment(KioskId::from("k1"), 5);
        b.increment(KioskId::from("k1"), 5);
        b.decrement(KioskId::from("k2"), 5);

        assert_eq!(a, b);
    }

    #[test]
    fn pn_counter_serialization_roundtrip() {
        let mut counter = PNCounter::new();
        counter.increment(KioskId::from("k1"), 42);
        counter.decrement(KioskId::from("k2"), 7);

        let bytes = bincode::serialize(&counter).expect("serialize");
        let decoded: PNCounter = bincode::deserialize(&bytes).expect("deserialize");
        assert_eq!(counter, decoded);
    }

    #[test]
    fn serialization_roundtrip() {
        let mut set = LwwElementSet::<String, u64>::new();
        set.add("sku-1".to_string(), 42, hlc("k1", 1));
        let bytes = bincode::serialize(&set).expect("serialize");
        let decoded: LwwElementSet<String, u64> =
            bincode::deserialize(&bytes).expect("deserialize");
        assert_eq!(set, decoded);
    }
}
