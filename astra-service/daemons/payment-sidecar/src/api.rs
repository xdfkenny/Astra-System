//! HTTP API for the local Verifone payment sidecar.
//!
//! The sidecar binds to 127.0.0.1 only. It is never reachable from the store
//! LAN or the internet — the kiosk browser talks to it over loopback, and the
//! native Verifone SDK talks to the terminal over its own private connection.

use std::net::SocketAddr;
use std::sync::Arc;

use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use tower_http::cors::{Any, CorsLayer};
use tower_http::limit::RequestBodyLimitLayer;
use tracing::{info, warn};

use crate::verifone::{AuthorizationResult, Terminal};
use crate::PaymentError;

/// Shared application state.
pub struct AppState {
    pub terminal: Arc<dyn Terminal>,
}

/// Start the sidecar HTTP server on the configured address. This function
/// blocks until the server is shut down.
pub async fn serve(addr: SocketAddr, terminal: Arc<dyn Terminal>) -> Result<(), std::io::Error> {
    let state = Arc::new(AppState { terminal });

    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    let app = Router::new()
        .route("/healthz", get(healthz))
        .route("/v1/payments/initiate", post(initiate_payment))
        .route("/v1/payments/:authorization_id/status", get(payment_status))
        .route("/v1/payments/:authorization_id/cancel", post(cancel_payment))
        .layer(cors)
        .layer(RequestBodyLimitLayer::new(64 * 1024)) // 64KB — payment payloads are tiny
        .with_state(state);

    info!(%addr, "Astra payment sidecar listening");
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await
}

async fn healthz() -> StatusCode {
    StatusCode::OK
}

#[derive(Debug, Deserialize)]
pub struct InitiateRequest {
    cart_id: String,
    amount_cents: u64,
    #[serde(default = "default_method")]
    method: String,
    #[serde(default = "default_currency")]
    currency: String,
    #[serde(default)]
    idempotency_key: Option<String>,
}

fn default_method() -> String {
    "credit_debit".to_string()
}

fn default_currency() -> String {
    "USD".to_string()
}

#[derive(Debug, Serialize)]
pub struct InitiateResponse {
    authorization_id: String,
    status: String,
    amount_cents: u64,
    method: String,
    approval_code: Option<String>,
    decline_reason: Option<String>,
    card_brand: Option<String>,
    card_last_four: Option<String>,
    receipt_text: Option<String>,
    opaque_token: String,
    idempotency_key: Option<String>,
}

async fn initiate_payment(
    State(state): State<Arc<AppState>>,
    Json(req): Json<InitiateRequest>,
) -> Result<Json<InitiateResponse>, PaymentError> {
    info!(cart_id = %req.cart_id, amount_cents = req.amount_cents, "initiating payment");

    let result = state
        .terminal
        .initiate(req.amount_cents, &req.currency, &req.method)
        .await?;

    Ok(Json(map_result(result, req.idempotency_key)))
}

async fn payment_status(
    State(state): State<Arc<AppState>>,
    Path(authorization_id): Path<String>,
) -> Result<Json<InitiateResponse>, PaymentError> {
    let result = state.terminal.status(&authorization_id).await?;
    Ok(Json(map_result(result, None)))
}

async fn cancel_payment(
    State(state): State<Arc<AppState>>,
    Path(authorization_id): Path<String>,
) -> Result<StatusCode, PaymentError> {
    state.terminal.cancel(&authorization_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

fn map_result(result: AuthorizationResult, idempotency_key: Option<String>) -> InitiateResponse {
    InitiateResponse {
        authorization_id: result.authorization_id,
        status: format!("{:?}", result.status).to_lowercase(),
        amount_cents: result.amount_cents,
        method: result.method,
        approval_code: result.approval_code,
        decline_reason: result.decline_reason,
        card_brand: result.card_brand,
        card_last_four: result.card_last_four,
        receipt_text: result.receipt_text,
        opaque_token: result.opaque_token,
        idempotency_key,
    }
}

impl axum::response::IntoResponse for PaymentError {
    fn into_response(self) -> axum::response::Response {
        let (status, message) = match self {
            PaymentError::InvalidRequest(msg) => (StatusCode::BAD_REQUEST, msg),
            PaymentError::Terminal(msg) => {
                warn!(error = %msg, "terminal error");
                (StatusCode::BAD_GATEWAY, msg)
            }
            PaymentError::Internal(msg) => {
                warn!(error = %msg, "internal error");
                (StatusCode::INTERNAL_SERVER_ERROR, msg)
            }
        };
        let body = Json(serde_json::json!({ "error": message }));
        (status, body).into_response()
    }
}
