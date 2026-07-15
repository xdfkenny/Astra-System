#![deny(unsafe_code)]

//! Simplified Raft consensus implementation for leader election among kiosks.
//!
//! This is a **leader-election-only** Raft variant. We do not replicate a full
//! log across all nodes; instead, we use Raft solely to:
//! - Elect a single leader among LAN peers.
//! - Detect network partitions.
//! - Trigger leader failover when the current leader disappears.
//!
//! The actual data sync happens via the CRDT engine, which is partition-tolerant
//! by design. The leader is only responsible for:
//! - Flushing batched data to the cloud NATS JetStream backend.
//! - Acting as the NATS consumer/producer for cloud operations.
//! - Serving as the authoritative time source for Lamport clock initialization.
//!
//! States: Follower -> Candidate -> Leader.

use std::sync::Arc;
use std::time::Duration;

use rand::Rng;
use tokio::sync::{mpsc, watch, RwLock};
use tokio::time::{interval, Instant};
use tracing::{info, trace, warn};

use crate::config::{Config, RaftConfig};
use crate::p2p::mesh::P2PMeshHandle;
use crate::protocol::SyncProtocol;
use crate::storage::sqlite::SyncDatabase;
use crate::{AstraSyncError, DaemonState};

/// Raft node states.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RaftState {
    Follower,
    Candidate,
    Leader,
}

/// Events emitted by the Raft node for external observers.
#[derive(Debug, Clone)]
pub enum RaftEvent {
    BecameLeader { term: u64 },
    SteppedDown { term: u64 },
    VoteGranted { term: u64, voter: String },
    HeartbeatReceived { term: u64, leader: String },
}

pub struct RaftNode {
    config: Arc<Config>,
    state: Arc<RwLock<DaemonState>>,
    #[allow(dead_code)]
    db: Arc<SyncDatabase>,
    mesh: P2PMeshHandle,
    /// Local Raft state machine.
    raft_state: Arc<RwLock<RaftStateMachine>>,
    /// Event broadcast channel.
    event_tx: mpsc::Sender<RaftEvent>,
}

/// The core Raft state machine, protected by a RwLock for concurrent access.
struct RaftStateMachine {
    state: RaftState,
    /// Current term, monotonically increasing.
    current_term: u64,
    /// Candidate voted for in the current term (empty string = none).
    voted_for: Option<String>,
    /// Leader peer ID (empty if unknown).
    leader_id: Option<String>,
    /// Last time a valid message was received from the leader.
    last_leader_heartbeat: Instant,
    /// Number of votes received in the current election.
    votes_received: usize,
    /// Total known peers in the cluster (including self).
    known_peers: usize,
    /// Local peer ID.
    my_id: String,
}

impl RaftNode {
    pub async fn new(
        config: Arc<Config>,
        state: Arc<RwLock<DaemonState>>,
        db: Arc<SyncDatabase>,
        mesh: P2PMeshHandle,
    ) -> Result<Self, AstraSyncError> {
        let my_id = config.daemon.kiosk_id.clone();
        let (event_tx, _event_rx) = mpsc::channel(256);

        let raft_state = Arc::new(RwLock::new(RaftStateMachine {
            state: RaftState::Follower,
            current_term: 0,
            voted_for: None,
            leader_id: None,
            last_leader_heartbeat: Instant::now(),
            votes_received: 0,
            known_peers: 1, // Self only until mesh discovery fills this in.
            my_id: my_id.clone(),
        }));

        info!(kiosk_id = %my_id, "Raft node initialized");

        Ok(Self {
            config,
            state,
            db,
            mesh,
            raft_state,
            event_tx,
        })
    }

    /// Starts the Raft event loop and returns a join handle.
    pub async fn start(
        self,
        mut shutdown: watch::Receiver<bool>,
    ) -> Result<tokio::task::JoinHandle<()>, AstraSyncError> {
        let raft_state = self.raft_state.clone();
        let state = self.state.clone();
        let cfg = self.config.raft.clone();
        let event_tx = self.event_tx.clone();
        let mesh = self.mesh.clone();

        let handle = tokio::spawn(async move {
            info!("Raft event loop started");

            let mut heartbeat_timer = interval(Duration::from_millis(cfg.heartbeat_interval_ms));
            heartbeat_timer.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);

            let mut election_check_timer = interval(Duration::from_millis(100));
            election_check_timer.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

            loop {
                tokio::select! {
                    _ = heartbeat_timer.tick() => {
                        let term;
                        let leader_id;
                        {
                            let sm = raft_state.read().await;
                            if sm.state != RaftState::Leader {
                                continue;
                            }
                            term = sm.current_term;
                            leader_id = sm.my_id.clone();
                        }
                        // Broadcast heartbeat to all peers.
                        if let Err(e) = broadcast_heartbeat(term, leader_id, &mesh).await {
                            warn!(%e, "Heartbeat broadcast failed");
                        }
                        // Update shared daemon state.
                        let mut ds = state.write().await;
                        ds.is_leader = true;
                        ds.raft_term = term;
                    }
                    _ = election_check_timer.tick() => {
                        let mut sm = raft_state.write().await;
                        match sm.state {
                            RaftState::Follower => {
                                let timeout = election_timeout(&cfg);
                                if sm.last_leader_heartbeat.elapsed() > timeout {
                                    let term = sm.current_term + 1;
                                    let my_id = sm.my_id.clone();
                                    info!("Election timeout elapsed — converting to candidate");
                                    sm.state = RaftState::Candidate;
                                    sm.current_term = term;
                                    sm.voted_for = Some(my_id.clone());
                                    sm.votes_received = 1; // Vote for self.
                                    sm.leader_id = None;
                                    drop(sm);
                                    // Request votes from peers.
                                    if let Err(e) = request_votes(term, my_id, &mesh).await {
                                        warn!(%e, "Vote request failed");
                                    }
                                }
                            }
                            RaftState::Candidate => {
                                // Check if we won the election.
                                let votes = sm.votes_received;
                                let total = sm.known_peers.max(1);
                                if votes > total / 2 {
                                    info!(term = sm.current_term, "Became Raft leader");
                                    sm.state = RaftState::Leader;
                                    sm.leader_id = Some(sm.my_id.clone());
                                    let _ = event_tx.send(RaftEvent::BecameLeader { term: sm.current_term }).await;
                                    let mut ds = state.write().await;
                                    ds.is_leader = true;
                                    ds.raft_term = sm.current_term;
                                } else if sm.last_leader_heartbeat.elapsed() > election_timeout(&cfg) * 2 {
                                    // Election timed out — start a new one.
                                    let term = sm.current_term + 1;
                                    let my_id = sm.my_id.clone();
                                    warn!("Election timed out — starting new election");
                                    sm.current_term = term;
                                    sm.voted_for = Some(my_id.clone());
                                    sm.votes_received = 1;
                                    sm.last_leader_heartbeat = Instant::now();
                                    drop(sm);
                                    if let Err(e) = request_votes(term, my_id, &mesh).await {
                                        warn!(%e, "Vote request failed");
                                    }
                                }
                            }
                            RaftState::Leader => {
                                // Ensure we remain leader in shared state.
                                let mut ds = state.write().await;
                                ds.is_leader = true;
                                ds.raft_term = sm.current_term;
                            }
                        }
                    }
                    _ = shutdown.changed() => {
                        info!("Raft event loop shutting down");
                        let mut sm = raft_state.write().await;
                        if sm.state == RaftState::Leader {
                            sm.state = RaftState::Follower;
                            sm.leader_id = None;
                            let _ = event_tx.send(RaftEvent::SteppedDown { term: sm.current_term }).await;
                        }
                        let mut ds = state.write().await;
                        ds.is_leader = false;
                        break;
                    }
                }
            }

            info!("Raft event loop stopped");
        });

        Ok(handle)
    }

    /// Handles an incoming Raft AppendEntries (heartbeat) message.
    pub async fn handle_append_entries(
        &self,
        term: u64,
        leader_id: &str,
    ) -> Result<bool, AstraSyncError> {
        let mut sm = self.raft_state.write().await;
        if term < sm.current_term {
            // Reject stale heartbeat.
            return Ok(false);
        }
        if term > sm.current_term {
            // Update term and step down if we were a leader.
            sm.current_term = term;
            sm.voted_for = None;
            if sm.state == RaftState::Leader {
                info!("Stepping down — higher term observed");
                sm.state = RaftState::Follower;
                let _ = self.event_tx.send(RaftEvent::SteppedDown { term }).await;
            }
        }
        sm.state = RaftState::Follower;
        sm.leader_id = Some(leader_id.to_string());
        sm.last_leader_heartbeat = Instant::now();
        sm.votes_received = 0;

        let mut ds = self.state.write().await;
        ds.is_leader = false;
        ds.raft_term = sm.current_term;

        Ok(true)
    }

    /// Handles an incoming RequestVote RPC.
    pub async fn handle_request_vote(
        &self,
        term: u64,
        candidate_id: &str,
        _last_log_index: u64,
        _last_log_term: u64,
    ) -> Result<bool, AstraSyncError> {
        let mut sm = self.raft_state.write().await;
        if term < sm.current_term {
            return Ok(false);
        }
        if term > sm.current_term {
            sm.current_term = term;
            sm.voted_for = None;
            sm.state = RaftState::Follower;
        }
        if let Some(ref voted) = sm.voted_for {
            if voted != candidate_id {
                return Ok(false);
            }
        }
        sm.voted_for = Some(candidate_id.to_string());
        sm.last_leader_heartbeat = Instant::now();
        let _ = self
            .event_tx
            .send(RaftEvent::VoteGranted {
                term,
                voter: sm.my_id.clone(),
            })
            .await;
        Ok(true)
    }

    /// Returns the current Raft state (for gRPC/health checks).
    pub async fn current_state(&self) -> (RaftState, u64, Option<String>) {
        let sm = self.raft_state.read().await;
        (sm.state, sm.current_term, sm.leader_id.clone())
    }
}

/// Computes a randomized election timeout in the configured range.
fn election_timeout(cfg: &RaftConfig) -> Duration {
    let mut rng = rand::thread_rng();
    let ms = rng.gen_range(cfg.election_timeout_min_ms..=cfg.election_timeout_max_ms);
    Duration::from_millis(ms)
}

/// Broadcasts a heartbeat to all known peers over the P2P mesh.
async fn broadcast_heartbeat(
    term: u64,
    leader_id: String,
    mesh: &P2PMeshHandle,
) -> Result<(), AstraSyncError> {
    trace!(term, "Broadcasting Raft heartbeat");
    let peers = mesh.peers().await;
    for peer in peers {
        let heartbeat = SyncProtocol::RaftHeartbeat(crate::protocol::RaftHeartbeat {
            term,
            leader_id: leader_id.clone(),
        });
        if let Err(e) = mesh.send_request(peer, heartbeat).await {
            warn!(%peer, %e, "Failed to send Raft heartbeat");
        }
    }
    Ok(())
}

/// Requests votes from all known peers during an election.
async fn request_votes(
    term: u64,
    candidate_id: String,
    mesh: &P2PMeshHandle,
) -> Result<(), AstraSyncError> {
    info!(term, "Requesting votes from peers");
    let peers = mesh.peers().await;
    for peer in peers {
        let vote_req = SyncProtocol::RaftRequestVote(crate::protocol::RaftRequestVote {
            term,
            candidate_id: candidate_id.clone(),
            last_log_index: 0,
            last_log_term: 0,
        });
        if let Err(e) = mesh.send_request(peer, vote_req).await {
            warn!(%peer, %e, "Failed to send RequestVote");
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_election_timeout_randomness() {
        let cfg = RaftConfig {
            heartbeat_interval_ms: 500,
            election_timeout_min_ms: 1500,
            election_timeout_max_ms: 3000,
            max_entries_per_append: 128,
        };
        let t1 = election_timeout(&cfg);
        let t2 = election_timeout(&cfg);
        assert!(t1 >= Duration::from_millis(1500));
        assert!(t1 <= Duration::from_millis(3000));
        assert!(t2 >= Duration::from_millis(1500));
        assert!(t2 <= Duration::from_millis(3000));
    }
}
