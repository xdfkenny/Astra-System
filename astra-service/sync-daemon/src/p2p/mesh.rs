#![deny(unsafe_code)]

//! libp2p-based P2P mesh networking layer.
//!
//! This module provides the production P2P transport for Astra kiosks:
//!
//! * QUIC as the transport protocol (NAT-friendly, connection migration).
//! * Noise XX handshake for authenticated encryption.
//! * mDNS for local LAN peer discovery without a bootstrap server.
//! * GossipSub for flooding sync messages across the mesh.
//! * A custom `/astra-sync/1.0.0` request-response protocol for direct sync
//!   sessions, delta calculation, and acknowledgments.
//!
//! The public API ([`P2PMesh`], [`P2PMeshHandle`]) is used by the sync engine
//! and gRPC service.  All libp2p specifics are encapsulated here.

use std::collections::HashSet;
use std::io;
use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use futures::StreamExt;
use libp2p::{
    gossipsub, identity, mdns, request_response,
    swarm::{NetworkBehaviour, SwarmEvent},
    Multiaddr, PeerId, StreamProtocol, Swarm, SwarmBuilder,
};
use opentelemetry::Context;
use tokio::sync::{broadcast, mpsc, watch, Mutex};
use tracing::{debug, info, span, trace, warn, Level, Span};

use crate::config::Config;
use crate::protocol::{SyncProtocol, PROTOCOL_VERSION};
use crate::telemetry::baggage;
use crate::{AstraSyncError, DataType};

#[cfg(test)]
use bytes::BytesMut;

/// Events emitted by the mesh for consumption by the sync engine.
#[derive(Debug, Clone)]
pub enum MeshEvent {
    /// A gossip message was received for a data type.
    Gossip {
        data_type: DataType,
        payload: Vec<u8>,
        source: PeerId,
    },
    /// A direct sync request was received from a peer.
    SyncRequest {
        request: SyncProtocol,
        source: PeerId,
    },
    /// A direct sync response was received from a peer.
    SyncResponse {
        response: SyncProtocol,
        source: PeerId,
    },
    /// A new peer was discovered or connected.
    PeerConnected { peer_id: PeerId },
    /// A peer disconnected.
    PeerDisconnected { peer_id: PeerId },
}

/// Handle to the running P2P mesh, used by other subsystems to broadcast,
/// query peers, and subscribe to mesh events.
#[derive(Debug, Clone)]
pub struct P2PMeshHandle {
    pub local_peer_id: PeerId,
    pub cmd_tx: mpsc::Sender<MeshCommand>,
    event_tx: broadcast::Sender<MeshEvent>,
    connected_peers: Arc<Mutex<HashSet<PeerId>>>,
}

impl P2PMeshHandle {
    /// Gossips an encrypted record blob to the mesh topic for this data type.
    pub async fn gossip(
        &self,
        data_type: DataType,
        payload: Vec<u8>,
    ) -> Result<(), AstraSyncError> {
        self.cmd_tx
            .send(MeshCommand::Gossip { data_type, payload })
            .await
            .map_err(|_| AstraSyncError::P2P("mesh command channel closed".to_string()))
    }

    /// Sends a sync request to a specific peer over `/astra-sync/1.0.0`.
    pub async fn request_sync(
        &self,
        peer: PeerId,
        since_lamport: u64,
    ) -> Result<(), AstraSyncError> {
        // Embed current OTel baggage in the handshake nonce for distributed
        // trace propagation across the mesh.
        let baggage_wire =
            crate::telemetry::baggage::encode_baggage(&opentelemetry::Context::current());
        let nonce = if !baggage_wire.is_empty() {
            baggage_wire.into_bytes()
        } else {
            crate::crypto::content_hash(&since_lamport.to_be_bytes())[..16].to_vec()
        };
        let request = SyncProtocol::Handshake(crate::protocol::SyncHandshake {
            protocol_version: PROTOCOL_VERSION,
            kiosk_id: self.local_peer_id.to_string(),
            supported_data_types: vec![
                DataType::Inventory as u8,
                DataType::Cart as u8,
                DataType::Transaction as u8,
            ],
            nonce,
        });
        self.send_request(peer, request).await
    }

    /// Sends an arbitrary sync protocol request to a peer.
    pub async fn send_request(
        &self,
        peer: PeerId,
        request: SyncProtocol,
    ) -> Result<(), AstraSyncError> {
        self.cmd_tx
            .send(MeshCommand::SendRequest { peer, request })
            .await
            .map_err(|_| AstraSyncError::P2P("mesh command channel closed".to_string()))
    }

    /// Returns a receiver for mesh events.
    pub fn subscribe(&self) -> broadcast::Receiver<MeshEvent> {
        self.event_tx.subscribe()
    }

    /// Returns the current set of connected peers.
    pub async fn peers(&self) -> Vec<PeerId> {
        self.connected_peers.lock().await.iter().copied().collect()
    }
}

#[derive(Debug)]
pub enum MeshCommand {
    Gossip {
        data_type: DataType,
        payload: Vec<u8>,
    },
    SendRequest {
        peer: PeerId,
        request: SyncProtocol,
    },
}

/// Combined libp2p network behaviour for the Astra mesh.
#[derive(NetworkBehaviour)]
#[behaviour(out_event = "AstraMeshEvent")]
pub struct AstraMeshBehaviour {
    mdns: mdns::tokio::Behaviour,
    gossipsub: gossipsub::Behaviour,
    sync: request_response::Behaviour<BincodeCodec>,
}

#[derive(Debug)]
pub enum AstraMeshEvent {
    Mdns(mdns::Event),
    Gossipsub(gossipsub::Event),
    Sync(request_response::Event<SyncProtocol, SyncProtocol>),
}

impl From<mdns::Event> for AstraMeshEvent {
    fn from(event: mdns::Event) -> Self {
        AstraMeshEvent::Mdns(event)
    }
}

impl From<gossipsub::Event> for AstraMeshEvent {
    fn from(event: gossipsub::Event) -> Self {
        AstraMeshEvent::Gossipsub(event)
    }
}

impl From<request_response::Event<SyncProtocol, SyncProtocol>> for AstraMeshEvent {
    fn from(event: request_response::Event<SyncProtocol, SyncProtocol>) -> Self {
        AstraMeshEvent::Sync(event)
    }
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
        let bytes = read_length_prefixed(io, 16 * 1024 * 1024).await?;
        SyncProtocol::from_bytes(&bytes)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))
    }

    async fn read_response<T: futures::AsyncRead + Unpin + Send>(
        &mut self,
        _protocol: &Self::Protocol,
        io: &mut T,
    ) -> io::Result<Self::Response> {
        let bytes = read_length_prefixed(io, 16 * 1024 * 1024).await?;
        SyncProtocol::from_bytes(&bytes)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e.to_string()))
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

/// libp2p-backed P2P mesh for the Astra sync daemon.
pub struct P2PMesh {
    #[allow(dead_code)]
    config: std::sync::Arc<Config>,
    local_peer_id: PeerId,
    swarm: Swarm<AstraMeshBehaviour>,
    connected_peers: Arc<Mutex<HashSet<PeerId>>>,
    event_tx: broadcast::Sender<MeshEvent>,
}

impl P2PMesh {
    /// Creates a new P2P mesh for the configured kiosk identity.
    pub async fn new(config: std::sync::Arc<Config>) -> Result<Self, AstraSyncError> {
        let local_key = derive_keypair(&config.daemon.kiosk_id)?;
        let local_peer_id = PeerId::from(local_key.public());
        info!(%local_peer_id, "Initializing P2P mesh");

        let swarm = build_swarm(local_key, &config)?;

        let (event_tx, _event_rx) = broadcast::channel(256);
        let connected_peers = Arc::new(Mutex::new(HashSet::new()));

        Ok(Self {
            config,
            local_peer_id,
            swarm,
            connected_peers,
            event_tx,
        })
    }

    /// Starts the mesh event loop and returns a handle plus a join handle.
    pub async fn start(
        mut self,
        mut shutdown: watch::Receiver<bool>,
    ) -> Result<(P2PMeshHandle, tokio::task::JoinHandle<()>), AstraSyncError> {
        let (cmd_tx, mut cmd_rx) = mpsc::channel::<MeshCommand>(256);
        let handle = P2PMeshHandle {
            local_peer_id: self.local_peer_id,
            cmd_tx,
            event_tx: self.event_tx.clone(),
            connected_peers: self.connected_peers.clone(),
        };
        let local_peer_id = self.local_peer_id;

        let join_handle = tokio::spawn(async move {
            info!(%local_peer_id, "P2P mesh event loop started");
            loop {
                tokio::select! {
                    biased;
                    _ = shutdown.changed() => {
                        info!("P2P mesh received shutdown signal");
                        break;
                    }
                    Some(cmd) = cmd_rx.recv() => {
                        match cmd {
                            MeshCommand::Gossip { data_type, payload } => {
                                let topic = data_type_topic(data_type);
                                trace!(?topic, len = payload.len(), "Publishing gossip message");
                                let _span = span!(Level::TRACE, "mesh.gossip.publish", ?topic).entered();
                                if let Err(e) = self.swarm.behaviour_mut().gossipsub.publish(topic, payload) {
                                    warn!(%e, "Gossip publish failed");
                                }
                            }
                            MeshCommand::SendRequest { peer, request } => {
                                let _span = span!(Level::TRACE, "mesh.send_request", %peer).entered();
                                let request_id = self.swarm.behaviour_mut().sync.send_request(&peer, request);
                                trace!(?request_id, "Sync request queued");
                            }
                        }
                    }
                    event = self.swarm.select_next_some() => {
                        match event {
                            SwarmEvent::NewListenAddr { address, .. } => {
                                info!(%address, "P2P mesh listening");
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Mdns(mdns::Event::Discovered(list))) => {
                                for (peer_id, multiaddr) in list {
                                    debug!(%peer_id, %multiaddr, "Discovered peer via mDNS");
                                }
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Mdns(mdns::Event::Expired(list))) => {
                                for (peer_id, _) in list {
                                    if self.connected_peers.lock().await.remove(&peer_id) {
                                        let _ = self.event_tx.send(MeshEvent::PeerDisconnected { peer_id });
                                    }
                                }
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Gossipsub(gossipsub::Event::Message {
                                propagation_source,
                                message,
                                ..
                            })) => {
                                let _span = span!(Level::TRACE, "mesh.gossip.recv", %propagation_source).entered();
                                trace!(topic = ?message.topic, "Received gossip message");
                                if let Some(data_type) = data_type_from_topic_hash(&message.topic.clone().into_string()
                                ) {
                                    let _ = self.event_tx.send(MeshEvent::Gossip {
                                        data_type,
                                        payload: message.data,
                                        source: propagation_source,
                                    });
                                }
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Sync(request_response::Event::Message {
                                peer,
                                message,
                                ..
                            })) => {
                                let _span = span!(Level::DEBUG, "mesh.sync.message", %peer).entered();
                                let request_event = match &message {
                                    request_response::Message::Request { request, .. } => {
                                        // Extract any baggage context from the handshake nonce
                                        if let SyncProtocol::Handshake(hs) = request {
                                            let ctx = baggage::decode_baggage(
                                                &Context::current(),
                                                &String::from_utf8_lossy(&hs.nonce),
                                            );
                                            let _attach = ctx.attach();
                                        }
                                        MeshEvent::SyncRequest {
                                            request: request.clone(),
                                            source: peer,
                                        }
                                    }
                                    request_response::Message::Response { response, .. } => {
                                        if let SyncProtocol::Handshake(hs) = response {
                                            let ctx = baggage::decode_baggage(
                                                &Context::current(),
                                                &String::from_utf8_lossy(&hs.nonce),
                                            );
                                            let _attach = ctx.attach();
                                        }
                                        MeshEvent::SyncResponse {
                                            response: response.clone(),
                                            source: peer,
                                        }
                                    }
                                };
                                let _ = self.event_tx.send(request_event);
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Sync(request_response::Event::OutboundFailure {
                                peer, error, ..
                            })) => {
                                warn!(%peer, %error, "Outbound sync request failed");
                            }
                            SwarmEvent::Behaviour(AstraMeshEvent::Sync(request_response::Event::InboundFailure {
                                peer, error, ..
                            })) => {
                                warn!(%peer, %error, "Inbound sync request failed");
                            }
                            SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                                let _span = span!(Level::DEBUG, "mesh.connection", %peer_id).entered();
                                debug!("Peer connected");
                                self.connected_peers.lock().await.insert(peer_id);
                                let _ = self.event_tx.send(MeshEvent::PeerConnected { peer_id });
                            }
                            SwarmEvent::ConnectionClosed { peer_id, cause, .. } => {
                                let _span = span!(Level::DEBUG, "mesh.disconnection", %peer_id).entered();
                                debug!(?cause, "Peer disconnected");
                                self.connected_peers.lock().await.remove(&peer_id);
                                let _ = self.event_tx.send(MeshEvent::PeerDisconnected { peer_id });
                            }
                            _ => {}
                        }
                    }
                }
            }
            info!("P2P mesh event loop stopped");
        });

        Ok((handle, join_handle))
    }
}

fn build_swarm(
    local_key: identity::Keypair,
    config: &Config,
) -> Result<Swarm<AstraMeshBehaviour>, AstraSyncError> {
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

    let sync_protocols = vec![(
        StreamProtocol::new(SYNC_PROTOCOL),
        request_response::ProtocolSupport::Full,
    )];
    let sync_cfg =
        request_response::Config::default().with_request_timeout(Duration::from_secs(10));
    let sync = request_response::Behaviour::with_codec(BincodeCodec, sync_protocols, sync_cfg);

    let behaviour = AstraMeshBehaviour {
        mdns,
        gossipsub,
        sync,
    };

    let mut swarm = SwarmBuilder::with_existing_identity(local_key)
        .with_tokio()
        .with_quic()
        .with_behaviour(|_key| Ok(behaviour))
        .map_err(|e| AstraSyncError::P2P(format!("swarm behaviour failed: {e}")))?
        .with_swarm_config(|cfg: libp2p::swarm::Config| {
            cfg.with_idle_connection_timeout(Duration::from_secs(config.p2p.conn_idle_timeout_secs))
        })
        .build();

    let listen_addr: Multiaddr = config
        .p2p
        .listen_addr
        .to_string()
        .parse()
        .map_err(|e| AstraSyncError::P2P(format!("invalid listen address: {e}")))?;
    swarm
        .listen_on(listen_addr)
        .map_err(|e| AstraSyncError::P2P(format!("listen failed: {e}")))?;

    Ok(swarm)
}

fn derive_keypair(kiosk_id: &str) -> Result<identity::Keypair, AstraSyncError> {
    let seed = crate::crypto::content_hash(kiosk_id.as_bytes());
    identity::Keypair::ed25519_from_bytes(seed)
        .map_err(|e| AstraSyncError::Crypto(format!("failed to derive libp2p keypair: {e}")))
}

fn data_type_topic(data_type: DataType) -> gossipsub::IdentTopic {
    gossipsub::IdentTopic::new(format!("astra/sync/{}", data_type.as_str()))
}

fn data_type_from_topic_hash(topic_hash: &str) -> Option<DataType> {
    let suffix = topic_hash.strip_prefix("astra/sync/")?;
    match suffix {
        "inventory" => Some(DataType::Inventory),
        "cart" => Some(DataType::Cart),
        "transaction" => Some(DataType::Transaction),
        "analytics" => Some(DataType::Analytics),
        _ => None,
    }
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
    fn topic_name_maps_to_data_type() {
        let topic = data_type_topic(DataType::Inventory);
        assert_eq!(
            data_type_from_topic_hash(&topic.to_string()),
            Some(DataType::Inventory)
        );
    }

    #[test]
    fn codec_roundtrip() {
        let request = SyncProtocol::Handshake(crate::protocol::SyncHandshake {
            protocol_version: PROTOCOL_VERSION,
            kiosk_id: "k1".into(),
            supported_data_types: vec![0],
            nonce: vec![1, 2, 3],
        });

        // The codec expects an async stream; test via the SyncProtocol encode/decode instead.
        let encoded = request.encode().expect("encode");
        let mut decoded = BytesMut::from(encoded.as_ref());
        let round = SyncProtocol::decode(&mut decoded)
            .expect("frame present")
            .expect("decode ok");
        assert_eq!(request, round);
    }
}
