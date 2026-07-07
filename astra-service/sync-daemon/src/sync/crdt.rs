#![deny(unsafe_code)]

//! Conflict-free Replicated Data Types (CRDTs) for the Astra sync engine.
//!
//! Implements three CRDT variants:
//! - `LWWRegister<T>`: Last-Writer-Wins register with Lamport timestamps.
//! - `PNCounter`: Positive-Negative counter for inventory counts.
//! - `ORSet<T>`: Observed-Removed set for tags and collections.
//!
//! All types are serializable and support deterministic merge operations.

use std::collections::HashMap;
use std::hash::Hash;

use serde::{Deserialize, Serialize};

use crate::{KioskId, AstraSyncError};

/// Trait bound for types that can be stored in an ORSet.
pub trait ORSetValue: Clone + Eq + Hash + Serialize + for<'de> Deserialize<'de> + Send + Sync {}
impl<T> ORSetValue for T where T: Clone + Eq + Hash + Serialize + for<'de> Deserialize<'de> + Send + Sync {}

// ---------------------------------------------------------------------------
// LWW Register
// ---------------------------------------------------------------------------

/// A Last-Writer-Wins register.
///
/// When two writes conflict, the one with the higher Lamport timestamp wins.
/// If Lamport timestamps are equal, the tie is broken by the lexicographic
/// ordering of the originating `KioskId` to ensure deterministic convergence.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LWWRegister<T> {
    /// The stored value.
    pub value: T,
    /// Lamport timestamp at the time of the write.
    pub lamport_ts: u64,
    /// Kiosk that performed the write.
    pub origin: KioskId,
    /// Wall-clock timestamp (millis since epoch) for human-readable ordering.
    pub wallclock_ts: u64,
}

impl<T: Clone + Serialize + for<'de> Deserialize<'de> + Send + Sync> LWWRegister<T> {
    /// Creates a new LWW register with an initial value and timestamp.
    pub fn new(value: T, origin: KioskId, lamport_ts: u64, wallclock_ts: u64) -> Self {
        Self { value, lamport_ts, origin, wallclock_ts }
    }

    /// Reads the current value.
    pub fn read(&self) -> &T {
        &self.value
    }

    /// Writes a new value if it dominates the current state.
    /// Returns true if the write was accepted (i.e., it dominated).
    pub fn write(&mut self, value: T, origin: KioskId, lamport_ts: u64, wallclock_ts: u64) -> bool {
        if Self::is_dominated(self.lamport_ts, &self.origin, lamport_ts, &origin) {
            self.value = value;
            self.lamport_ts = lamport_ts;
            self.origin = origin;
            self.wallclock_ts = wallclock_ts;
            true
        } else {
            false
        }
    }

    /// Merges another register into this one, keeping the dominant write.
    pub fn merge(&mut self, other: &Self) {
        if Self::is_dominated(self.lamport_ts, &self.origin, other.lamport_ts, &other.origin) {
            self.value = other.value.clone();
            self.lamport_ts = other.lamport_ts;
            self.origin = other.origin.clone();
            self.wallclock_ts = other.wallclock_ts;
        }
    }

    /// Returns true if the candidate (new_ts, new_origin) dominates the incumbent.
    fn is_dominated(inc_ts: u64, inc_origin: &KioskId, new_ts: u64, new_origin: &KioskId) -> bool {
        new_ts > inc_ts || (new_ts == inc_ts && new_origin.0 > inc_origin.0.clone())
    }
}

impl<T: PartialEq + Clone + Serialize + for<'de> Deserialize<'de> + Send + Sync> PartialEq for LWWRegister<T> {
    fn eq(&self, other: &Self) -> bool {
        self.lamport_ts == other.lamport_ts
            && self.origin == other.origin
            && self.value == other.value
    }
}

// ---------------------------------------------------------------------------
// PN Counter
// ---------------------------------------------------------------------------

/// A Positive-Negative counter that tracks increments and decrements per origin.
///
/// Converges by taking the maximum of each origin's positive and negative counts.
/// Suitable for inventory counts where multiple kiosks may sell or restock items.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PNCounter {
    /// Per-origin positive (increment) counts.
    positives: HashMap<KioskId, u64>,
    /// Per-origin negative (decrement) counts.
    negatives: HashMap<KioskId, u64>,
}

impl PNCounter {
    pub fn new() -> Self {
        Self { positives: HashMap::new(), negatives: HashMap::new() }
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

    /// Returns the current value (sum of positives minus sum of negatives).
    /// Saturates at zero to prevent negative inventory.
    pub fn value(&self) -> u64 {
        let pos: u64 = self.positives.values().sum();
        let neg: u64 = self.negatives.values().sum();
        pos.saturating_sub(neg)
    }

    /// Merges another PNCounter into this one.
    pub fn merge(&mut self, other: &Self) {
        for (origin, count) in &other.positives {
            let entry = self.positives.entry(origin.clone()).or_insert(0);
            *entry = (*entry).max(*count);
        }
        for (origin, count) in &other.negatives {
            let entry = self.negatives.entry(origin.clone()).or_insert(0);
            *entry = (*entry).max(*count);
        }
    }

    /// Returns the underlying positive map (for serialization/debugging).
    pub fn positives(&self) -> &HashMap<KioskId, u64> { &self.positives }
    pub fn negatives(&self) -> &HashMap<KioskId, u64> { &self.negatives }
}

impl Default for PNCounter {
    fn default() -> Self { Self::new() }
}

impl PartialEq for PNCounter {
    fn eq(&self, other: &Self) -> bool {
        self.value() == other.value()
    }
}

// ---------------------------------------------------------------------------
// OR Set
// ---------------------------------------------------------------------------

/// An Observed-Removed set where each element is tagged with a unique tag.
///
/// Tags are UUIDs generated by the adding kiosk. Removal is accomplished by
/// removing all observed tags for an element. This is a classic Add-Wins OR-Set.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(bound = "T: ORSetValue")]
pub struct ORSet<T: ORSetValue> {
    /// Map from element to the set of tags that have added it.
    entries: HashMap<T, Vec<String>>,
    /// Set of all tags that have ever been removed (tombstones).
    removed_tags: Vec<String>,
}

impl<T: ORSetValue> ORSet<T> {
    pub fn new() -> Self {
        Self { entries: HashMap::new(), removed_tags: Vec::new() }
    }

    /// Adds an element with a fresh tag.
    pub fn add(&mut self, element: T, tag: String) {
        // Skip if this tag was already removed (defensive).
        if self.removed_tags.contains(&tag) {
            return;
        }
        self.entries.entry(element).or_default().push(tag);
    }

    /// Removes an element by moving all of its current tags into the tombstone set.
    pub fn remove(&mut self, element: &T) {
        if let Some(tags) = self.entries.remove(element) {
            self.removed_tags.extend(tags);
        }
    }

    /// Returns true if the element is present in the set.
    pub fn contains(&self, element: &T) -> bool {
        self.entries.get(element).map_or(false, |tags| {
            !tags.is_empty() && tags.iter().any(|t| !self.removed_tags.contains(t))
        })
    }

    /// Returns all elements currently in the set.
    pub fn elements(&self) -> Vec<T> {
        self.entries
            .iter()
            .filter(|(_, tags)| tags.iter().any(|t| !self.removed_tags.contains(t)))
            .map(|(e, _)| e.clone())
            .collect()
    }

    /// Merges another ORSet into this one.
    pub fn merge(&mut self, other: &Self) {
        // Merge tombstones.
        for tag in &other.removed_tags {
            if !self.removed_tags.contains(tag) {
                self.removed_tags.push(tag.clone());
            }
        }
        // Merge entries.
        for (element, tags) in &other.entries {
            let local_tags = self.entries.entry(element.clone()).or_default();
            for tag in tags {
                if !local_tags.contains(tag) && !self.removed_tags.contains(tag) {
                    local_tags.push(tag.clone());
                }
            }
        }
        // Clean up entries whose only tags are now tombstoned.
        self.entries.retain(|_, tags| {
            tags.retain(|t| !self.removed_tags.contains(t));
            !tags.is_empty()
        });
    }

    pub fn len(&self) -> usize {
        self.entries.len()
    }

    pub fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }
}

impl<T: ORSetValue> Default for ORSet<T> {
    fn default() -> Self { Self::new() }
}

impl<T: ORSetValue> PartialEq for ORSet<T> {
    fn eq(&self, other: &Self) -> bool {
        let self_elems: std::collections::HashSet<_> = self.elements().into_iter().collect();
        let other_elems: std::collections::HashSet<_> = other.elements().into_iter().collect();
        self_elems == other_elems
    }
}

// ---------------------------------------------------------------------------
// Lamport Clock
// ---------------------------------------------------------------------------

/// A Lamport clock for causal ordering.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LamportClock {
    counter: u64,
}

impl LamportClock {
    pub fn new() -> Self { Self { counter: 0 } }

    /// Returns the current time and increments the counter.
    pub fn tick(&mut self) -> u64 {
        self.counter += 1;
        self.counter
    }

    /// Updates the clock with a remote timestamp, taking the max + 1.
    pub fn update(&mut self, remote_ts: u64) -> u64 {
        self.counter = self.counter.max(remote_ts) + 1;
        self.counter
    }

    pub fn now(&self) -> u64 { self.counter }
}

impl Default for LamportClock {
    fn default() -> Self { Self::new() }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_lww_register_dominance() {
        let mut reg = LWWRegister::new("old".to_string(), KioskId::from("k1"), 1, 1000);
        let updated = reg.write("new".to_string(), KioskId::from("k2"), 2, 2000);
        assert!(updated);
        assert_eq!(reg.read(), &"new".to_string());
    }

    #[test]
    fn test_lww_register_tie_break() {
        let mut reg = LWWRegister::new("old".to_string(), KioskId::from("a"), 5, 1000);
        let updated = reg.write("new".to_string(), KioskId::from("b"), 5, 2000);
        assert!(updated);
    }

    #[test]
    fn test_pn_counter_merge() {
        let mut c1 = PNCounter::new();
        c1.increment(KioskId::from("k1"), 10);
        c1.decrement(KioskId::from("k1"), 3);

        let mut c2 = PNCounter::new();
        c2.increment(KioskId::from("k2"), 5);
        c2.decrement(KioskId::from("k2"), 2);

        c1.merge(&c2);
        assert_eq!(c1.value(), 10);
    }

    #[test]
    fn test_or_set_basic() {
        let mut set = ORSet::new();
        set.add("item-a".to_string(), "tag-1".to_string());
        set.add("item-b".to_string(), "tag-2".to_string());
        assert!(set.contains(&"item-a".to_string()));
        set.remove(&"item-a".to_string());
        assert!(!set.contains(&"item-a".to_string()));
        assert!(set.contains(&"item-b".to_string()));
    }

    #[test]
    fn test_or_set_merge() {
        let mut s1 = ORSet::new();
        s1.add("x".to_string(), "t1".to_string());
        let mut s2 = ORSet::new();
        s2.add("y".to_string(), "t2".to_string());
        s1.merge(&s2);
        assert!(s1.contains(&"x".to_string()));
        assert!(s1.contains(&"y".to_string()));
    }

    #[test]
    fn test_lamport_clock() {
        let mut c1 = LamportClock::new();
        assert_eq!(c1.tick(), 1);
        assert_eq!(c1.tick(), 2);
        c1.update(5);
        assert_eq!(c1.now(), 6);
    }
}
