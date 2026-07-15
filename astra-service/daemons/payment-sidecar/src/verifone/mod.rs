//! Safe adapter over the Verifone terminal.
//!
//! `Terminal` is the trait used by the HTTP API. In production it is backed by
//! the FFI SDK; in development it is backed by the in-memory simulator. The
//! adapter ensures all card data stays inside the native layer and only
//! PCI-safe opaque tokens reach the browser.

use std::sync::Arc;

use tokio::sync::Mutex;

use crate::PaymentError;

pub mod ffi;
pub mod sim;

/// A terminal-agnostic payment authorization result. No PAN/track/CVV.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AuthorizationResult {
    pub authorization_id: String,
    pub status: AuthorizationStatus,
    pub method: String,
    pub amount_cents: u64,
    pub approval_code: Option<String>,
    pub decline_reason: Option<String>,
    pub card_brand: Option<String>,
    pub card_last_four: Option<String>,
    pub receipt_text: Option<String>,
    pub opaque_token: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum AuthorizationStatus {
    Authorized,
    Declined,
    Cancelled,
    Pending,
    NetworkError,
}

/// Trait implemented by both the real FFI terminal and the simulator.
#[async_trait::async_trait]
pub trait Terminal: Send + Sync {
    async fn initiate(
        &self,
        amount_cents: u64,
        currency: &str,
        method: &str,
    ) -> Result<AuthorizationResult, PaymentError>;

    async fn status(&self, authorization_id: &str) -> Result<AuthorizationResult, PaymentError>;

    async fn cancel(&self, authorization_id: &str) -> Result<(), PaymentError>;
}

/// Simulator-backed terminal. This is the default implementation for the
/// scaffold because the proprietary Verifone SDK is not redistributable.
pub struct SimulatedTerminal {
    inner: Arc<Mutex<sim::SimTerminal>>,
}

impl SimulatedTerminal {
    pub fn new(uri: String) -> Self {
        Self {
            inner: Arc::new(Mutex::new(sim::SimTerminal::new(uri))),
        }
    }
}

#[async_trait::async_trait]
impl Terminal for SimulatedTerminal {
    async fn initiate(
        &self,
        amount_cents: u64,
        currency: &str,
        method: &str,
    ) -> Result<AuthorizationResult, PaymentError> {
        let token = self
            .inner
            .lock()
            .await
            .initiate(amount_cents, currency)
            .await?;
        let state = self
            .inner
            .lock()
            .await
            .status(&token.transaction_id)
            .await?;
        Ok(build_result(
            &token.transaction_id,
            method,
            amount_cents,
            &token,
            &state,
        ))
    }

    async fn status(&self, authorization_id: &str) -> Result<AuthorizationResult, PaymentError> {
        let state = self.inner.lock().await.status(authorization_id).await?;
        // amount/method/opaque_token are not available from state alone in the
        // simulator; we reconstruct a minimal result. In the FFI adapter these
        // would be cached alongside the handle.
        let token = ffi::VerifoneOpaqueToken {
            transaction_id: authorization_id.to_string(),
            reference_number: state.reference().unwrap_or_default().to_string(),
            terminal_uri: "sim://localhost".to_string(),
            authorized_at: chrono::Utc::now().to_rfc3339(),
        };
        Ok(build_result(
            authorization_id,
            "credit_debit",
            0,
            &token,
            &state,
        ))
    }

    async fn cancel(&self, authorization_id: &str) -> Result<(), PaymentError> {
        self.inner.lock().await.cancel(authorization_id).await
    }
}

fn build_result(
    authorization_id: &str,
    method: &str,
    amount_cents: u64,
    token: &ffi::VerifoneOpaqueToken,
    state: &sim::SimState,
) -> AuthorizationResult {
    let (status, approval_code, decline_reason) = match state {
        sim::SimState::Authorized { auth_code, .. } => (
            AuthorizationStatus::Authorized,
            Some(auth_code.clone()),
            None,
        ),
        sim::SimState::Declined { reason } => {
            (AuthorizationStatus::Declined, None, Some(reason.clone()))
        }
        sim::SimState::Cancelled => (AuthorizationStatus::Cancelled, None, None),
        sim::SimState::Pending => (AuthorizationStatus::Pending, None, None),
    };

    AuthorizationResult {
        authorization_id: authorization_id.to_string(),
        status,
        method: method.to_string(),
        amount_cents,
        approval_code,
        decline_reason,
        card_brand: state.card_brand().map(|s| s.to_string()),
        card_last_four: state.card_last_four().map(|s| s.to_string()),
        receipt_text: Some(format!("Astra receipt for auth {}", authorization_id)),
        opaque_token: token.encode(),
    }
}
