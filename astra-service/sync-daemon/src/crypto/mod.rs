#![deny(unsafe_code)]

//! Cryptographic primitives for the Astra sync daemon.
//!
//! Provides:
//! - XChaCha20-Poly1305 authenticated encryption for P2P sync messages.
//! - HMAC-SHA256 for signing offline payment tokens and record integrity.
//! - Blake3 for fast content hashing and deterministic IDs.
//! - Constant-time key comparison and zeroization on drop.

use chacha20poly1305::{
    aead::{Aead, KeyInit, OsRng},
    XChaCha20Poly1305, XNonce,
};
use hmac::{Hmac, Mac};
use rand::{Rng, RngCore};
use secrecy::{ExposeSecret, Secret, SecretBox, SecretVec};
use sha2::Sha256;
use zeroize::{Zeroize, ZeroizeOnDrop};

use crate::AstraSyncError;

/// 32-byte symmetric key for XChaCha20-Poly1305.
#[derive(Clone, Zeroize, ZeroizeOnDrop)]
pub struct SyncKey([u8; 32]);

impl SyncKey {
    /// Loads a 32-byte key from a file on disk. The file is expected to contain
    /// exactly 32 raw bytes (no hex encoding, no newline). Permissions are
    /// checked to be 0o600 or stricter.
    pub fn from_file(path: impl AsRef<std::path::Path>) -> Result<Self, AstraSyncError> {
        let path = path.as_ref();
        let metadata = std::fs::metadata(path)
            .map_err(|e| AstraSyncError::Crypto(format!("failed to read key file: {e}")))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::MetadataExt;
            let mode = metadata.mode() & 0o777;
            if mode > 0o600 {
                return Err(AstraSyncError::Crypto(
                    "key file permissions too permissive (must be <= 0o600)".to_string(),
                ));
            }
        }
        let bytes = std::fs::read(path)
            .map_err(|e| AstraSyncError::Crypto(format!("failed to read key file: {e}")))?;
        if bytes.len() != 32 {
            return Err(AstraSyncError::Crypto(format!(
                "key file must contain exactly 32 bytes, got {}",
                bytes.len()
            )));
        }
        let mut arr = [0u8; 32];
        arr.copy_from_slice(&bytes);
        Ok(Self(arr))
    }

    /// Generates a fresh random key.
    pub fn generate() -> Self {
        let mut key = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut key);
        Self(key)
    }

    /// Writes the key to a file with 0o600 permissions.
    pub fn write_to_file(&self, path: impl AsRef<std::path::Path>) -> Result<(), AstraSyncError> {
        let path = path.as_ref();
        std::fs::write(path, &self.0)
            .map_err(|e| AstraSyncError::Crypto(format!("failed to write key file: {e}")))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = std::fs::metadata(path)
                .map_err(|e| AstraSyncError::Io(e))?
                .permissions();
            perms.set_mode(0o600);
            std::fs::set_permissions(path, perms)
                .map_err(|e| AstraSyncError::Io(e))?;
        }
        Ok(())
    }

    pub fn as_bytes(&self) -> &[u8; 32] {
        &self.0
    }
}

/// 32-byte HMAC key for offline payment token signing.
#[derive(Clone, Zeroize, ZeroizeOnDrop)]
pub struct HmacKey([u8; 32]);

impl std::fmt::Debug for HmacKey {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("HmacKey").field("bytes", &"[REDACTED]").finish()
    }
}

impl HmacKey {
    /// Loads a 32-byte HMAC key from a file with the same permission checks as `SyncKey`.
    pub fn from_file(path: impl AsRef<std::path::Path>) -> Result<Self, AstraSyncError> {
        let key = SyncKey::from_file(path)?;
        let mut arr = [0u8; 32];
        arr.copy_from_slice(key.as_bytes());
        Ok(Self(arr))
    }

    /// Constructs an HMAC key from a 32-byte slice.
    pub fn from_bytes(bytes: &[u8; 32]) -> Self {
        Self(*bytes)
    }

    pub fn generate() -> Self {
        let mut key = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut key);
        Self(key)
    }

    pub fn as_bytes(&self) -> &[u8; 32] {
        &self.0
    }
}

/// Encrypts `plaintext` with XChaCha20-Poly1305 using a random nonce.
/// Returns `(nonce, ciphertext)` where nonce is 24 bytes.
pub fn encrypt_sync_message(
    key: &SyncKey,
    plaintext: &[u8],
) -> Result<(Vec<u8>, Vec<u8>), AstraSyncError> {
    let cipher = XChaCha20Poly1305::new_from_slice(key.as_bytes())
        .map_err(|e| AstraSyncError::Crypto(format!("invalid key length: {e}")))?;
    let nonce = XNonce::from_slice(&[0u8; 24]); // In production, use random nonce via OsRng
    let mut nonce_bytes = [0u8; 24];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = XNonce::from_slice(&nonce_bytes);
    let ciphertext = cipher
        .encrypt(nonce, plaintext)
        .map_err(|e| AstraSyncError::Crypto(format!("encryption failed: {e}")))?;
    Ok((nonce_bytes.to_vec(), ciphertext))
}

/// Decrypts a message encrypted with `encrypt_sync_message`.
pub fn decrypt_sync_message(
    key: &SyncKey,
    nonce: &[u8],
    ciphertext: &[u8],
) -> Result<Vec<u8>, AstraSyncError> {
    if nonce.len() != 24 {
        return Err(AstraSyncError::Crypto(
            "nonce must be exactly 24 bytes for XChaCha20-Poly1305".to_string(),
        ));
    }
    let cipher = XChaCha20Poly1305::new_from_slice(key.as_bytes())
        .map_err(|e| AstraSyncError::Crypto(format!("invalid key length: {e}")))?;
    let nonce = XNonce::from_slice(nonce);
    let plaintext = cipher
        .decrypt(nonce, ciphertext)
        .map_err(|e| AstraSyncError::Crypto(format!("decryption failed (bad key or tampered ciphertext): {e}")))?;
    Ok(plaintext)
}

/// Computes HMAC-SHA256 over the provided message bytes.
/// Returns a 32-byte MAC tag.
pub fn hmac_sign(key: &HmacKey, message: &[u8]) -> Vec<u8> {
    type HmacSha256 = Hmac<Sha256>;
    let mut mac = <HmacSha256 as Mac>::new_from_slice(key.as_bytes())
        .expect("HMAC can handle any key size");
    mac.update(message);
    mac.finalize().into_bytes().to_vec()
}

/// Verifies an HMAC-SHA256 tag in constant time.
pub fn hmac_verify(key: &HmacKey, message: &[u8], tag: &[u8]) -> Result<(), AstraSyncError> {
    let expected = hmac_sign(key, message);
    if subtle::constant_time_eq(&expected, tag) {
        Ok(())
    } else {
        Err(AstraSyncError::Crypto("HMAC verification failed".to_string()))
    }
}

/// Computes a Blake3 hash of the input bytes.
/// Used for content-addressed deduplication and Merkle tree roots.
pub fn content_hash(data: &[u8]) -> [u8; 32] {
    blake3::hash(data).into()
}

/// Derives a 32-byte key from a password using Argon2id.
/// This is used when the on-disk key file is itself password-protected.
pub fn derive_key_from_password(
    password: &str,
    salt: &[u8],
    m_cost: u32,
    t_cost: u32,
    p_cost: u32,
) -> Result<[u8; 32], AstraSyncError> {
    use argon2::{Argon2, PasswordHasher, Algorithm, Version, Params};
    use password_hash::Salt;

    let params = Params::new(m_cost, t_cost, p_cost, Some(32))
        .map_err(|e| AstraSyncError::Crypto(format!("invalid Argon2 params: {e}")))?;
    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    let salt_b64 = {
        use base64::Engine;
        base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(salt)
    };
    let salt = Salt::from_b64(&salt_b64)
        .map_err(|e| AstraSyncError::Crypto(format!("invalid salt: {e}")))?;
    let password_hash = argon2
        .hash_password(password.as_bytes(), salt)
        .map_err(|e| AstraSyncError::Crypto(format!("Argon2 hashing failed: {e}")))?;
    let hash = password_hash
        .hash
        .ok_or_else(|| AstraSyncError::Crypto("Argon2 produced no hash".to_string()))?;
    let mut key = [0u8; 32];
    key.copy_from_slice(hash.as_bytes());
    Ok(key)
}

/// Securely compares two byte slices in constant time to prevent timing attacks.
mod subtle {
    /// Constant-time equality comparison.
    pub fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
        if a.len() != b.len() {
            return false;
        }
        let mut result = 0u8;
        for (x, y) in a.iter().zip(b.iter()) {
            result |= x ^ y;
        }
        result == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let key = SyncKey::generate();
        let plaintext = b"The quick brown fox syncs over the lazy mesh.";
        let (nonce, ciphertext) = encrypt_sync_message(&key, plaintext).unwrap();
        let decrypted = decrypt_sync_message(&key, &nonce, &ciphertext).unwrap();
        assert_eq!(&decrypted[..], plaintext.as_slice());
    }

    #[test]
    fn test_hmac_sign_verify() {
        let key = HmacKey::generate();
        let msg = b"offline-payment-token-12345";
        let tag = hmac_sign(&key, msg);
        assert!(hmac_verify(&key, msg, &tag).is_ok());
        assert!(hmac_verify(&key, b"tampered", &tag).is_err());
    }

    #[test]
    fn test_content_hash_deterministic() {
        let data = b"deterministic hash test";
        let h1 = content_hash(data);
        let h2 = content_hash(data);
        assert_eq!(h1, h2);
        assert_ne!(h1, content_hash(b"different data"));
    }
}
