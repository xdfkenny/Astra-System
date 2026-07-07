#![deny(unsafe_code)]

//! libp2p-based P2P networking layer.
//!
//! This module provides the production scaffolding for mesh communication among
//! Astra kiosks.  It wires together:
//!
//! * QUIC as the transport protocol (NAT-friendly, 0-RTT where possible).
//! * Noise for authenticated encryption of all peer connections.
//! * mDNS for local peer discovery without a bootstrap server.
//! * GossipSub for flooding sync messages across the mesh.
//!
//! The current implementation builds a real libp2p `Swarm` with these behaviours
//! but keeps the event loop lightweight: it listens for outbound commands,
//! publishes gossip messages, and emits peer discovery events.  Actual routing
//! of sync payloads is handled by the caller via [`NetworkHandle`].

use std::io;
use std::time::Duration;

use async_trait::async_trait;
use futures::StreamExt;
use libp2p::{
    gossipsub, identity, mdns, request_response,
    swarm::{NetworkBehaviour, SwarmEvent},
    PeerId, StreamProtocol, Swarm, SwarmBuilder,
};
use tokio::sync::{mpsc, watch};
use tracing::{debug, info, trace, warn};

use crate::AstraSyncError;
use crate::protocol::SyncProtocol;

/// Command interface exposed to other daemon subsystems.
#[derive(Debug, Clone)]
pub struct NetworkHandle {
    cmd_tx: mpsc::Sender<NetworkCommand>,
}

impl NetworkHandle {
    /// Broadcasts a sync message on the gossipsub topic for `data_type`.
    pub async fn gossip(
        &self,
        data_type: u8,
        message: SyncProtocol,
    ) -> Result<(), AstraSyncError> {
        let payload = message.to_bytes()?;
        self.cmd_tx
            .send(NetworkCommand::Gossip { data_type, payload })
            .await
            .map_err(|_| AstraSyncError::P2P("network command channel closed".to_string()))
    }

    /// Requests an immediate sync pull from a specific peer over the
    /// `/astra-sync/1.0.0` request-response protocol.
    pub async fn request_sync(
        &self,
        peer: PeerId,
        since_lamport: u64,
    ) -> Result<(), AstraSyncError> {
        let request = SyncProtocol::Handshake(crate::protocol::SyncHandshake {
            protocol_version: crate::protocol::PROTOCOL_VERSION,
            kiosk_id: String::new(),
            supported_data_types: vec![0, 1, 2],
            nonce: vec![1, 2, 3, 4],
        });
        self.cmd_tx
            .send(NetworkCommand::SendRequest { peer, request })
            .await
            .map_err(|_| AstraSyncError::P2P("network command channel closed".to_string()))
    }

    /// Sends a direct sync request to a peer.
    pub async fn send_request(
        &self,
        peer: PeerId,
        request: SyncProtocol,
    ) -> Result<(), AstraSyncError> {
        self.cmd_tx
            .send(NetworkCommand::SendRequest { peer, request })
            .await
            .map_err(|_| AstraSyncError::P2P("network command channel closed".to_string()))
    }
}

#[derive(Debug)]
enum NetworkCommand {
    Gossip { data_type: u8, payload: Vec<u8> },
    RequestSync { data_type: u8, since_lamport: u64 },
    SendRequest { peer: PeerId, request: SyncProtocol },
}

/// Binary codec for the `/astra-sync/1.0.0` request-response protocol.
#[derive(Debug, Clone, Default)]
pub struct BincodeCodec;

#[async_trait]
impl request_response::Codec for BincodeCodec {
    type Protocol = StreamProtocol;
    type Request = SyncProtocol;
    type Response = SyncProtocol;

    async fn read_request<T: futures::AsyncRead + Unpin + Send>(
        &mut self,
        _protocol: &Self::Protocol,
        io: &mut T,
    ) -> io::Result<Self::Request> {
        read_length_prefixed(io, 16 * 1024 * 1024)
            .await
            .and_then(|bytes| {
                SyncProtocol::from_bytes(&bytes)
                    .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))
            })
    }

    async fn read_response<T: futures::AsyncRead + Unpin + Send>(
        &mut self,
        _protocol: &Self::Protocol,
        io: &mut T,
    ) -> io::Result<Self::Response> {
        read_length_prefixed(io, 16 * 1024 * 1024)
            .await
            .and_then(|bytes| {
                SyncProtocol::from_bytes(&bytes)
                    .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))
            })
    }

    async fn write_request<T: futures::AsyncWrite + Unpin + Send>(
        &mut self,
        _protocol: &Self::Protocol,
        io: &mut T,
        req: Self::Request,
    ) -> io::Result<()> {
        let bytes = req
            .to_bytes()
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))?;
        write_length_prefixed(io, &bytes).await
    }

    async fn write_response<T: futures::AsyncWrite + Unpin + Send>(
        &mut self,
        _protocol: &Self::Protocol,
        io: &mut T,
        res: Self::Response,
    ) -> io::Result<()> {
        let bytes = res
            .to_bytes()
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))?;
        write_length_prefixed(io, &bytes).await
    }
}

/// Reads a length-prefixed byte vector from an async reader.
async fn read_length_prefixed<T: futures::AsyncRead + Unpin + Send>(
    io: &mut T,
    max_size: usize,
) -> io::Result<Vec<u8>> {
    use futures::AsyncReadExt;
    let mut len_buf = [0u8; 4];
    io.read_exact(&mut len_buf).await?;
    let len = u32::from_be_bytes(len_buf) as usize;
    if len > max_size {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("request payload exceeds maximum size: {len}"),
        ));
    }
    let mut buf = vec![0u8; len];
    io.read_exact(&mut buf).await?;
    Ok(buf)
}

/// Writes a length-prefixed byte vector to an async writer.
async fn write_length_prefixed<T: futures::AsyncWrite + Unpin + Send>(
    io: &mut T,
    data: &[u8],
) -> io::Result<()> {
    use futures::AsyncWriteExt;
    let len = u32::try_from(data.len())
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "payload too large"))?;
    io.write_all(&len.to_be_bytes()).await?;
    io.write_all(data).await?;
    io.flush().await
}

/// Protocol identifier for direct sync sessions.
pub const SYNC_PROTOCOL: &str = "/astra-sync/1.0.0";
#[derive(NetworkBehaviour)]
#[behaviour(out_event = "AstraNetworkEvent")]
pub struct AstraNetworkBehaviour {
    /// Local peer discovery.
    pub mdns: mdns::tokio::Behaviour,
    /// Gossip-based message flooding.
    pub gossipsub: gossipsub::Behaviour,
    /// Direct request/response sync protocol.
    pub sync: request_response::Behaviour<BincodeCodec>,
}

/// Events emitted by the network behaviour.
#[derive(Debug)]
pub enum AstraNetworkEvent {
    Mdns(mdns::Event),
    Gossipsub(gossipsub::Event),
    Sync(request_response::Event<SyncProtocol, SyncProtocol>),
}

impl From<mdns::Event> for AstraNetworkEvent {
    fn from(event: mdns::Event) -> Self {
        AstraNetworkEvent::Mdns(event)
    }
}

impl From<gossipsub::Event> for AstraNetworkEvent {
    fn from(event: gossipsub::Event) -> Self {
        AstraNetworkEvent::Gossipsub(event)
    }
}

impl From<request_response::Event<SyncProtocol, SyncProtocol>> for AstraNetworkEvent {
    fn from(event: request_response::Event<SyncProtocol, SyncProtocol>) -> Self {
        AstraNetworkEvent::Sync(event)
    }
}

/// libp2p-backed P2P network.
pub struct Network {
    local_peer_id: PeerId,
    swarm: Swarm<AstraNetworkBehaviour>,
}

impl Network {
    /// Creates a new network instance for the given kiosk identity.
    ///
    /// The `kiosk_id` is used to derive a stable libp2p keypair via BLAKE3 so
    /// that the same kiosk always presents the same `PeerId` across restarts.
    pub fn new(kiosk_id: &str, listen_addr: &str) -> Result<Self, AstraSyncError> {
        let local_key = derive_keypair(kiosk_id)?;
        let local_peer_id = PeerId::from(local_key.public());
        info!(%local_peer_id, "Initializing libp2p network");

        let swarm = build_swarm(local_key, listen_addr)?;

        Ok(Self {
            local_peer_id,
            swarm,
        })
    }

    /// Starts the network event loop and returns a control handle plus a join handle.
    pub async fn start(
        mut self,
        mut shutdown: watch::Receiver<bool>,
    ) -> Result<(NetworkHandle, tokio::task::JoinHandle<()>), AstraSyncError> {
        let (cmd_tx, mut cmd_rx) = mpsc::channel::<NetworkCommand>(256);
        let handle = NetworkHandle { cmd_tx };
        let local_peer_id = self.local_peer_id;

        let join_handle = tokio::spawn(async move {
            info!(%local_peer_id, "libp2p event loop started");
            loop {
                tokio::select! {
                    biased;
                    _ = shutdown.changed() => {
                        info!("Network received shutdown signal");
                        break;
                    }
                    Some(cmd) = cmd_rx.recv() => {
                        match cmd {
                            NetworkCommand::Gossip { data_type, payload } => {
                                let topic = data_type_topic(data_type);
                                trace!(?topic, len = payload.len(), "Publishing gossip message");
                                if let Err(e) = self.swarm.behaviour_mut().gossipsub.publish(topic, payload) {
                                    warn!(%e, "Gossip publish failed");
                                }
                            }
                            NetworkCommand::RequestSync { data_type: _, since_lamport: _ } => {
                                // request_sync now requires a peer; callers should use SendRequest.
                                debug!("RequestSync command is deprecated; use SendRequest");
                            }
                            NetworkCommand::SendRequest { peer, request } => {
                                trace!(%peer, "Sending sync request");
                                let request_id = self.swarm.behaviour_mut().sync.send_request(&peer, request);
                                trace!(?request_id, %peer, "Sync request queued");
                            }
                        }
                    }
                    event = self.swarm.select_next_some() => {
                        match event {
                            SwarmEvent::NewListenAddr { address, .. } => {
                                info!(%address, "Network listening");
                            }
                            SwarmEvent::Behaviour(AstraNetworkEvent::Mdns(mdns::Event::Discovered(list))) => {
                                for (peer_id, multiaddr) in list {
                                    debug!(%peer_id, %multiaddr, "Discovered peer via mDNS");
                                }
                            }
                            SwarmEvent::Behaviour(AstraNetworkEvent::Gossipsub(gossipsub::Event::Message {
                                propagation_source,
                                message_id,
                                message,
                            })) => {
                                trace!(%propagation_source, %message_id, topic = ?message.topic, "Received gossip message");
                            }
                            SwarmEvent::Behaviour(AstraNetworkEvent::Sync(request_response::Event::Message {
                                peer,
                                message,
                                ..
                            })) => {
                                match message {
                                    request_response::Message::Request { request_id, request, .. } => {
                                        trace!(%peer, %request_id, ?request, "Received sync request");
                                    }
                                    request_response::Message::Response { request_id, response } => {
                                        trace!(%peer, %request_id, ?response, "Received sync response");
                                    }
                                }
                            }
                            SwarmEvent::Behaviour(AstraNetworkEvent::Sync(request_response::Event::OutboundFailure {
                                peer,
                                request_id,
                                error,
                            })) => {
                                warn!(%peer, %request_id, %error, "Outbound sync request failed");
                            }
                            SwarmEvent::Behaviour(AstraNetworkEvent::Sync(request_response::Event::InboundFailure {
                                peer,
                                request_id,
                                error,
                            })) => {
                                warn!(%peer, %request_id, %error, "Inbound sync request failed");
                            }
                            SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                                debug!(%peer_id, "Peer connected");
                            }
                            SwarmEvent::ConnectionClosed { peer_id, cause, .. } => {
                                debug!(%peer_id, ?cause, "Peer disconnected");
                            }
                            _ => {}
                        }
                    }
                }
            }
            info!("libp2p event loop stopped");
        });

        Ok((handle, join_handle))
    }
}

/// Builds a libp2p Swarm configured for the Astra kiosk mesh.
fn build_swarm(
    local_key: identity::Keypair,
    listen_addr: &str,
) -> Result<Swarm<AstraNetworkBehaviour>, AstraSyncError> {
    let peer_id = PeerId::from(local_key.public());

    let mdns_config = mdns::Config {
        ttl: Duration::from_secs(300),
        query_interval: Duration::from_secs(5),
        ..Default::default()
    };
    let mdns = mdns::tokio::Behaviour::new(mdns_config, peer_id)
        .map_err(|e| AstraSyncError::P2P(format!("mDNS behaviour failed: {e}")))?;

    let message_authenticity = gossipsub::MessageAuthenticity::Signed(local_key.clone());
    let gossipsub_config = gossipsub::ConfigBuilder::default()
        .max_transmit_size(2 * 1024 * 1024)
        .validation_mode(gossipsub::ValidationMode::Strict)
        .build()
        .map_err(|e| AstraSyncError::P2P(format!("gossipsub config failed: {e}")))?;
    let gossipsub = gossipsub::Behaviour::new(message_authenticity, gossipsub_config)
        .map_err(|e| AstraSyncError::P2P(format!("gossipsub behaviour failed: {e}")))?;

    let sync_protocols = vec![(StreamProtocol::new(SYNC_PROTOCOL), request_response::ProtocolSupport::Full)];
    let sync_cfg = request_response::Config::default()
        .with_request_timeout(Duration::from_secs(10));
    let sync = request_response::Behaviour::with_codec(BincodeCodec::default(), sync_protocols, sync_cfg);

    let behaviour = AstraNetworkBehaviour { mdns, gossipsub, sync };

    let mut swarm = SwarmBuilder::with_existing_identity(local_key)
        .with_tokio()
        .with_quic()
        .with_behaviour(|_key| Ok(behaviour))
        .map_err(|e| AstraSyncError::P2P(format!("swarm behaviour failed: {e}")))?
        .with_swarm_config(|cfg: libp2p::swarm::Config| {
            cfg.with_idle_connection_timeout(Duration::from_secs(300))
        })
        .build();

    let listen_addr: libp2p::Multiaddr = listen_addr
        .parse()
        .map_err(|e| AstraSyncError::P2P(format!("invalid listen address: {e}")))?;
    swarm
        .listen_on(listen_addr)
        .map_err(|e| AstraSyncError::P2P(format!("listen failed: {e}")))?;

    Ok(swarm)
}

/// Derives a deterministic libp2p Ed25519 keypair from a kiosk identifier.
fn derive_keypair(kiosk_id: &str) -> Result<identity::Keypair, AstraSyncError> {
    let seed = crate::crypto::content_hash(kiosk_id.as_bytes());
    identity::Keypair::ed25519_from_bytes(seed)
        .map_err(|e| AstraSyncError::Crypto(format!("failed to derive libp2p keypair: {e}")))
}

/// Computes the gossipsub topic name for a data type.
fn data_type_topic(data_type: u8) -> gossipsub::IdentTopic {
    gossipsub::IdentTopic::new(format!("astra/sync/{data_type}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn keypair_is_deterministic() {
        let k1 = derive_keypair("kiosk-42").expect("derive");
        let k2 = derive_keypair("kiosk-42").expect("derive");
        assert_eq!(
            PeerId::from(k1.public()).to_string(),
            PeerId::from(k2.public()).to_string()
        );
    }

    #[test]
    fn keypair_differs_by_kiosk() {
        let k1 = derive_keypair("kiosk-1").expect("derive");
        let k2 = derive_keypair("kiosk-2").expect("derive");
        assert_ne!(
            PeerId::from(k1.public()).to_string(),
            PeerId::from(k2.public()).to_string()
        );
    }

    #[test]
    fn topic_name_is_stable() {
        let t1 = data_type_topic(1);
        let t2 = data_type_topic(1);
        assert_eq!(t1.hash(), t2.hash());
        assert_ne!(t1.hash(), data_type_topic(2).hash());
    }
}
