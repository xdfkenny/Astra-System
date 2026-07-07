#![warn(missing_docs)]
//! Safe Rust bindings to the Verifone Point of Sale SDK.
//!
//! The underlying C SDK is exposed through `bindgen`-generated raw FFI
//! declarations. This crate wraps those declarations in idiomatic Rust
//! functions that convert integer error codes into a typed [`VerifoneError`].

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};

use thiserror::Error;

#[allow(
    non_upper_case_globals,
    non_camel_case_types,
    non_snake_case,
    dead_code
)]
mod ffi {
    include!(concat!(env!("OUT_DIR"), "/bindings.rs"));
}

/// Raw status code returned by the Verifone SDK.
pub type VerifoneStatus = c_int;

/// Error type for Verifone SDK operations.
#[derive(Debug, Error, Clone, Copy, PartialEq, Eq)]
pub enum VerifoneError {
    /// The terminal driver has not been initialised.
    #[error("terminal not initialized")]
    NotInitialized,

    /// One or more arguments are invalid.
    #[error("invalid parameter")]
    InvalidParameter,

    /// The operation exceeded its time limit.
    #[error("operation timed out")]
    Timeout,

    /// The card could not be read successfully.
    #[error("card read failed")]
    CardReadFailed,

    /// A payment processing failure occurred.
    #[error("payment processing error")]
    Processing,

    /// A network communication error occurred.
    #[error("network error")]
    Network,

    /// The operation was cancelled by the user or system.
    #[error("operation canceled")]
    Canceled,

    /// The terminal connection is closed.
    #[error("terminal closed")]
    Closed,

    /// An SDK-specific error code not covered by the variants above.
    #[error("unknown Verifone error: {0}")]
    Unknown(VerifoneStatus),
}

impl VerifoneError {
    /// Convert a raw SDK status code into a Rust error.
    ///
    /// Returns `None` when the status is [`ffi::VX_OK`].
    pub fn from_status(status: VerifoneStatus) -> Option<Self> {
        match status {
            ffi::VX_OK => None,
            ffi::VX_ERR_NOT_INITIALIZED => Some(Self::NotInitialized),
            ffi::VX_ERR_INVALID_PARAM => Some(Self::InvalidParameter),
            ffi::VX_ERR_TIMEOUT => Some(Self::Timeout),
            ffi::VX_ERR_CARD_READ_FAILED => Some(Self::CardReadFailed),
            ffi::VX_ERR_PROCESSING => Some(Self::Processing),
            ffi::VX_ERR_NETWORK => Some(Self::Network),
            ffi::VX_ERR_CANCELED => Some(Self::Canceled),
            ffi::VX_ERR_CLOSED => Some(Self::Closed),
            _ => Some(Self::Unknown(status)),
        }
    }
}

fn check_status(status: VerifoneStatus) -> Result<(), VerifoneError> {
    match VerifoneError::from_status(status) {
        None => Ok(()),
        Some(err) => Err(err),
    }
}

/// Initialise the Verifone terminal driver.
///
/// Uses the default SDK configuration. Call [`close_terminal`] to release
/// resources when the terminal is no longer required.
pub fn init_terminal() -> Result<(), VerifoneError> {
    let config = CString::new("").map_err(|_| VerifoneError::InvalidParameter)?;
    let status = unsafe { ffi::VxInitTerminal(config.as_ptr()) };
    check_status(status)
}

/// Start a new transaction.
///
/// # Arguments
///
/// * `amount`   - Amount in the smallest currency unit (e.g. cents).
/// * `currency` - Three-letter ISO-4217 currency code (e.g. `"USD"`).
pub fn start_transaction(amount: i64, currency: &str) -> Result<(), VerifoneError> {
    if currency.len() != 3 {
        return Err(VerifoneError::InvalidParameter);
    }

    let currency_cstr =
        CString::new(currency).map_err(|_| VerifoneError::InvalidParameter)?;

    let status = unsafe { ffi::VxStartTransaction(amount, currency_cstr.as_ptr()) };
    check_status(status)
}

/// Wait for the cardholder to present a card.
///
/// Uses a default timeout of 30 seconds.
pub fn wait_for_card() -> Result<(), VerifoneError> {
    const TIMEOUT_MS: u32 = 30_000;
    let status = unsafe { ffi::VxWaitForCard(TIMEOUT_MS) };
    check_status(status)
}

/// Process the payment for the active transaction.
///
/// On success, returns the transaction identifier assigned by the SDK.
pub fn process_payment() -> Result<String, VerifoneError> {
    let mut buffer = vec![0u8; (ffi::VX_TXN_ID_LEN as usize) + 1];

    let status = unsafe {
        ffi::VxProcessPayment(buffer.as_mut_ptr() as *mut c_char, buffer.len())
    };
    check_status(status)?;

    let cstr = unsafe { CStr::from_ptr(buffer.as_ptr() as *const c_char) };
    cstr.to_str()
        .map(|s| s.to_owned())
        .map_err(|_| VerifoneError::Processing)
}

/// Refund a previously completed transaction.
///
/// # Arguments
///
/// * `transaction_id` - The identifier returned by [`process_payment`].
pub fn refund(transaction_id: &str) -> Result<(), VerifoneError> {
    let txn_cstr =
        CString::new(transaction_id).map_err(|_| VerifoneError::InvalidParameter)?;
    let status = unsafe { ffi::VxRefund(txn_cstr.as_ptr()) };
    check_status(status)
}

/// Close the terminal driver and release any associated resources.
pub fn close_terminal() -> Result<(), VerifoneError> {
    let status = unsafe { ffi::VxCloseTerminal() };
    check_status(status)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::raw::{c_char, c_int};
    use std::sync::atomic::{AtomicI32, Ordering};
    use std::sync::Mutex;

    /// Global status that mock FFI functions will return on the next call.
    static NEXT_STATUS: AtomicI32 = AtomicI32::new(0);

    /// Counter for generating deterministic transaction identifiers in mocks.
    static NEXT_TXN: AtomicI32 = AtomicI32::new(1);

    /// Serialises tests that mutate shared mock state so that the value of
    /// `NEXT_STATUS` is stable between `set_status` and the wrapper call.
    static TEST_MUTEX: Mutex<()> = Mutex::new(());

    /// Override the symbols expected by the generated FFI bindings with test
    /// doubles. Because the bindings only declare these functions, the linker
    /// resolves calls from the safe wrapper to these mock implementations.
    #[no_mangle]
    pub unsafe extern "C" fn VxInitTerminal(_config_path: *const c_char) -> c_int {
        NEXT_STATUS.load(Ordering::SeqCst)
    }

    #[no_mangle]
    pub unsafe extern "C" fn VxStartTransaction(
        _amount: i64,
        _currency: *const c_char,
    ) -> c_int {
        NEXT_STATUS.load(Ordering::SeqCst)
    }

    #[no_mangle]
    pub unsafe extern "C" fn VxWaitForCard(_timeout_ms: u32) -> c_int {
        NEXT_STATUS.load(Ordering::SeqCst)
    }

    #[no_mangle]
    pub unsafe extern "C" fn VxProcessPayment(
        transaction_id: *mut c_char,
        len: usize,
    ) -> c_int {
        let status = NEXT_STATUS.load(Ordering::SeqCst);
        if status == ffi::VX_OK && !transaction_id.is_null() && len > 0 {
            let id = format!("txn-{:08}", NEXT_TXN.fetch_add(1, Ordering::SeqCst));
            let bytes = id.as_bytes();
            let copy_len = bytes.len().min(len.saturating_sub(1));
            std::ptr::copy_nonoverlapping(
                bytes.as_ptr() as *const c_char,
                transaction_id,
                copy_len,
            );
            *transaction_id.add(copy_len) = 0;
        }
        status
    }

    #[no_mangle]
    pub unsafe extern "C" fn VxRefund(_transaction_id: *const c_char) -> c_int {
        NEXT_STATUS.load(Ordering::SeqCst)
    }

    #[no_mangle]
    pub unsafe extern "C" fn VxCloseTerminal() -> c_int {
        NEXT_STATUS.load(Ordering::SeqCst)
    }

    fn set_status(status: c_int) {
        NEXT_STATUS.store(status, Ordering::SeqCst);
    }

    #[test]
    fn ok_status_maps_to_none() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        assert_eq!(VerifoneError::from_status(ffi::VX_OK), None);
    }

    #[test]
    fn known_error_codes_map_correctly() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_NOT_INITIALIZED),
            Some(VerifoneError::NotInitialized)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_INVALID_PARAM),
            Some(VerifoneError::InvalidParameter)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_TIMEOUT),
            Some(VerifoneError::Timeout)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_CARD_READ_FAILED),
            Some(VerifoneError::CardReadFailed)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_PROCESSING),
            Some(VerifoneError::Processing)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_NETWORK),
            Some(VerifoneError::Network)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_CANCELED),
            Some(VerifoneError::Canceled)
        );
        assert_eq!(
            VerifoneError::from_status(ffi::VX_ERR_CLOSED),
            Some(VerifoneError::Closed)
        );
    }

    #[test]
    fn unknown_error_code_maps_to_variant() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        assert_eq!(
            VerifoneError::from_status(-999),
            Some(VerifoneError::Unknown(-999))
        );
    }

    #[test]
    fn init_terminal_success_and_failure() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        assert!(init_terminal().is_ok());

        set_status(ffi::VX_ERR_NOT_INITIALIZED);
        assert_eq!(init_terminal(), Err(VerifoneError::NotInitialized));
    }

    #[test]
    fn start_transaction_rejects_invalid_currency() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        assert_eq!(
            start_transaction(100, "US"),
            Err(VerifoneError::InvalidParameter)
        );
        assert_eq!(
            start_transaction(100, "USDC"),
            Err(VerifoneError::InvalidParameter)
        );
    }

    #[test]
    fn start_transaction_accepts_valid_currency() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        assert!(start_transaction(199, "USD").is_ok());
    }

    #[test]
    fn start_transaction_propagates_sdk_error() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_ERR_NETWORK);
        assert_eq!(
            start_transaction(199, "USD"),
            Err(VerifoneError::Network)
        );
    }

    #[test]
    fn wait_for_card_propagates_timeout() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_ERR_TIMEOUT);
        assert_eq!(wait_for_card(), Err(VerifoneError::Timeout));
    }

    #[test]
    fn process_payment_returns_transaction_id() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        NEXT_TXN.store(1, Ordering::SeqCst);

        let txn = process_payment().expect("process_payment should succeed");
        assert!(txn.starts_with("txn-"));
        assert_eq!(txn.len(), 12); // "txn-" + 8 digits
    }

    #[test]
    fn process_payment_propagates_error() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_ERR_PROCESSING);
        assert_eq!(process_payment(), Err(VerifoneError::Processing));
    }

    #[test]
    fn refund_success_and_failure() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        assert!(refund("txn-12345").is_ok());

        set_status(ffi::VX_ERR_CLOSED);
        assert_eq!(refund("txn-12345"), Err(VerifoneError::Closed));
    }

    #[test]
    fn close_terminal_success_and_error() {
        let _guard = TEST_MUTEX.lock().expect("test mutex poisoned");
        set_status(ffi::VX_OK);
        assert!(close_terminal().is_ok());

        set_status(ffi::VX_ERR_CANCELED);
        assert_eq!(close_terminal(), Err(VerifoneError::Canceled));
    }
}
