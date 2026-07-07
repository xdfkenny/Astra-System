// Package security provides zero-trust boundary primitives shared across
// every Astra-Service Go microservice: request signing, HMAC verification,
// and constant-time comparison helpers. Every internal service-to-service
// call is authenticated with an HMAC-SHA256 signature over the canonical
// request, independent of (and in addition to) mTLS at the transport layer —
// defense in depth against a compromised sidecar/proxy.
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrSignatureMismatch is returned when an HMAC signature fails verification.
var ErrSignatureMismatch = errors.New("security: signature mismatch")

// ErrRequestExpired is returned when a signed request's timestamp is outside
// the allowed clock-skew window, preventing replay attacks.
var ErrRequestExpired = errors.New("security: request timestamp outside allowed window")

// MaxClockSkew bounds how far a signed request's timestamp may drift from
// the verifier's clock. Kiosks sync via NTP but industrial hardware clocks
// can drift after a 48h offline window, so this is intentionally generous.
const MaxClockSkew = 5 * time.Minute

// SignRequest produces a hex-encoded HMAC-SHA256 signature over a canonical
// string built from method, path, unix timestamp, and body hash. This
// canonical form (not just raw bytes) prevents ambiguity attacks where two
// different logical requests hash identically due to encoding differences.
func SignRequest(key []byte, method, path string, unixTimestamp int64, bodySHA256Hex string) string {
	canonical := canonicalForm(method, path, unixTimestamp, bodySHA256Hex)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRequest checks a signature against the canonical form and enforces
// the replay-protection time window. Uses constant-time comparison to avoid
// timing side-channel leakage of valid signature prefixes.
func VerifyRequest(key []byte, method, path string, unixTimestamp int64, bodySHA256Hex, signature string) error {
	now := time.Now().Unix()
	skew := now - unixTimestamp
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Second > MaxClockSkew {
		return fmt.Errorf("%w: skew=%ds", ErrRequestExpired, skew)
	}

	expected := SignRequest(key, method, path, unixTimestamp, bodySHA256Hex)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) != 1 {
		return ErrSignatureMismatch
	}
	return nil
}

// Sha256Hex returns the lowercase hex SHA-256 digest of the given bytes —
// used to build the bodySHA256Hex component of the canonical signing form
// without needing to buffer full request bodies into the signer itself.
func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func canonicalForm(method, path string, unixTimestamp int64, bodySHA256Hex string) string {
	var b strings.Builder
	b.WriteString(strings.ToUpper(method))
	b.WriteByte('\n')
	b.WriteString(path)
	b.WriteByte('\n')
	b.WriteString(strconv.FormatInt(unixTimestamp, 10))
	b.WriteByte('\n')
	b.WriteString(bodySHA256Hex)
	return b.String()
}
