package service

import (
	"context"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/webauthn-service/internal/repository"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/webauthn"
	authv1 "github.com/astra-systems/astra-service/proto/gen/go/auth"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginVerification(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCredential(&repository.Credential{
		ActorID:      "emp-1",
		ActorType:    repository.ActorTypeEmployee,
		CredentialID: "cred-1",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	})
	verifier := &webauthn.FakeVerifier{}
	svc := NewAuthService(repo, verifier, []byte("secret-secret-secret-secret-secret"), 5*time.Minute)

	resp, err := svc.BeginVerification(context.Background(), &authv1.BeginVerificationRequest{
		ActorId:   "emp-1",
		ActorType: authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		StoreId:   "store-1",
		KioskId:   "kiosk-1",
		Reason:    "override",
	})
	require.NoError(t, err)
	assert.Equal(t, "challenge", resp.Challenge)
	assert.Equal(t, "localhost", resp.RpId)
}

func TestBeginVerificationMissingActor(t *testing.T) {
	svc := NewAuthService(repository.NewMemoryRepository(), &webauthn.FakeVerifier{}, []byte("secret"), 5*time.Minute)
	_, err := svc.BeginVerification(context.Background(), &authv1.BeginVerificationRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "actor_id is required")
}

func TestBeginVerificationCredentialNotFound(t *testing.T) {
	svc := NewAuthService(repository.NewMemoryRepository(), &webauthn.FakeVerifier{}, []byte("secret"), 5*time.Minute)
	_, err := svc.BeginVerification(context.Background(), &authv1.BeginVerificationRequest{
		ActorId:   "emp-1",
		ActorType: authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credential not found")
}

func TestVerifyAssertion(t *testing.T) {
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

	verifier := &webauthn.FakeVerifier{}
	svc := NewAuthService(repo, verifier, []byte("secret-secret-secret-secret-secret"), 5*time.Minute)

	resp, err := svc.VerifyAssertion(context.Background(), &authv1.VerifyAssertionRequest{
		ActorId:      "emp-1",
		ActorType:    authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion:    validAssertion(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.NotEmpty(t, resp.OverrideToken)
	assert.Equal(t, "emp-1", resp.ActorId)
}

func TestVerifyAssertionInvalidChallenge(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCredential(&repository.Credential{
		ActorID:      "emp-1",
		ActorType:    repository.ActorTypeEmployee,
		CredentialID: "cred-1",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	})
	verifier := &webauthn.FakeVerifier{}
	svc := NewAuthService(repo, verifier, []byte("secret"), 5*time.Minute)

	_, err := svc.VerifyAssertion(context.Background(), &authv1.VerifyAssertionRequest{
		ActorId:      "emp-1",
		ActorType:    authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion:    validAssertion(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge not found")
}

func TestVerifyAssertionVerificationFailure(t *testing.T) {
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
		RelyingPartyID: "localhost",
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
	})

	verifier := &webauthn.FakeVerifier{
		FinishLoginFunc: func(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
			return nil, webauthn.ErrVerificationFailed
		},
	}
	svc := NewAuthService(repo, verifier, []byte("secret"), 5*time.Minute)

	resp, err := svc.VerifyAssertion(context.Background(), &authv1.VerifyAssertionRequest{
		ActorId:      "emp-1",
		ActorType:    authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion:    validAssertion(),
	})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

func TestValidateOverrideToken(t *testing.T) {
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

	svc := NewAuthService(repo, &webauthn.FakeVerifier{}, []byte("secret-secret-secret-secret-secret"), 5*time.Minute)
	verifyResp, err := svc.VerifyAssertion(context.Background(), &authv1.VerifyAssertionRequest{
		ActorId:      "emp-1",
		ActorType:    authv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion:    validAssertion(),
	})
	require.NoError(t, err)
	require.True(t, verifyResp.Valid)

	validateResp, err := svc.ValidateOverrideToken(context.Background(), &authv1.ValidateOverrideTokenRequest{
		OverrideToken: verifyResp.OverrideToken,
	})
	require.NoError(t, err)
	assert.True(t, validateResp.Valid)
	assert.Equal(t, "emp-1", validateResp.ActorId)
	assert.Equal(t, authv1.ActorType_ACTOR_TYPE_EMPLOYEE, validateResp.ActorType)
}

func TestValidateOverrideTokenInvalid(t *testing.T) {
	svc := NewAuthService(repository.NewMemoryRepository(), &webauthn.FakeVerifier{}, []byte("secret"), 5*time.Minute)
	resp, err := svc.ValidateOverrideToken(context.Background(), &authv1.ValidateOverrideTokenRequest{
		OverrideToken: "not-a-token",
	})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

func validAssertion() *authv1.AuthenticatorAssertionResponse {
	clientDataJSON := []byte(`{"type":"webauthn.get","challenge":"challenge","origin":"http://localhost:5170","crossOrigin":false}`)
	authData := make([]byte, 37)
	copy(authData, []byte("rp-id-hash----------------------"))
	authData[32] = 0x01
	return &authv1.AuthenticatorAssertionResponse{
		ClientDataJson:    clientDataJSON,
		AuthenticatorData: authData,
		Signature:         []byte("signature"),
		UserHandle:        []byte("emp-1"),
	}
}
