#![deny(unsafe_code)]

//! Binary sync protocol for P2P mesh communication.
//!
//! The protocol is deliberately simple and compact: every message is a
//! length-prefixed bincode-encoded frame carrying one of three message types.
//! Frames are encrypted at the transport layer (Noise over QUIC) before being
//! placed on the wire.
//!
//! Message types:
//! * [`SyncHandshake`] — initial capability exchange and session nonce.
//! * [`SyncDelta`] — a bundle of CRDT records to be applied by the peer.
//! * [`SyncAck`] — acknowledgement of received deltas with sync watermarks.

use bytes::{Buf, BufMut, Bytes, BytesMut};
use serde::{Deserialize, Serialize};

use crate::AstraSyncError;
use crate::crdt::hlc::Hlc;

/// Magic bytes at the start of every frame.
pub const SYNC_MAGIC: [u8; 4] = [0x41, 0x53, 0x54, 0x52]; // "ASTR"

/// Current protocol version.
pub const PROTOCOL_VERSION: u8 = 1;

/// Maximum allowed payload size for a single frame (16 MiB).
pub const MAX_FRAME_SIZE: usize = 16 * 1024 * 1024;

/// Top-level protocol message.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum SyncProtocol {
    /// Handshake sent when a sync session is established.
    Handshake(SyncHandshake),
    /// One or more CRDT records propagated from peer to peer.
    Delta(SyncDelta),
    /// Acknowledgement that a delta was received and applied up to a watermark.
    Ack(SyncAck),
    /// Raft heartbeat from the current leader.
    RaftHeartbeat(RaftHeartbeat),
    /// Raft RequestVote RPC from a candidate.
    RaftRequestVote(RaftRequestVote),
    /// Raft response to a RequestVote RPC.
    RaftVoteResponse(RaftVoteResponse),
}

impl SyncProtocol {
    /// Serializes the message to bincode bytes.
    pub fn to_bytes(&self) -> Result<Vec<u8>, AstraSyncError> {
        bincode::serialize(self)
            .map_err(|e| AstraSyncError::Serialization(format!("bincode serialize failed: {e}")))
    }

    /// Deserializes a message from bincode bytes.
    pub fn from_bytes(bytes: &[u8]) -> Result<Self, AstraSyncError> {
        bincode::deserialize(bytes)
            .map_err(|e| AstraSyncError::Serialization(format!("bincode deserialize failed: {e}")))
    }

    /// Encodes this message into a length-prefixed wire frame.
    pub fn encode(&self) -> Result<Bytes, AstraSyncError> {
        let payload = self.to_bytes()?;
        if payload.len() > MAX_FRAME_SIZE {
            return Err(AstraSyncError::P2P(format!(
                "protocol frame payload too large: {} bytes (max {})",
                payload.len(),
                MAX_FRAME_SIZE
            )));
        }
        let total_len = SYNC_MAGIC.len() + 1 + 4 + payload.len();
        let mut buf = BytesMut::with_capacity(total_len);
        buf.put_slice(&SYNC_MAGIC);
        buf.put_u8(PROTOCOL_VERSION);
        buf.put_u32(payload.len() as u32);
        buf.put_slice(&payload);
        Ok(buf.freeze())
    }

    /// Attempts to decode the next frame from `buf`.
    ///
    /// Returns `None` if `buf` does not yet contain a complete frame.  Returns
    /// `Some(Err(...))` if the frame is malformed.
    pub fn decode(buf: &mut BytesMut) -> Option<Result<Self, AstraSyncError>> {
        if buf.len() < SYNC_MAGIC.len() + 1 + 4 {
            return None;
        }
        if &buf[0..SYNC_MAGIC.len()] != SYNC_MAGIC {
            // Stream misalignment: consume one byte and let the caller retry.
            buf.advance(1);
            return Some(Err(AstraSyncError::P2P(
                "protocol frame magic mismatch".to_string(),
            )));
        }
        let version = buf[SYNC_MAGIC.len()];
        if version != PROTOCOL_VERSION {
            buf.advance(SYNC_MAGIC.len() + 1);
            return Some(Err(AstraSyncError::P2P(format!(
                "unsupported protocol version: {version}"
            ))));
        }
        let payload_len = u32::from_be_bytes([
            buf[SYNC_MAGIC.len() + 1],
            buf[SYNC_MAGIC.len() + 2],
            buf[SYNC_MAGIC.len() + 3],
            buf[SYNC_MAGIC.len() + 4],
        ]) as usize;
        if payload_len > MAX_FRAME_SIZE {
            buf.advance(SYNC_MAGIC.len() + 1 + 4);
            return Some(Err(AstraSyncError::P2P(format!(
                "protocol frame payload length exceeds maximum: {payload_len}"
            ))));
        }
        let frame_len = SYNC_MAGIC.len() + 1 + 4 + payload_len;
        if buf.len() < frame_len {
            return None;
        }
        let payload = buf[SYNC_MAGIC.len() + 5..frame_len].to_vec();
        buf.advance(frame_len);
        Some(Self::from_bytes(&payload))
    }
}

/// Handshake message exchanged when two peers establish a sync session.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SyncHandshake {
    /// Protocol version supported by the sender.
    pub protocol_version: u8,
    /// Identity of the sending kiosk.
    pub kiosk_id: String,
    /// Data types this peer is willing to sync (using the [`DataType`](crate::DataType) repr).
    pub supported_data_types: Vec<u8>,
    /// Session nonce used to prevent replay of handshake messages.
    pub nonce: Vec<u8>,
}

/// A delta of CRDT records propagated during a sync session.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SyncDelta {
    /// Opaque session identifier echoed in the corresponding ack.
    pub session_id: String,
    /// Data type being synced.
    pub data_type: u8,
    /// Sequence number within the session (monotonic from zero).
    pub sequence_number: u64,
    /// Serialized CRDT records.  The interpretation depends on `data_type`.
    pub records: Vec<Vec<u8>>,
}

/// Acknowledgement that deltas up to a given HLC have been received.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SyncAck {
    /// Opaque session identifier.
    pub session_id: String,
    /// Number of records successfully received.
    pub received_count: u64,
    /// Highest HLC observed in the received records.
    pub last_hlc: Hlc,
}

/// Raft heartbeat from the current leader.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RaftHeartbeat {
    /// Current Raft term.
    pub term: u64,
    /// Leader kiosk identifier.
    pub leader_id: String,
}

/// Raft RequestVote RPC payload.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RaftRequestVote {
    /// Current Raft term.
    pub term: u64,
    /// Candidate kiosk identifier.
    pub candidate_id: String,
    /// Index of the candidate's last log entry.
    pub last_log_index: u64,
    /// Term of the candidate's last log entry.
    pub last_log_term: u64,
}

/// Raft response to a RequestVote RPC.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RaftVoteResponse {
    /// Current Raft term.
    pub term: u64,
    /// True if the vote was granted.
    pub vote_granted: bool,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn handshake_roundtrip() {
        let msg = SyncProtocol::Handshake(SyncHandshake {
            protocol_version: PROTOCOL_VERSION,
            kiosk_id: "kiosk-1".to_string(),
            supported_data_types: vec![0, 1, 2],
            nonce: vec![1, 2, 3, 4],
        });
        let encoded = msg.encode().expect("encode");
        let mut buf = BytesMut::from(encoded.as_ref());
        let decoded = SyncProtocol::decode(&mut buf).expect("frame present").expect("decode ok");
        assert_eq!(msg, decoded);
    }

    #[test]
    fn delta_roundtrip() {
        let msg = SyncProtocol::Delta(SyncDelta {
            session_id: "sess-abc".to_string(),
            data_type: 0,
            sequence_number: 7,
            records: vec![vec![10, 20, 30], vec![40, 50]],
        });
        let encoded = msg.encode().expect("encode");
        let mut buf = BytesMut::from(encoded.as_ref());
        let decoded = SyncProtocol::decode(&mut buf).expect("frame present").expect("decode ok");
        assert_eq!(msg, decoded);
    }

    #[test]
    fn ack_roundtrip() {
        let msg = SyncProtocol::Ack(SyncAck {
            session_id: "sess-abc".to_string(),
            received_count: 42,
            last_hlc: Hlc::new("peer").expect("valid node id"),
        });
        let encoded = msg.encode().expect("encode");
        let mut buf = BytesMut::from(encoded.as_ref());
        let decoded = SyncProtocol::decode(&mut buf).expect("frame present").expect("decode ok");
        assert_eq!(msg, decoded);
    }

    #[test]
    fn decode_returns_none_until_complete() {
        let msg = SyncProtocol::Handshake(SyncHandshake {
            protocol_version: PROTOCOL_VERSION,
            kiosk_id: "k1".to_string(),
            supported_data_types: vec![0],
            nonce: vec![9],
        });
        let encoded = msg.encode().expect("encode");
        let mut buf = BytesMut::from(encoded.as_ref());
        let head = buf.split_off(5);
        assert!(SyncProtocol::decode(&mut buf).is_none());
        buf.unsplit(head);
        let decoded = SyncProtocol::decode(&mut buf).expect("frame present").expect("decode ok");
        assert_eq!(decoded, msg);
    }

    #[test]
    fn decode_rejects_bad_magic() {
        let mut buf = BytesMut::from(&[0u8; 20][..]);
        assert!(SyncProtocol::decode(&mut buf).unwrap().is_err());
    }
}
