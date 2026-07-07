//! Strongly-typed error codes returned by the Verifone terminal SDK.

use std::fmt;

/// Errors that can occur when interacting with a Verifone terminal.
///
/// Each variant maps to a stable integer code returned by the C SDK (see
/// `packages/verifone-ffi/include/verifone.h`).  Unknown codes are preserved
/// verbatim in [`VerifoneError::SdkCode`] so that integration logs remain
/// actionable.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum VerifoneError {
    /// An unknown SDK failure.
    Unknown,
    /// A function argument was invalid.
    InvalidParameter,
    /// The terminal has not been initialized.
    NotInitialized,
    /// The terminal is already initialized.
    AlreadyInitialized,
    /// Communication with the terminal failed.
    Communication,
    /// The operation timed out.
    Timeout,
    /// The card could not be read.
    CardRead,
    /// The cardholder cancelled the transaction.
    TransactionCancelled,
    /// The issuer declined the transaction.
    Declined,
    /// The terminal could not reach the acquirer network.
    Network,
    /// The transaction is not eligible for refund.
    RefundNotAllowed,
    /// The requested amount exceeds the terminal or issuer limit.
    AmountExceedsLimit,
    /// The currency is not supported by the terminal configuration.
    CurrencyNotSupported,
    /// The terminal is busy with another transaction.
    TerminalBusy,
    /// The SDK returned a successful status but a required handle was null.
    UnexpectedNullHandle,
    /// An unrecognized SDK error code was returned.
    SdkCode(i32),
}

impl VerifoneError {
    /// Maps an SDK result code to a [`VerifoneError`].
    ///
    /// `VX_OK` (0) maps to `Ok(())`; all other codes map to `Err(...)`.
    pub fn from_result(code: i32) -> Result<(), Self> {
        match code {
            0 => Ok(()),
            1 => Err(Self::Unknown),
            2 => Err(Self::InvalidParameter),
            3 => Err(Self::NotInitialized),
            4 => Err(Self::AlreadyInitialized),
            5 => Err(Self::Communication),
            6 => Err(Self::Timeout),
            7 => Err(Self::CardRead),
            8 => Err(Self::TransactionCancelled),
            9 => Err(Self::Declined),
            10 => Err(Self::Network),
            11 => Err(Self::RefundNotAllowed),
            12 => Err(Self::AmountExceedsLimit),
            13 => Err(Self::CurrencyNotSupported),
            14 => Err(Self::TerminalBusy),
            other => Err(Self::SdkCode(other)),
        }
    }
}

impl fmt::Display for VerifoneError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unknown => write!(f, "unknown Verifone SDK error"),
            Self::InvalidParameter => write!(f, "invalid parameter"),
            Self::NotInitialized => write!(f, "terminal not initialized"),
            Self::AlreadyInitialized => write!(f, "terminal already initialized"),
            Self::Communication => write!(f, "terminal communication failure"),
            Self::Timeout => write!(f, "terminal operation timed out"),
            Self::CardRead => write!(f, "card read failed"),
            Self::TransactionCancelled => write!(f, "transaction cancelled"),
            Self::Declined => write!(f, "transaction declined"),
            Self::Network => write!(f, "terminal network error"),
            Self::RefundNotAllowed => write!(f, "refund not allowed"),
            Self::AmountExceedsLimit => write!(f, "amount exceeds limit"),
            Self::CurrencyNotSupported => write!(f, "currency not supported"),
            Self::TerminalBusy => write!(f, "terminal busy"),
            Self::UnexpectedNullHandle => {
                write!(f, "SDK returned success but the handle was null")
            }
            Self::SdkCode(code) => write!(f, "unrecognized Verifone SDK error code {code}"),
        }
    }
}

impl std::error::Error for VerifoneError {}

impl From<VerifoneError> for crate::AstraSyncError {
    fn from(err: VerifoneError) -> Self {
        crate::AstraSyncError::Verifone(err.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ok_maps_to_ok() {
        assert!(VerifoneError::from_result(0).is_ok());
    }

    #[test]
    fn known_codes_map_to_variants() {
        assert_eq!(
            VerifoneError::from_result(9).unwrap_err(),
            VerifoneError::Declined
        );
        assert_eq!(
            VerifoneError::from_result(14).unwrap_err(),
            VerifoneError::TerminalBusy
        );
    }

    #[test]
    fn unknown_code_is_preserved() {
        assert_eq!(
            VerifoneError::from_result(999).unwrap_err(),
            VerifoneError::SdkCode(999)
        );
    }
}
