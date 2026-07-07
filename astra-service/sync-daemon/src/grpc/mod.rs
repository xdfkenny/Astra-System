#![deny(unsafe_code)]

//! gRPC server for local IPC between the sync daemon and other Astra services.
//!
//! The gRPC server listens on a Unix domain socket or TCP loopback address
//! and exposes the `AstraSync` service defined in `proto/sync.proto`.
//! Other services (e.g., the POS frontend, inventory manager, analytics
//! exporter) can query sync status, request immediate syncs, and submit
//! transactions via this interface.

use std::net::SocketAddr;
use std::sync::Arc;
use std::time::SystemTime;

use tokio::sync::{watch, RwLock};
use tonic::{transport::Server, Request, Response, Status};
use tracing::{debug, error, info, trace, warn};

use crate::config::Config;
use crate::storage::sqlite::SyncDatabase;
use crate::sync::engine::SyncEngineHandle;
use crate::{DaemonState, AstraSyncError, DataType, VERSION};

// Include the generated protobuf code.
pub mod proto {
    tonic::include_proto!("astra.sync");
}

use proto::astra_sync_server::{AstraSync, AstraSyncServer};
use proto::*;

/// gRPC server implementation.
pub struct GrpcServer {
    config: Arc<Config>,
    state: Arc<RwLock<DaemonState>>,
    db: Arc<SyncDatabase>,
    sync_handle: SyncEngineHandle,
}

impl GrpcServer {
    pub fn new(
        config: Arc<Config>,
        state: Arc<RwLock<DaemonState>>,
        db: Arc<SyncDatabase>,
        sync_handle: SyncEngineHandle,
    ) -> Self {
        Self { config, state, db, sync_handle }
    }

    /// Starts the gRPC server and returns a join handle.
    pub async fn start(self, mut shutdown: watch::Receiver<bool>) -> Result<tokio::task::JoinHandle<()>, AstraSyncError> {
        let addr = self.config.grpc.listen_addr;
        let service = AstraSyncService {
            state: self.state.clone(),
            db: self.db.clone(),
            sync_handle: self.sync_handle.clone(),
        };

        info!(addr = %addr, "gRPC server starting");

        let mut shutdown_for_serve = shutdown.clone();
        let server = Server::builder()
            .add_service(AstraSyncServer::new(service))
            .serve_with_shutdown(addr, async move {
                let _ = shutdown_for_serve.changed().await;
                info!("gRPC server received shutdown signal");
            });

        let handle = tokio::spawn(async move {
            if let Err(e) = server.await {
                error!(%e, "gRPC server error");
            }
            info!("gRPC server stopped");
        });

        Ok(handle)
    }
}

/// Inner service that implements the generated `AstraSync` trait.
#[derive(Debug, Clone)]
struct AstraSyncService {
    state: Arc<RwLock<DaemonState>>,
    db: Arc<SyncDatabase>,
    sync_handle: SyncEngineHandle,
}

#[tonic::async_trait]
impl AstraSync for AstraSyncService {
    async fn health_check(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<HealthResponse>, Status> {
        let state = self.state.read().await;
        let uptime = SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64
            - (state.started_at / 1000) as i64;
        let uptime_proto = prost_types::Timestamp {
            seconds: uptime,
            nanos: 0,
        };
        Ok(Response::new(HealthResponse {
            healthy: true,
            version: VERSION.to_string(),
            peer_id: state.kiosk_id.to_string(),
            uptime: Some(uptime_proto),
        }))
    }

    async fn sync_now(
        &self,
        request: Request<SyncRequest>,
    ) -> Result<Response<SyncResponse>, Status> {
        let req = request.into_inner();
        let data_type = match req.data_type() {
            proto::DataType::Inventory => DataType::Inventory,
            proto::DataType::Cart => DataType::Cart,
            proto::DataType::Transaction => DataType::Transaction,
            proto::DataType::Analytics => DataType::Analytics,
        };

        match self.sync_handle.request_sync(data_type).await {
            Ok(_) => Ok(Response::new(SyncResponse {
                success: true,
                message: format!("Sync request for {:?} accepted", data_type),
                synced_records: 0, // Actual count is tracked asynchronously.
            })),
            Err(e) => Ok(Response::new(SyncResponse {
                success: false,
                message: format!("Sync request failed: {}", e),
                synced_records: 0,
            })),
        }
    }

    async fn get_sync_status(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<SyncStatus>, Status> {
        let state = self.state.read().await;
        let (inv_ts, _) = self.db.get_sync_state(DataType::Inventory).await
            .unwrap_or((0, 0));
        let (cart_ts, _) = self.db.get_sync_state(DataType::Cart).await
            .unwrap_or((0, 0));
        let (tx_ts, _) = self.db.get_sync_state(DataType::Transaction).await
            .unwrap_or((0, 0));
        let (ana_ts, _) = self.db.get_sync_state(DataType::Analytics).await
            .unwrap_or((0, 0));

        let pending_inventory = self.db.load_dirty::<serde_json::Value>(DataType::Inventory, 1000).await
            .unwrap_or_default().len() as u64;
        let pending_carts = self.db.load_dirty::<serde_json::Value>(DataType::Cart, 1000).await
            .unwrap_or_default().len() as u64;
        let pending_tx = self.db.load_dirty::<serde_json::Value>(DataType::Transaction, 1000).await
            .unwrap_or_default().len() as u64;
        let pending_ana = self.db.load_dirty::<serde_json::Value>(DataType::Analytics, 1000).await
            .unwrap_or_default().len() as u64;

        let offline = self.db.load_offline_queue(1).await.unwrap_or_default();
        let has_internet = state.online;

        Ok(Response::new(SyncStatus {
            is_online: state.online,
            is_leader: state.is_leader,
            last_sync_timestamp: inv_ts.max(cart_ts).max(tx_ts).max(ana_ts),
            pending_local_records: pending_inventory + pending_carts + pending_tx + pending_ana,
            pending_cloud_records: if state.is_leader && has_internet { 0 } else { pending_tx },
            peers: vec![], // In production, populate from P2P mesh peer list.
            inventory_status: Some(DataSyncStatus {
                data_type: proto::DataType::Inventory as i32,
                last_sync_timestamp: inv_ts,
                pending_count: pending_inventory,
                priority: proto::SyncPriority::Immediate as i32,
            }),
            cart_status: Some(DataSyncStatus {
                data_type: proto::DataType::Cart as i32,
                last_sync_timestamp: cart_ts,
                pending_count: pending_carts,
                priority: proto::SyncPriority::Batched as i32,
            }),
            transaction_status: Some(DataSyncStatus {
                data_type: proto::DataType::Transaction as i32,
                last_sync_timestamp: tx_ts,
                pending_count: pending_tx,
                priority: proto::SyncPriority::Batched as i32,
            }),
            analytics_status: Some(DataSyncStatus {
                data_type: proto::DataType::Analytics as i32,
                last_sync_timestamp: ana_ts,
                pending_count: pending_ana,
                priority: proto::SyncPriority::Delayed as i32,
            }),
        }))
    }

    async fn submit_transaction(
        &self,
        request: Request<TransactionPayload>,
    ) -> Result<Response<TransactionResponse>, Status> {
        let tx = request.into_inner();
        debug!(transaction_id = %tx.transaction_id, "Transaction submitted via gRPC");
        // In production, validate the HMAC, deserialize the payload, and
        // enqueue it in the sync engine's transaction queue.
        Ok(Response::new(TransactionResponse {
            accepted: true,
            transaction_id: tx.transaction_id,
            status: "queued".to_string(),
        }))
    }

    async fn get_mesh_info(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<MeshInfo>, Status> {
        let state = self.state.read().await;
        Ok(Response::new(MeshInfo {
            local_peer_id: state.kiosk_id.to_string(),
            connected_peers: 0, // Populate from P2P mesh in production.
            known_peers: 0,
            network_name: "astra-kiosk-mesh".to_string(),
            protocols: vec![
                "/astra/sync/1.0.0".to_string(),
                "/astra/identify/1.0.0".to_string(),
            ],
        }))
    }

    async fn get_leader_status(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<LeaderStatus>, Status> {
        let state = self.state.read().await;
        Ok(Response::new(LeaderStatus {
            is_leader: state.is_leader,
            leader_id: if state.is_leader {
                state.kiosk_id.to_string()
            } else {
                "".to_string()
            },
            term: state.raft_term,
            last_heartbeat: 0, // Populate from Raft state in production.
        }))
    }

    async fn get_offline_queue(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<OfflineQueueStatus>, Status> {
        let state = self.state.read().await;
        let queue = self.db.load_offline_queue(10000).await
            .unwrap_or_default();
        let oldest = queue.first().map(|(_, _, _)| 0u64).unwrap_or(0);
        Ok(Response::new(OfflineQueueStatus {
            queued_count: queue.len() as u32,
            total_value: 0, // Compute from payloads in production.
            oldest_timestamp: oldest,
            has_internet: state.online,
        }))
    }

    async fn force_cloud_flush(
        &self,
        _request: Request<proto::Empty>,
    ) -> Result<Response<FlushResponse>, Status> {
        let state = self.state.read().await;
        if !state.is_leader {
            return Ok(Response::new(FlushResponse {
                success: false,
                flushed_records: 0,
                error: "Only the leader can force a cloud flush".to_string(),
            }));
        }
        // In production, send a CloudCommand::ForceFlush to the cloud sync loop.
        Ok(Response::new(FlushResponse {
            success: true,
            flushed_records: 0,
            error: "".to_string(),
        }))
    }
}
