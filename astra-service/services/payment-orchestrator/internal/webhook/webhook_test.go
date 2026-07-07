package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
)

func TestParseNotification_Valid(t *testing.T) {
	body := []byte(`{"payment_id":"11111111-1111-1111-1111-111111111111","status":"CAPTURED"}`)
	n, err := ParseNotification(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if n.PaymentID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected payment_id: %s", n.PaymentID)
	}
	if n.Status != "CAPTURED" {
		t.Fatalf("unexpected status: %s", n.Status)
	}
}

func TestParseNotification_InvalidUUID(t *testing.T) {
	body := []byte(`{"payment_id":"not-a-uuid","status":"CAPTURED"}`)
	if _, err := ParseNotification(body); err == nil {
		t.Fatal("expected error for invalid uuid")
	}
}

func TestParseNotification_MissingStatus(t *testing.T) {
	body := []byte(`{"payment_id":"11111111-1111-1111-1111-111111111111"}`)
	if _, err := ParseNotification(body); err == nil {
		t.Fatal("expected error for missing status")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"payment_id":"11111111-1111-1111-1111-111111111111","status":"CAPTURED"}`)
	sig := hmacSig(body, secret)
	if err := VerifySignature(body, sig, secret); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"payment_id":"11111111-1111-1111-1111-111111111111","status":"CAPTURED"}`)
	if err := VerifySignature(body, "bad-sig", secret); err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestVerifySignature_Missing(t *testing.T) {
	if err := VerifySignature([]byte("{}"), "", []byte("secret")); err == nil {
		t.Fatal("expected error for missing signature")
	}
}

func TestMapStatus(t *testing.T) {
	cases := []struct {
		in   string
		want domain.PaymentStatus
	}{
		{"CAPTURED", domain.StatusCaptured},
		{"settled", domain.StatusSettled},
		{"FAILED", domain.StatusFailed},
		{"declined", domain.StatusFailed},
	}
	for _, tc := range cases {
		got, err := MapStatus(tc.in)
		if err != nil {
			t.Fatalf("map %q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("map %q: want %s, got %s", tc.in, tc.want, got)
		}
	}
}

func TestMapStatus_Unknown(t *testing.T) {
	if _, err := MapStatus("PENDING"); err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func hmacSig(body, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
