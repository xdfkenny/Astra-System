package idempotency

import (
	"testing"

	"github.com/google/uuid"
)

func TestFingerprint_Deterministic(t *testing.T) {
	body := []byte(`{"order_id":"ord-1","amount_cents":1000}`)
	a := Fingerprint(body)
	b := Fingerprint(body)
	if a != b {
		t.Fatalf("fingerprint not deterministic: %s != %s", a, b)
	}
	if a == "" {
		t.Fatal("fingerprint empty")
	}
}

func TestFingerprint_DifferentInputs(t *testing.T) {
	a := Fingerprint([]byte("a"))
	b := Fingerprint([]byte("b"))
	if a == b {
		t.Fatal("different inputs produced same fingerprint")
	}
}

func TestStore_LockWithoutRedis(t *testing.T) {
	store := NewStore(nil, nil)
	ok, err := store.Lock(t.Context(), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !ok {
		t.Fatal("expected lock to succeed when redis is absent")
	}
}

func TestStore_UnlockWithoutRedis(t *testing.T) {
	store := NewStore(nil, nil)
	if err := store.Unlock(t.Context(), uuid.Must(uuid.NewV7())); err != nil {
		t.Fatalf("unlock: %v", err)
	}
}
