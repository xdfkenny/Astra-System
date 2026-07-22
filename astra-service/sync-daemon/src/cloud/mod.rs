#![deny(unsafe_code)]

//! Cloud synchronization via NATS JetStream.
//!
//! The cloud sync module is only active when:
//! 1. This kiosk is the Raft-elected leader.
//! 2. An internet connection is available (detected via P2P mesh online state).
//!
//! When active, it:
//! - Reads dirty records from the local SQLite database.
//! - Publishes them to a NATS JetStream subject.
//! - Consumes cloud updates and writes them to the local database.
//! - Flushes offline payment tokens to the cloud backend for reconciliation.
//!
//! When offline, the module idles and records remain queued locally.

use std::sync::Arc;
use std::time::Duration;

use async_nats::jetstream::context::Context;
use async_nats::jetstream::stream::Config as StreamConfig;
use base64::Engine;
use tokio::sync::{watch, RwLock};
use tokio::time::interval;
use tracing::{debug, error, info, trace, warn};

use crate::config::Config;
use crate::differential_privacy::{privatize_analytics_payload, DEFAULT_EPSILON};
use crate::storage::sqlite::SyncDatabase;
use crate::{AstraSyncError, DaemonState, DataType, SyncRecord};

/// Cloud sync handle, returned after starting the cloud sync loop.
#[derive(Debug, Clone)]
pub struct CloudSyncHandle {
    pub cmd_tx: tokio::sync::mpsc::Sender<CloudCommand>,
}

#[derive(Debug, Clone)]
pub enum CloudCommand {
    ForceFlush,
    Shutdown,
}

pub struct CloudSync {
    config: Arc<Config>,
    state: Arc<RwLock<DaemonState>>,
    db: Arc<SyncDatabase>,
    jetstream: Option<Arc<Context>>,
}

impl CloudSync {
    pub async fn new(
        config: Arc<Config>,
        state: Arc<RwLock<DaemonState>>,
        db: Arc<SyncDatabase>,
    ) -> Result<Self, AstraSyncError> {
        // Attempt to connect to NATS on startup. If it fails, we operate in
        // offline mode and retry periodically.
        let jetstream = match connect_nats(&config).await {
            Ok(js) => {
                info!("NATS JetStream connected on startup");
                Some(Arc::new(js))
            }
            Err(e) => {
                warn!(%e, "NATS JetStream unavailable on startup — operating offline");
                None
            }
        };

        Ok(Self {
            config,
            state,
            db,
            jetstream,
        })
    }

    /// Starts the cloud sync background loop.
    pub async fn start(
        self,
        mut shutdown: watch::Receiver<bool>,
    ) -> Result<tokio::task::JoinHandle<()>, AstraSyncError> {
        let config = self.config.clone();
        let state = self.state.clone();
        let db = self.db.clone();
        let mut jetstream = self.jetstream.clone();

        let (_cmd_tx, mut cmd_rx) = tokio::sync::mpsc::channel::<CloudCommand>(64);

        let handle = tokio::spawn(async move {
            let mut flush_timer =
                interval(Duration::from_secs(config.cloud.flush_interval_seconds));
            flush_timer.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);

            let mut reconnect_timer = interval(Duration::from_secs(30));
            reconnect_timer.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

            info!("Cloud sync event loop started");

            loop {
                tokio::select! {
                    _ = flush_timer.tick() => {
                        let is_leader = { state.read().await.is_leader };
                        let online = { state.read().await.online };

                        if !is_leader {
                            trace!("Not leader — skipping cloud flush");
                            continue;
                        }
                        if !online {
                            trace!("Offline — skipping cloud flush");
                            continue;
                        }

                        // Ensure we have a JetStream connection.
                        if jetstream.is_none() {
                            trace!("JetStream not connected — skipping cloud flush");
                            continue;
                        }

                        if let Some(js) = jetstream.as_ref() {
                            if let Err(e) = flush_to_cloud(&db, js, &config).await {
                                error!(%e, "Cloud flush failed");
                            }
                        }
                    }
                    _ = reconnect_timer.tick() => {
                        if jetstream.is_none() {
                            match connect_nats(&config).await {
                                Ok(js) => {
                                    info!("NATS JetStream reconnected");
                                    jetstream = Some(Arc::new(js));
                                }
                                Err(e) => {
                                    trace!(%e, "NATS reconnection attempt failed");
                                }
                            }
                        }
                    }
                    Some(cmd) = cmd_rx.recv() => {
                        match cmd {
                            CloudCommand::ForceFlush => {
                                info!("Forced cloud flush requested");
                                if let Some(ref js) = jetstream {
                                    if let Err(e) = flush_to_cloud(&db, js, &config).await {
                                        error!(%e, "Forced cloud flush failed");
                                    }
                                } else {
                                    warn!("Cannot force flush — JetStream not connected");
                                }
                            }
                            CloudCommand::Shutdown => {
                                info!("Cloud sync shutting down");
                                break;
                            }
                        }
                    }
                    _ = shutdown.changed() => {
                        info!("Cloud sync received shutdown signal");
                        break;
                    }
                }
            }

            info!("Cloud sync event loop stopped");
        });

        Ok(handle)
    }
}

/// Connects to NATS and returns a JetStream context.
async fn connect_nats(config: &Config) -> Result<Context, AstraSyncError> {
    let opts = async_nats::ConnectOptions::new()
        .require_tls(true)
        .connection_timeout(Duration::from_secs(config.cloud.connect_timeout_secs));

    let client = opts
        .connect(&config.cloud.nats_url)
        .await
        .map_err(|e| AstraSyncError::Cloud(format!("NATS connection failed: {e}")))?;

    let js = async_nats::jetstream::new(client);

    // Ensure the stream exists (idempotent).
    let stream_name = format!("ASTRA_{}", config.p2p.network_name.to_uppercase());
    let _ = js
        .create_stream(StreamConfig {
            name: stream_name.clone(),
            subjects: vec![format!("astra.{}.*", config.p2p.network_name)],
            max_messages: 1_000_000,
            max_bytes: 10 * 1024 * 1024 * 1024, // 10 GiB
            ..Default::default()
        })
        .await
        .map_err(|e| AstraSyncError::Cloud(format!("stream creation failed: {e}")))?;

    debug!(stream = %stream_name, "JetStream context ready");
    Ok(js)
}

/// Flushes dirty records and offline payments to the cloud.
async fn flush_to_cloud(
    db: &Arc<SyncDatabase>,
    js: &Context,
    config: &Config,
) -> Result<(), AstraSyncError> {
    let mut total_flushed = 0usize;

    // Flush each data type in priority order.
    for data_type in [
        DataType::Inventory,
        DataType::Cart,
        DataType::Transaction,
        DataType::Analytics,
    ] {
        let records: Vec<SyncRecord<serde_json::Value>> = db.load_dirty(data_type, 1000).await?;
        if records.is_empty() {
            continue;
        }

        let subject = format!("astra.{}.{}", config.p2p.network_name, data_type.as_str());
        let mut batch = Vec::new();
        let mut ids = Vec::new();

        for rec in records {
            let id = rec.id.clone();
            let payload = if data_type == DataType::Analytics {
                privatize_analytics_payload(rec.payload, DEFAULT_EPSILON)?
            } else {
                rec.payload
            };
            let record = SyncRecord {
                data_type: rec.data_type,
                id,
                origin: rec.origin,
                lamport_ts: rec.lamport_ts,
                wallclock_ts: rec.wallclock_ts,
                payload,
                hmac: rec.hmac,
            };
            let json = serde_json::to_vec(&record)
                .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            if json.len() > config.cloud.max_msg_size_bytes {
                warn!(record_id = %record.id, "Record exceeds max message size — skipping");
                continue;
            }
            batch.push(json);
            ids.push(record.id);

            // Flush when batch reaches 100 records or 512 KiB.
            if batch.len() >= 100 || batch.iter().map(|v| v.len()).sum::<usize>() > 512 * 1024 {
                publish_batch(js, &subject, &batch).await?;
                total_flushed += batch.len();
                batch.clear();
            }
        }

        if !batch.is_empty() {
            publish_batch(js, &subject, &batch).await?;
            total_flushed += batch.len();
        }

        // Mark records as flushed.
        db.flush(data_type).await?;
        debug!(data_type = ?data_type, flushed = ids.len(), "Flushed records to cloud");
    }

    // Flush offline payment tokens.
    let offline_tokens = db.load_offline_queue(1000).await?;
    if !offline_tokens.is_empty() {
        let subject = format!("astra.{}.offline_payments", config.p2p.network_name);
        let mut batch = Vec::new();
        let mut ids = Vec::new();
        for (token_id, payload_json, hmac) in offline_tokens {
            let msg = serde_json::json!({
                "token_id": token_id,
                "payload": payload_json,
                "hmac": base64::engine::general_purpose::STANDARD.encode(&hmac),
            });
            let bytes = serde_json::to_vec(&msg)
                .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            batch.push(bytes);
            ids.push(token_id);
            if batch.len() >= 100 {
                publish_batch(js, &subject, &batch).await?;
                batch.clear();
            }
        }
        if !batch.is_empty() {
            publish_batch(js, &subject, &batch).await?;
        }
        db.mark_offline_flushed(&ids).await?;
        debug!(count = ids.len(), "Flushed offline payment tokens to cloud");
    }

    if total_flushed > 0 {
        info!(total_flushed, "Cloud flush completed");
    }
    Ok(())
}

/// Publishes a batch of JSON bytes to a NATS JetStream subject.
async fn publish_batch(
    js: &Context,
    subject: &str,
    batch: &[Vec<u8>],
) -> Result<(), AstraSyncError> {
    for payload in batch {
        js.publish(subject.to_string(), payload.clone().into())
            .await
            .map_err(|e| AstraSyncError::Cloud(format!("publish failed: {e}")))?
            .await
            .map_err(|e| AstraSyncError::Cloud(format!("publish ack failed: {e}")))?;
    }
    Ok(())
}
