//! Safe Rust API for the Verifone payment terminal SDK.
//!
//! The Verifone SDK is only available as a proprietary C library.  All `unsafe`
//! code is isolated in the tiny [`ffi`] submodule; this module exposes a fully
//! safe, idiomatic Rust API on top.  Callers never need to use `unsafe`.
//!
//! Crate-level `#![deny(unsafe_code)]` is impossible for a crate that calls a C
//! SDK, so every other module denies unsafe code and the unsafe boundary is
//! confined to `src/verifone/ffi.rs` with documented invariants.

#![deny(unsafe_code)]

pub mod error;
pub mod ffi;

use std::fmt;

pub use self::error::VerifoneError;

/// Opaque handle to an initialized Verifone terminal session.
///
/// The underlying C pointer is owned by this struct and released by
/// [`TerminalHandle::close`] or on drop.
pub struct TerminalHandle {
    ptr: *mut ffi::VxTerminal,
    closed: bool,
}

impl fmt::Debug for TerminalHandle {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("TerminalHandle")
            .field("ptr", &"*mut VxTerminal")
            .field("closed", &self.closed)
            .finish()
    }
}

impl Drop for TerminalHandle {
    fn drop(&mut self) {
        if !self.closed {
            let _ = self.close_internal();
        }
    }
}

impl TerminalHandle {
    fn new(ptr: *mut ffi::VxTerminal) -> Self {
        Self { ptr, closed: false }
    }

    fn close_internal(&mut self) -> Result<(), VerifoneError> {
        if self.closed {
            return Ok(());
        }
        ffi::close_terminal(self.ptr)?;
        self.closed = true;
        Ok(())
    }
}

/// Information about a presented card.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CardPresented {
    pub brand: String,
    pub last_four: String,
}

/// Payment authorization result returned by the terminal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PaymentToken {
    pub auth_code: String,
    pub opaque_token: String,
    pub terminal_id: String,
    pub authorized_amount_cents: u64,
}

/// Refund receipt returned by the terminal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RefundReceipt {
    pub reference: String,
    pub refunded_amount_cents: u64,
}

/// Initializes the connection to the Verifone terminal.
///
/// Returns a [`TerminalHandle`] that closes the connection when dropped.
pub fn init_terminal() -> Result<TerminalHandle, VerifoneError> {
    let ptr = ffi::init_terminal()?;
    Ok(TerminalHandle::new(ptr))
}

/// Begins a new transaction for the requested amount.
pub fn start_transaction(
    terminal: &mut TerminalHandle,
    amount_cents: u64,
    currency: &str,
) -> Result<(), VerifoneError> {
    if terminal.closed {
        return Err(VerifoneError::NotInitialized);
    }
    if currency.len() != 3 {
        return Err(VerifoneError::InvalidParameter);
    }
    let c_currency =
        std::ffi::CString::new(currency).map_err(|_| VerifoneError::InvalidParameter)?;
    ffi::start_transaction(terminal.ptr, amount_cents, &c_currency)
}

/// Waits until a card is presented or the timeout expires.
pub fn wait_for_card(
    terminal: &mut TerminalHandle,
    timeout_ms: u32,
) -> Result<CardPresented, VerifoneError> {
    if terminal.closed {
        return Err(VerifoneError::NotInitialized);
    }
    let info = ffi::wait_for_card(terminal.ptr, timeout_ms)?;
    if info.present == 0 {
        return Err(VerifoneError::Timeout);
    }
    Ok(CardPresented {
        brand: c_str_to_string(&info.brand).unwrap_or_default(),
        last_four: c_str_to_string(&info.last_four).unwrap_or_default(),
    })
}

/// Processes the payment after a card has been presented.
pub fn process_payment(terminal: &mut TerminalHandle) -> Result<PaymentToken, VerifoneError> {
    if terminal.closed {
        return Err(VerifoneError::NotInitialized);
    }
    let result = ffi::process_payment(terminal.ptr)?;
    Ok(PaymentToken {
        auth_code: c_str_to_string(&result.auth_code).unwrap_or_default(),
        opaque_token: c_str_to_string(&result.opaque_token).unwrap_or_default(),
        terminal_id: c_str_to_string(&result.terminal_id).unwrap_or_default(),
        authorized_amount_cents: result.authorized_amount_cents,
    })
}

/// Refunds a previous transaction.
pub fn refund(
    terminal: &mut TerminalHandle,
    transaction_id: &str,
    amount_cents: u64,
    currency: &str,
) -> Result<RefundReceipt, VerifoneError> {
    if terminal.closed {
        return Err(VerifoneError::NotInitialized);
    }
    if currency.len() != 3 {
        return Err(VerifoneError::InvalidParameter);
    }
    let c_txn =
        std::ffi::CString::new(transaction_id).map_err(|_| VerifoneError::InvalidParameter)?;
    let c_currency =
        std::ffi::CString::new(currency).map_err(|_| VerifoneError::InvalidParameter)?;
    let result = ffi::refund(terminal.ptr, &c_txn, amount_cents, &c_currency)?;
    Ok(RefundReceipt {
        reference: c_str_to_string(&result.reference).unwrap_or_default(),
        refunded_amount_cents: result.refunded_amount_cents,
    })
}

/// Closes the terminal connection and releases resources.
pub fn close_terminal(terminal: &mut TerminalHandle) -> Result<(), VerifoneError> {
    terminal.close_internal()
}

/// Converts a fixed-size C string buffer to a Rust `String`.
fn c_str_to_string(buf: &[std::ffi::c_char]) -> Result<String, VerifoneError> {
    let len = buf.iter().position(|&c| c == 0).unwrap_or(buf.len());
    let bytes: Vec<u8> = buf[..len].iter().map(|&c| c as u8).collect();
    String::from_utf8(bytes).map_err(|_| VerifoneError::InvalidParameter)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn c_str_to_string_parses_null_terminated() {
        let mut buf = [0i8; 8];
        buf[0] = b'V' as i8;
        buf[1] = b'i' as i8;
        buf[2] = b's' as i8;
        buf[3] = b'a' as i8;
        assert_eq!(c_str_to_string(&buf).unwrap(), "Visa");
    }

    #[test]
    fn c_str_to_string_parses_full_when_no_nul() {
        let buf = [b'x' as i8; 4];
        assert_eq!(c_str_to_string(&buf).unwrap(), "xxxx");
    }
}
