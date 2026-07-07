#![deny(unsafe_code)]

//! Custom sync protocol definitions and message framing for P2P mesh communication.
//!
//! This module defines the wire format for sync messages, including:
//! - `SyncMessage`: the top-level envelope for all P2P sync payloads.
//! - `MessageType`: discriminant for routing to the correct CRDT handler.
//! - `SyncFrame`: a length-prefixed, encrypted frame used on QUIC streams.
//!
//! All messages are serialized with bincode for compactness, then encrypted
//! with XChaCha20-Poly1305 before transmission.

use bytes::{Buf, BufMut, Bytes, BytesMut};
use serde::{Deserialize, Serialize};
use tracing::{debug, trace, warn};

use crate::crypto::{SyncKey, decrypt_sync_message, encrypt_sync_message};
use crate::{AstraSyncError, DataType, KioskId};

/// Magic bytes at the start of every sync frame for quick identification.
pub const SYNC_MAGIC: [u8; 4] = [0x41, 0x53, 0x54, 0x52]; // "ASTR"

/// Current protocol version (semver-inspired: 0x01 = v1).
pub const SYNC_VERSION: u8 = 0x01;

/// Maximum frame payload size (1 MiB).
pub const MAX_FRAME_SIZE: usize = 1024 * 1024;

/// Top-level message envelope sent over the P2P mesh.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncMessage {
    /// Protocol version.
    pub version: u8,
    /// The kiosk that originated this message.
    pub origin: KioskId,
    /// Lamport timestamp at the origin.
    pub lamport_ts: u64,
    /// Wall-clock timestamp.
    pub wallclock_ts: u64,
    /// The actual payload.
    pub payload: MessagePayload,
}

impl SyncMessage {
    pub fn new(origin: KioskId, lamport_ts: u64, wallclock_ts: u64, payload: MessagePayload) -> Self {
        Self { version: SYNC_VERSION, origin, lamport_ts, wallclock_ts, payload }
    }

    /// Serializes the message to bincode bytes.
    pub fn to_bytes(&self) -> Result<Vec<u8>, AstraSyncError> {
        bincode::serialize(self)
            .map_err(|e| AstraSyncError::Serialization(format!("bincode serialize failed: {e}")))
    }

    /// Deserializes from bincode bytes.
    pub fn from_bytes(bytes: &[u8]) -> Result<Self, AstraSyncError> {
        bincode::deserialize(bytes)
            .map_err(|e| AstraSyncError::Serialization(format!("bincode deserialize failed: {e}")))
    }

    /// Encrypts the message into a `SyncFrame`.
    pub fn encrypt(&self, key: &SyncKey) -> Result<SyncFrame, AstraSyncError> {
        let plaintext = self.to_bytes()?;
        let (nonce, ciphertext) = encrypt_sync_message(key, &plaintext)?;
        Ok(SyncFrame::new(nonce, ciphertext))
    }

    /// Decrypts a `SyncFrame` into a `SyncMessage`.
    pub fn decrypt(frame: &SyncFrame, key: &SyncKey) -> Result<Self, AstraSyncError> {
        let plaintext = decrypt_sync_message(key, &frame.nonce, &frame.ciphertext)?;
        Self::from_bytes(&plaintext)
    }
}

/// The payload variants carried by a sync message.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MessagePayload {
    /// Inventory update (LWW register value).
    InventoryUpdate {
        record_id: String,
        sku: String,
        count: u64,
        unit_price_cents: u64,
        location: String,
    },
    /// Cart state update (LWW register value).
    CartUpdate {
        record_id: String,
        cart_id: String,
        items_json: String,
        total_cents: u64,
        status: String,
    },
    /// Transaction batch (one or more transactions).
    TransactionBatch {
        transactions_json: Vec<String>,
    },
    /// Analytics event.
    AnalyticsEvent {
        event_id: String,
        event_type: String,
        metadata_json: String,
    },
    /// Raft AppendEntries RPC.
    RaftAppendEntries {
        term: u64,
        leader_id: String,
        prev_log_index: u64,
        prev_log_term: u64,
        entries: Vec<RaftLogEntry>,
        leader_commit: u64,
    },
    /// Raft RequestVote RPC.
    RaftRequestVote {
        term: u64,
        candidate_id: String,
        last_log_index: u64,
        last_log_term: u64,
    },
    /// Raft vote response.
    RaftVoteResponse {
        term: u64,
        vote_granted: bool,
    },
    /// Raft AppendEntries response.
    RaftAppendResponse {
        term: u64,
        success: bool,
        match_index: u64,
    },
    /// Heartbeat / keepalive.
    Heartbeat {
        leader_id: String,
        term: u64,
    },
}

/// A single Raft log entry embedded in an AppendEntries payload.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RaftLogEntry {
    pub term: u64,
    pub index: u64,
    pub command_type: u8,
    pub command_data: Vec<u8>,
}

/// A framed, encrypted message suitable for transmission over a stream.
/// Wire format:
///   [4 bytes magic] [1 byte version] [4 bytes nonce length] [N bytes nonce]
///   [4 bytes ciphertext length] [M bytes ciphertext]
#[derive(Debug, Clone)]
pub struct SyncFrame {
    pub nonce: Vec<u8>,
    pub ciphertext: Vec<u8>,
}

impl SyncFrame {
    pub fn new(nonce: Vec<u8>, ciphertext: Vec<u8>) -> Self {
        Self { nonce, ciphertext }
    }

    /// Encodes the frame into a byte buffer suitable for writing to a stream.
    pub fn encode(&self) -> Result<Bytes, AstraSyncError> {
        let total_len = 4 + 1 + 4 + self.nonce.len() + 4 + self.ciphertext.len();
        if total_len > MAX_FRAME_SIZE + 13 {
            return Err(AstraSyncError::P2P(format!(
                "frame too large: {} bytes (max {})", total_len, MAX_FRAME_SIZE
            )));
        }
        let mut buf = BytesMut::with_capacity(total_len);
        buf.put_slice(&SYNC_MAGIC);
        buf.put_u8(SYNC_VERSION);
        buf.put_u32(self.nonce.len() as u32);
        buf.put_slice(&self.nonce);
        buf.put_u32(self.ciphertext.len() as u32);
        buf.put_slice(&self.ciphertext);
        Ok(buf.freeze())
    }

    /// Decodes a frame from a byte buffer read from a stream.
    /// Returns `None` if the buffer doesn't contain a complete frame yet.
    pub fn decode(buf: &mut BytesMut) -> Option<Result<Self, AstraSyncError>> {
        if buf.len() < 9 {
            return None; // Need at least magic + version + nonce length
        }
        let magic = &buf[0..4];
        if magic != SYNC_MAGIC {
            // Corrupt or misaligned stream. Consume one byte and try again.
            buf.advance(1);
            return Some(Err(AstraSyncError::P2P(
                "frame magic mismatch — stream may be misaligned".to_string()
            )));
        }
        let version = buf[4];
        if version != SYNC_VERSION {
            buf.advance(5);
            return Some(Err(AstraSyncError::P2P(format!(
                "unsupported protocol version: {version}"
            ))));
        }
        let nonce_len = u32::from_be_bytes([buf[5], buf[6], buf[7], buf[8]]) as usize;
        let header_len = 9 + 4 + nonce_len;
        if buf.len() < header_len {
            return None; // Need more data for nonce + ciphertext length
        }
        let ct_len = u32::from_be_bytes([
            buf[9 + nonce_len],
            buf[10 + nonce_len],
            buf[11 + nonce_len],
            buf[12 + nonce_len],
        ]) as usize;
        let total_len = header_len + ct_len;
        if buf.len() < total_len {
            return None; // Need more data for ciphertext
        }
        let nonce = buf[9..9 + nonce_len].to_vec();
        let ciphertext = buf[13 + nonce_len..13 + nonce_len + ct_len].to_vec();
        buf.advance(total_len);
        Some(Ok(Self { nonce, ciphertext }))
    }
}

/// A stream parser that continuously reads `SyncFrame`s from an async byte stream.
pub struct FrameReader;

impl FrameReader {
    /// Attempts to parse the next frame from the provided buffer.
    /// If the buffer is exhausted, returns `None`. If a frame is parsed,
    /// returns `Some(Ok(frame))`. If an error occurs, returns `Some(Err(...))`.
    pub fn next_frame(buf: &mut BytesMut) -> Option<Result<SyncFrame, AstraSyncError>> {
        SyncFrame::decode(buf)
    }
}

/// Computes the priority-based topic name for a given data type.
pub fn topic_for(data_type: DataType) -> String {
    format!("astra/{}", data_type.as_str())
}

/// Computes the priority queue weight: lower number = higher priority.
pub fn priority_weight(data_type: DataType) -> u8 {
    match data_type.priority() {
        crate::SyncPriority::Immediate => 0,
        crate::SyncPriority::Batched => 1,
        crate::SyncPriority::Delayed => 2,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sync_message_roundtrip() {
        let msg = SyncMessage::new(
            KioskId::from("k1"),
            42,
            1000,
            MessagePayload::Heartbeat { leader_id: "k1".to_string(), term: 1 },
        );
        let bytes = msg.to_bytes().unwrap();
        let msg2 = SyncMessage::from_bytes(&bytes).unwrap();
        assert_eq!(msg.origin, msg2.origin);
        assert_eq!(msg.lamport_ts, msg2.lamport_ts);
    }

    #[test]
    fn test_sync_frame_roundtrip() {
        let key = SyncKey::generate();
        let msg = SyncMessage::new(
            KioskId::from("k1"),
            1,
            1000,
            MessagePayload::Heartbeat { leader_id: "k1".to_string(), term: 1 },
        );
        let frame = msg.encrypt(&key).unwrap();
        let encoded = frame.encode().unwrap();
        let mut buf = BytesMut::from(encoded.as_ref());
        let decoded = SyncFrame::decode(&mut buf).unwrap().unwrap();
        let decrypted = SyncMessage::decrypt(&decoded, &key).unwrap();
        assert_eq!(msg.origin, decrypted.origin);
    }

    #[test]
    fn test_priority_weight() {
        assert_eq!(priority_weight(DataType::Inventory), 0);
        assert_eq!(priority_weight(DataType::Transaction), 1);
        assert_eq!(priority_weight(DataType::Analytics), 2);
    }
}
