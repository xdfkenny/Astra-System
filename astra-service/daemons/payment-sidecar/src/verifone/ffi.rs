//! Verifone C/C++ SDK FFI declarations.
//!
//! The actual Verifone SDK is a proprietary native library distributed under
//! NDA. This module declares the minimal surface area Astra-Service needs:
//! initialization, transaction initiation, status polling, and cancellation.
//! In production these symbols are linked against the vendor library. In
//! development/simulator builds the `sim` module provides a safe stand-in that
//! returns deterministic, PCI-safe responses.
//!
//! # Safety invariant
//! All `extern "C"` functions are `unsafe` by definition. The safe wrappers
//! in `verifone::adapter` enforce the invariants (null checks, length checks,
//! exclusive access) so the rest of the crate never uses raw pointers.

use std::ffi::{c_char, c_int};
use std::marker::PhantomData;

/// Opaque handle to a Verifone terminal session. Mirrored from the SDK's
/// `VFIHandle` type. It is Send but not Sync because the SDK is not
/// thread-safe; the adapter serializes access with a mutex.
#[repr(C)]
pub struct VfiHandle {
    _private: [u8; 0],
}

/// Result codes returned by the Verifone SDK.
#[repr(C)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(non_camel_case_types)]
pub enum VfiResult {
    VFI_OK = 0,
    VFI_ERR_INVALID_PARAM = -1,
    VFI_ERR_NOT_CONNECTED = -2,
    VFI_ERR_TRANSACTION_FAILED = -3,
    VFI_ERR_CANCELLED = -4,
    VFI_ERR_TIMEOUT = -5,
    VFI_ERR_UNKNOWN = -99,
}

/// Transaction request descriptor passed to `vfi_initiate_transaction`.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VfiTransactionRequest {
    pub amount_cents: c_int,
    pub currency_numeric: c_int, // 840 = USD per ISO 4217
    pub transaction_type: c_int, // 0 = sale, 1 = refund
}

/// Transaction response descriptor returned by `vfi_get_transaction_result`.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VfiTransactionResponse {
    pub result_code: VfiResult,
    pub authorized: c_int, // 0 = false, 1 = true
    pub amount_cents: c_int,
    pub auth_code: [c_char; 16],
    pub reference_number: [c_char; 32],
    pub card_last_four: [c_char; 5],
    pub card_brand: [c_char; 16],
}

extern "C" {
    /// Opens a connection to the Verifone terminal at the given URI.
    /// `uri` is a null-terminated UTF-8 string, e.g. "verifone://10.0.1.21".
    /// On success, `*out_handle` is set and the caller must later call
    /// `vfi_disconnect`.
    pub fn vfi_connect(uri: *const c_char, out_handle: *mut *mut VfiHandle) -> VfiResult;

    /// Disconnects and frees the terminal session.
    pub fn vfi_disconnect(handle: *mut VfiHandle) -> VfiResult;

    /// Initiates a transaction on the terminal. This is non-blocking; the
    /// terminal prompts the customer and the caller polls with
    /// `vfi_get_transaction_result`.
    pub fn vfi_initiate_transaction(
        handle: *mut VfiHandle,
        request: *const VfiTransactionRequest,
        out_transaction_id: *mut c_char,
        transaction_id_len: usize,
    ) -> VfiResult;

    /// Polls for the result of a previously initiated transaction. If the
    /// transaction is still in progress, returns `VFI_OK` with
    /// `response.authorized == 0` and empty strings.
    pub fn vfi_get_transaction_result(
        handle: *mut VfiHandle,
        transaction_id: *const c_char,
        response: *mut VfiTransactionResponse,
    ) -> VfiResult;

    /// Cancels an in-progress transaction.
    pub fn vfi_cancel_transaction(handle: *mut VfiHandle, transaction_id: *const c_char) -> VfiResult;
}

/// Marker type indicating the FFI layer is present. Production binaries link
/// the vendor SDK; simulator binaries use `SimAdapter` instead.
pub struct FfiAdapter {
    _marker: PhantomData<()>,
}

impl FfiAdapter {
    pub fn new() -> Self {
        Self { _marker: PhantomData }
    }
}

impl Default for FfiAdapter {
    fn default() -> Self {
        Self::new()
    }
}

/// Safe wrapper around a connected terminal handle.
pub struct TerminalHandle {
    pub(crate) raw: *mut VfiHandle,
}

unsafe impl Send for TerminalHandle {}

impl Drop for TerminalHandle {
    fn drop(&mut self) {
        if !self.raw.is_null() {
            unsafe {
                let _ = vfi_disconnect(self.raw);
            }
        }
    }
}

/// A PCI-safe opaque token returned by the Verifone terminal. It contains no
/// card data — only a signed reference that the cloud payment orchestrator can
/// redeem for settlement.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct VerifoneOpaqueToken {
    pub transaction_id: String,
    pub reference_number: String,
    pub terminal_uri: String,
    pub authorized_at: String,
}

impl VerifoneOpaqueToken {
    /// Serializes the token to a base64-encoded JSON string for transport.
    pub fn encode(&self) -> String {
        use base64::Engine;
        base64::engine::general_purpose::STANDARD.encode(serde_json::to_vec(self).unwrap_or_default())
    }

    /// Deserializes from a base64-encoded JSON string.
    pub fn decode(s: &str) -> Result<Self, serde_json::Error> {
        use base64::Engine;
        let bytes = base64::engine::general_purpose::STANDARD
            .decode(s)
            .unwrap_or_default();
        serde_json::from_slice(&bytes)
    }
}
