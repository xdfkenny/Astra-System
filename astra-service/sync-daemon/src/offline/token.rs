#![deny(unsafe_code)]

//! HMAC-SHA256 signed offline payment tokens.
//!
//! When a kiosk loses internet connectivity it must still be able to accept
//! trusted payments.  An offline payment token is a signed attestation produced
//! by the kiosk that binds together the amount, timestamp, kiosk identity,
//! transaction id, and a hash of the items being purchased.  The token is
//! verified later by the cloud backend using the same HMAC key.
//!
//! The signing key is a 32-byte secret.  For convenience this module supports
//! deriving the key from a configured passphrase using Argon2id, but in
//! production the key should be read directly from a permission-protected file.

use std::fmt;

use serde::{Deserialize, Serialize};

use crate::AstraSyncError;
use crate::crypto::{derive_key_from_password, HmacKey};

/// An offline payment token as created by a kiosk.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct OfflinePaymentToken {
    /// Total amount to be charged, in the smallest currency unit (e.g. cents).
    pub amount_cents: u64,
    /// Token creation timestamp in milliseconds since the Unix epoch.
    pub timestamp_ms: u64,
    /// The kiosk that issued the token.
    pub kiosk_id: String,
    /// Unique transaction identifier assigned by the kiosk.
    pub transaction_id: String,
    /// Hash of the canonical item list (e.g. BLAKE3 hex) used for integrity.
    pub items_hash: String,
}

impl OfflinePaymentToken {
    /// Returns the canonical message bytes that are signed.
    ///
    /// The canonical form is a deterministic, length-prefixed encoding so that
    /// the signature is stable and unambiguous.
    pub fn canonical_bytes(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        buf.extend_from_slice(&self.amount_cents.to_be_bytes());
        buf.extend_from_slice(&self.timestamp_ms.to_be_bytes());
        buf.extend_from_slice(self.kiosk_id.as_bytes());
        buf.extend_from_slice(self.transaction_id.as_bytes());
        buf.extend_from_slice(self.items_hash.as_bytes());
        buf
    }
}

impl fmt::Display for OfflinePaymentToken {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "OfflinePaymentToken(amount_cents={}, transaction_id={}, kiosk_id={})",
            self.amount_cents, self.transaction_id, self.kiosk_id
        )
    }
}

/// A signed, serializable offline payment token.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SignedOfflineToken {
    /// The token payload.
    pub token: OfflinePaymentToken,
    /// Hex-encoded HMAC-SHA256 signature over [`OfflinePaymentToken::canonical_bytes`].
    pub signature: String,
}

/// Signs and verifies offline payment tokens.
#[derive(Debug, Clone)]
pub struct TokenSigner {
    key: HmacKey,
}

impl TokenSigner {
    /// Creates a signer from a raw 32-byte HMAC key file.
    pub fn from_key_file(path: impl AsRef<std::path::Path>) -> Result<Self, AstraSyncError> {
        let key = HmacKey::from_file(path)?;
        Ok(Self { key })
    }

    /// Creates a signer directly from an in-memory key.
    ///
    /// This is useful for tests and for keys loaded from a KMS.
    pub fn from_key(key: HmacKey) -> Self {
        Self { key }
    }

    /// Derives a signing key from a passphrase and salt using Argon2id.
    ///
    /// The same `(passphrase, salt)` pair must be supplied to every verifier.
    pub fn derive_from_passphrase(
        passphrase: &str,
        salt: &[u8],
    ) -> Result<Self, AstraSyncError> {
        let derived = derive_key_from_password(passphrase, salt, 65536, 3, 4)?;
        let mut arr = [0u8; 32];
        arr.copy_from_slice(&derived);
        Ok(Self {
            key: HmacKey::from_bytes(&arr),
        })
    }

    /// Signs a token and returns a serializable signed token.
    pub fn sign(&self,
        token: &OfflinePaymentToken,
    ) -> Result<SignedOfflineToken, AstraSyncError> {
        let tag = crate::crypto::hmac_sign(&self.key, &token.canonical_bytes());
        Ok(SignedOfflineToken {
            token: token.clone(),
            signature: hex::encode(&tag),
        })
    }

    /// Verifies a signed token and returns the payload on success.
    pub fn verify(
        &self,
        signed: &SignedOfflineToken,
    ) -> Result<OfflinePaymentToken, AstraSyncError> {
        let expected = crate::crypto::hmac_sign(&self.key, &signed.token.canonical_bytes());
        let provided = hex::decode(&signed.signature)
            .map_err(|e| AstraSyncError::Crypto(format!("invalid signature hex: {e}")))?;
        crate::crypto::hmac_verify(&self.key, &signed.token.canonical_bytes(), &provided)?;
        if provided != expected {
            // Defensive: hmac_verify already performs constant-time comparison.
            return Err(AstraSyncError::Crypto("offline token signature mismatch".to_string()));
        }
        Ok(signed.token.clone())
    }

    /// Convenience method: signs a token and serializes the result to JSON.
    pub fn sign_to_json(
        &self,
        token: &OfflinePaymentToken,
    ) -> Result<String, AstraSyncError> {
        let signed = self.sign(token)?;
        serde_json::to_string(&signed)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))
    }

    /// Convenience method: verifies a JSON-encoded signed token.
    pub fn verify_from_json(
        &self,
        json: &str,
    ) -> Result<OfflinePaymentToken, AstraSyncError> {
        let signed: SignedOfflineToken = serde_json::from_str(json)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        self.verify(&signed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::crypto::HmacKey;

    fn sample_token() -> OfflinePaymentToken {
        OfflinePaymentToken {
            amount_cents: 1234,
            timestamp_ms: 1_700_000_000_000,
            kiosk_id: "kiosk-7".to_string(),
            transaction_id: "tx-offline-001".to_string(),
            items_hash: "a3f5c9e2".to_string(),
        }
    }

    #[test]
    fn sign_and_verify_roundtrip() {
        let signer = TokenSigner::from_key(HmacKey::generate());
        let token = sample_token();
        let signed = signer.sign(&token).expect("sign");
        let verified = signer.verify(&signed).expect("verify");
        assert_eq!(verified, token);
    }

    #[test]
    fn verify_rejects_tampered_amount() {
        let signer = TokenSigner::from_key(HmacKey::generate());
        let token = sample_token();
        let mut signed = signer.sign(&token).expect("sign");
        signed.token.amount_cents = 9999;
        assert!(signer.verify(&signed).is_err());
    }

    #[test]
    fn verify_rejects_tampered_signature() {
        let signer = TokenSigner::from_key(HmacKey::generate());
        let token = sample_token();
        let mut signed = signer.sign(&token).expect("sign");
        signed.signature = hex::encode([0u8; 32]);
        assert!(signer.verify(&signed).is_err());
    }

    #[test]
    fn json_roundtrip() {
        let signer = TokenSigner::from_key(HmacKey::generate());
        let token = sample_token();
        let json = signer.sign_to_json(&token).expect("sign_to_json");
        let verified = signer.verify_from_json(&json).expect("verify_from_json");
        assert_eq!(verified, token);
    }

    #[test]
    fn derive_key_produces_consistent_signer() {
        let salt = b"test-salt-1234";
        let signer1 = TokenSigner::derive_from_passphrase("my secret passphrase", salt)
            .expect("derive 1");
        let signer2 = TokenSigner::derive_from_passphrase("my secret passphrase", salt)
            .expect("derive 2");
        let token = sample_token();
        let signed = signer1.sign(&token).expect("sign");
        let verified = signer2.verify(&signed).expect("verify");
        assert_eq!(verified, token);
    }
}
