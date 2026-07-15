//! Verifone terminal simulator for development and CI.
//!
//! This provides the same `Terminal` trait as the real FFI adapter but without
//! linking the proprietary SDK. It is enabled by default; production builds
//! can swap to the FFI-backed adapter at compile time or runtime via config.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use chrono::Utc;
use ring::rand::SecureRandom;
use uuid::Uuid;

use crate::verifone::ffi::VerifoneOpaqueToken;
use crate::PaymentError;

/// Transaction state kept by the simulator.
#[derive(Debug, Clone)]
pub enum SimState {
    Pending,
    Authorized {
        auth_code: String,
        reference: String,
        card_last_four: String,
        card_brand: String,
    },
    Declined {
        reason: String,
    },
    Cancelled,
}

/// In-memory simulator of a Verifone terminal.
pub struct SimTerminal {
    uri: String,
    transactions: Arc<Mutex<HashMap<String, SimState>>>,
    rng: ring::rand::SystemRandom,
}

impl SimTerminal {
    pub fn new(uri: String) -> Self {
        Self {
            uri,
            transactions: Arc::new(Mutex::new(HashMap::new())),
            rng: ring::rand::SystemRandom::new(),
        }
    }

    /// Initiates a simulated transaction. The simulator immediately succeeds
    /// for amounts ending in an even dollar (e.g. $1.00) and declines for
    /// amounts ending in $0.99, making deterministic test cases possible.
    pub async fn initiate(
        &self,
        amount_cents: u64,
        _currency: &str,
    ) -> Result<VerifoneOpaqueToken, PaymentError> {
        let transaction_id = Uuid::now_v7().to_string();
        let reference = self.random_hex(16);

        let state = if amount_cents % 100 == 99 {
            SimState::Declined {
                reason: "simulated_decline".to_string(),
            }
        } else {
            SimState::Authorized {
                auth_code: self.random_hex(6),
                reference: reference.clone(),
                card_last_four: self.random_digits(4),
                card_brand: "visa".to_string(),
            }
        };

        self.transactions
            .lock()
            .unwrap()
            .insert(transaction_id.clone(), state);

        Ok(VerifoneOpaqueToken {
            transaction_id,
            reference_number: reference,
            terminal_uri: self.uri.clone(),
            authorized_at: Utc::now().to_rfc3339(),
        })
    }

    /// Returns the current state of a simulated transaction.
    pub async fn status(&self, transaction_id: &str) -> Result<SimState, PaymentError> {
        let txns = self.transactions.lock().unwrap();
        txns.get(transaction_id)
            .cloned()
            .ok_or_else(|| PaymentError::Terminal("transaction not found".to_string()))
    }

    /// Cancels a pending transaction.
    pub async fn cancel(&self, transaction_id: &str) -> Result<(), PaymentError> {
        let mut txns = self.transactions.lock().unwrap();
        match txns.get(transaction_id) {
            Some(SimState::Pending) | Some(SimState::Authorized { .. }) => {
                txns.insert(transaction_id.to_string(), SimState::Cancelled);
                Ok(())
            }
            Some(SimState::Cancelled) => Ok(()),
            Some(SimState::Declined { .. }) => Err(PaymentError::Terminal(
                "cannot cancel a declined transaction".to_string(),
            )),
            None => Err(PaymentError::Terminal("transaction not found".to_string())),
        }
    }

    fn random_hex(&self, len: usize) -> String {
        let mut buf = vec![0u8; len / 2 + len % 2];
        self.rng.fill(&mut buf).expect("rng failure");
        hex::encode(&buf)[..len].to_string()
    }

    fn random_digits(&self, count: usize) -> String {
        let mut buf = vec![0u8; count];
        self.rng.fill(&mut buf).expect("rng failure");
        buf.iter().map(|b| (b % 10).to_string()).collect::<String>()[..count].to_string()
    }
}

impl SimState {
    pub fn is_authorized(&self) -> bool {
        matches!(self, SimState::Authorized { .. })
    }

    pub fn auth_code(&self) -> Option<&str> {
        match self {
            SimState::Authorized { auth_code, .. } => Some(auth_code.as_str()),
            _ => None,
        }
    }

    pub fn reference(&self) -> Option<&str> {
        match self {
            SimState::Authorized { reference, .. } => Some(reference.as_str()),
            _ => None,
        }
    }

    pub fn card_last_four(&self) -> Option<&str> {
        match self {
            SimState::Authorized { card_last_four, .. } => Some(card_last_four.as_str()),
            _ => None,
        }
    }

    pub fn card_brand(&self) -> Option<&str> {
        match self {
            SimState::Authorized { card_brand, .. } => Some(card_brand.as_str()),
            _ => None,
        }
    }

    pub fn decline_reason(&self) -> Option<&str> {
        match self {
            SimState::Declined { reason } => Some(reason.as_str()),
            _ => None,
        }
    }
}
