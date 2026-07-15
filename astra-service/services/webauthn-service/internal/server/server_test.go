package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authv1 "github.com/astra-systems/astra-service/proto/gen/go/auth"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/repository"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/service"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/webauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHTTPServerBeginVerification(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCredential(&repository.Credential{
		ActorID:      "emp-1",
		ActorType:    repository.ActorTypeEmployee,
		CredentialID: "cred-1",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	})
	svc := service.NewAuthService(repo, &webauthn.FakeVerifier{}, []byte("secret-secret-secret-secret-secret"), 5*time.Minute)
	srv := New("0", "0", svc)

	body, _ := json.Marshal(map[string]any{
		"actor_id":   "emp-1",
		"actor_type": 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/webauthn/begin", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.mux().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp authv1.BeginVerificationResponse
	require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "challenge", resp.Challenge)
}

func TestHTTPServerVerifyAssertion(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCredential(&repository.Credential{
		ActorID:      "emp-1",
		ActorType:    repository.ActorTypeEmployee,
		CredentialID: "cred-1",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	})
	repo.CreateSession(context.Background(), &repository.Session{
		SessionID:      "sess-1",
		ActorID:        "emp-1",
		ActorType:      repository.ActorTypeEmployee,
		Challenge:      "challenge",
		StoreID:        "store-1",
		TenantID:       "tenant-1",
		RelyingPartyID: "localhost",
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
	})

	svc := service.NewAuthService(repo, &webauthn.FakeVerifier{}, []byte("secret-secret-secret-secret-secret"), 5*time.Minute)
	srv := New("0", "0", svc)

	clientDataJSON := []byte(`{"type":"webauthn.get","challenge":"challenge","origin":"http://localhost:5170","crossOrigin":false}`)
	authData := make([]byte, 37)
	copy(authData, []byte("rp-id-hash----------------------"))
	authData[32] = 0x01

	reqProto := &authv1.VerifyAssertionRequest{
		ActorId:      "emp-1",
		ActorType:    authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion: &authv1.AuthenticatorAssertionResponse{
			ClientDataJson:    clientDataJSON,
			AuthenticatorData: authData,
			Signature:         []byte("signature"),
			UserHandle:        []byte("emp-1"),
		},
	}
	body, _ := protojson.Marshal(reqProto)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/webauthn/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.mux().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp authv1.VerifyAssertionResponse
	require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Valid)
	assert.NotEmpty(t, resp.OverrideToken)
}

func TestGRPCServerInterface(t *testing.T) {
	svc := service.NewAuthService(repository.NewMemoryRepository(), &webauthn.FakeVerifier{}, []byte("secret"), 5*time.Minute)
	srv := New("0", "0", svc)
	_ = srv
}
