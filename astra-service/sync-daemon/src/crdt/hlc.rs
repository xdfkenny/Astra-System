#![deny(unsafe_code)]

//! Hybrid Logical Clock (HLC) implementation.
//!
//! HLC combines a physical wall-clock component with a logical counter to provide
//! causality tracking that is resilient to clock skew and avoids the total ordering
//! problems of pure Lamport clocks.  The implementation follows the algorithm
//! described in "Logical Physical Clocks and Consistent Snapshots in Globally
//! Distributed Databases" (Kulkarni et al., 2014).
//!
//! Each timestamp is a triple `(wallclock_ms, counter, node_id)`.  `node_id` is used
//! only as a deterministic tie-breaker and does not affect happens-before semantics.

use std::cmp::Ordering;
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};

use crate::AstraSyncError;

/// A Hybrid Logical Clock timestamp.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub struct Hlc {
    /// Wall-clock component in milliseconds since the Unix epoch.
    pub wallclock_ms: u64,
    /// Logical counter for events that share the same wall-clock component.
    pub counter: u32,
    /// Originating node identifier, used for deterministic tie-breaking only.
    pub node_id: String,
}

impl Hlc {
    /// Creates a fresh HLC for a local event on `node_id`.
    pub fn new(node_id: impl Into<String>) -> Result<Self, AstraSyncError> {
        let node_id = node_id.into();
        if node_id.is_empty() {
            return Err(AstraSyncError::SyncEngine(
                "HLC node_id must not be empty".to_string(),
            ));
        }
        Ok(Self {
            wallclock_ms: physical_now_ms(),
            counter: 0,
            node_id,
        })
    }

    /// Returns a copy of this HLC without mutating it.
    pub fn snapshot(&self) -> Self {
        self.clone()
    }

    /// Advances the clock for a local event and returns the new timestamp.
    ///
    /// Local advancement rules:
    /// * `l' = max(l, pt)` where `pt` is the current physical time.
    /// * If `l' == l`, increment the counter; otherwise reset it to zero.
    pub fn tick(&mut self) -> Self {
        let pt = physical_now_ms();
        if pt > self.wallclock_ms {
            self.wallclock_ms = pt;
            self.counter = 0;
        } else {
            self.wallclock_ms = self.wallclock_ms.saturating_add(1);
            // If we wrapped the wall-clock component (extremely unlikely in practice),
            // keep the counter monotonically increasing to preserve ordering.
            self.counter = self.counter.saturating_add(1);
        }
        self.snapshot()
    }

    /// Produces a timestamp to send to a remote peer.
    ///
    /// A send is treated as a local event, so the clock is ticked.
    pub fn send(&mut self) -> Self {
        self.tick()
    }

    /// Updates this clock after receiving a remote timestamp.
    ///
    /// Receive rules:
    /// * `l' = max(l, pt, l_m)`.
    /// * If `l' == l == l_m` then `c = max(c, c_m) + 1`.
    /// * Else if `l' == l` then `c = c + 1`.
    /// * Else if `l' == l_m` then `c = c_m + 1`.
    /// * Else `c = 0`.
    pub fn receive(&mut self, remote: &Hlc) -> Self {
        let pt = physical_now_ms();
        let new_wallclock = self.wallclock_ms.max(pt).max(remote.wallclock_ms);

        let new_counter =
            if new_wallclock == self.wallclock_ms && new_wallclock == remote.wallclock_ms {
                self.counter.max(remote.counter).saturating_add(1)
            } else if new_wallclock == self.wallclock_ms {
                self.counter.saturating_add(1)
            } else if new_wallclock == remote.wallclock_ms {
                remote.counter.saturating_add(1)
            } else {
                0
            };

        self.wallclock_ms = new_wallclock;
        self.counter = new_counter;
        self.snapshot()
    }

    /// Partial comparison based on causality.
    ///
    /// Returns `Some(Ordering)` when the two timestamps are comparable (i.e. one
    /// happened-before the other or they are equal).  Returns `None` only when the
    /// wall-clock and counter components are equal but the originating `node_id`
    /// differs, which cannot happen for causally-related events but is possible for
    /// truly concurrent events.  In that case callers should use
    /// [`Hlc::compare_total`] for deterministic convergence.
    pub fn compare(&self, other: &Hlc) -> Option<Ordering> {
        match self.wallclock_ms.cmp(&other.wallclock_ms) {
            Ordering::Equal => match self.counter.cmp(&other.counter) {
                Ordering::Equal => {
                    if self.node_id == other.node_id {
                        Some(Ordering::Equal)
                    } else {
                        None
                    }
                }
                ord => Some(ord),
            },
            ord => Some(ord),
        }
    }

    /// Total deterministic comparison suitable for conflict resolution.
    ///
    /// Causality is preserved: if `a` happened-before `b` then `a.compare_total(b)`
    /// is `Less`.  Concurrent events are ordered deterministically by `node_id`.
    pub fn compare_total(&self, other: &Hlc) -> Ordering {
        self.compare(other)
            .unwrap_or_else(|| self.node_id.cmp(&other.node_id))
    }

    /// Returns `true` if `self` is strictly newer than `other` in causal terms.
    pub fn dominates(&self, other: &Hlc) -> bool {
        matches!(self.compare(other), Some(Ordering::Greater))
    }
}

/// Returns the current physical time in milliseconds since the Unix epoch.
fn physical_now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hlc_local_tick_increases() {
        let mut hlc = Hlc::new("k1").expect("valid node id");
        let t1 = hlc.tick();
        let t2 = hlc.tick();
        assert!(t2.compare_total(&t1).is_gt());
    }

    #[test]
    fn hlc_send_advances_clock() {
        let mut hlc = Hlc::new("k1").expect("valid node id");
        let sent = hlc.send();
        let current = hlc.snapshot();
        assert_eq!(sent, current);
        assert!(current.compare_total(&Hlc::new("k1").unwrap()).is_gt());
    }

    #[test]
    fn hlc_receive_merges_remote_clock() {
        let mut local = Hlc::new("local").expect("valid node id");
        let remote = Hlc {
            wallclock_ms: local.wallclock_ms + 10_000,
            counter: 5,
            node_id: "remote".to_string(),
        };
        let merged = local.receive(&remote);
        assert_eq!(merged.wallclock_ms, remote.wallclock_ms);
        assert_eq!(merged.counter, remote.counter + 1);
    }

    #[test]
    fn hlc_causality_is_preserved() {
        let mut a = Hlc::new("a").expect("valid node id");
        let event1 = a.tick();
        let event2 = a.tick();
        assert!(event2.dominates(&event1));
        assert_eq!(event1.compare(&event2), Some(Ordering::Less));
    }

    #[test]
    fn hlc_concurrent_total_order_is_deterministic() {
        let base = physical_now_ms();
        let x = Hlc {
            wallclock_ms: base,
            counter: 0,
            node_id: "x".to_string(),
        };
        let y = Hlc {
            wallclock_ms: base,
            counter: 0,
            node_id: "y".to_string(),
        };
        assert_eq!(x.compare(&y), None);
        assert_eq!(x.compare_total(&y), Ordering::Less);
        assert_eq!(y.compare_total(&x), Ordering::Greater);
    }

    #[test]
    fn hlc_rejects_empty_node_id() {
        assert!(Hlc::new("").is_err());
    }
}
