package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	webauthnv1 "github.com/astra-systems/astra-service/proto/gen/go/webauthn"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/repository"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/webauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebAuthnBeginRegistration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewWebAuthnService(repo, &webauthn.FakeVerifier{})

	resp, err := svc.BeginRegistration(context.Background(), &webauthnv1.BeginRegistrationRequest{
		ActorId:   "emp-1",
		ActorType: webauthnv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		StoreId:   "store-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "challenge", resp.Challenge)
	assert.NotNil(t, resp.Options)
	assert.Equal(t, "Astra", resp.Options.RelyingParty.Name)
}

func TestWebAuthnFinishRegistration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.CreateSession(context.Background(), &repository.Session{
		SessionID:      "sess-1",
		ActorID:        "emp-1",
		ActorType:      repository.ActorTypeEmployee,
		Challenge:      "challenge",
		RelyingPartyID: "localhost",
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
	})

	svc := NewWebAuthnService(repo, &webauthn.FakeVerifier{})
	resp, err := svc.FinishRegistration(context.Background(), &webauthnv1.FinishRegistrationRequest{
		ActorId:      "emp-1",
		ActorType:    webauthnv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Attestation:  validAttestation(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "emp-1", resp.ActorId)

	cred, err := repo.GetCredential(context.Background(), "emp-1", repository.ActorTypeEmployee)
	require.NoError(t, err)
	assert.Equal(t, []byte("public-key"), cred.PublicKey)
}

func validAttestation() *webauthnv1.AuthenticatorAttestationResponse {
	clientDataJSON, _ := base64.RawURLEncoding.DecodeString("eyJjaGFsbGVuZ2UiOiJXOEd6RlU4cEdqaG9SYldyTERsYW1BZnFfeTRTMUNaRzFWdW9lUkxBUnJFIiwib3JpZ2luIjoiaHR0cHM6Ly93ZWJhdXRobi5pbyIsInR5cGUiOiJ3ZWJhdXRobi5jcmVhdGUifQ")
	attestationObject, _ := base64.RawURLEncoding.DecodeString("o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YVjEdKbqkhPJnC90siSSsyDPQCYqlMGpUKA5fyklC2CEHvBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAQOsa7QYSUFukFOLTmgeK6x2ktirNMgwy_6vIwwtegxI2flS1X-JAkZL5dsadg-9bEz2J7PnsbB0B08txvsyUSvKlAQIDJiABIVggLKF5xS0_BntttUIrm2Z2tgZ4uQDwllbdIfrrBMABCNciWCDHwin8Zdkr56iSIh0MrB5qZiEzYLQpEOREhMUkY6q4Vw")
	return &webauthnv1.AuthenticatorAttestationResponse{
		ClientDataJson:    clientDataJSON,
		AttestationObject: attestationObject,
	}
}

func TestWebAuthnBeginAuthentication(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCredential(&repository.Credential{
		ActorID:      "emp-1",
		ActorType:    repository.ActorTypeEmployee,
		CredentialID: "cred-1",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	})
	svc := NewWebAuthnService(repo, &webauthn.FakeVerifier{})

	resp, err := svc.BeginAuthentication(context.Background(), &webauthnv1.BeginAuthenticationRequest{
		ActorId:   "emp-1",
		ActorType: webauthnv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		StoreId:   "store-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "challenge", resp.Challenge)
}

func TestWebAuthnFinishAuthentication(t *testing.T) {
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

	svc := NewWebAuthnService(repo, &webauthn.FakeVerifier{})
	resp, err := svc.FinishAuthentication(context.Background(), &webauthnv1.FinishAuthenticationRequest{
		ActorId:      "emp-1",
		ActorType:    webauthnv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Assertion:    validWebAuthnAssertion(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "emp-1", resp.ActorId)
}

func validWebAuthnAssertion() *webauthnv1.AuthenticatorAssertionResponse {
	clientDataJSON := []byte(`{"type":"webauthn.get","challenge":"challenge","origin":"http://localhost:5170","crossOrigin":false}`)
	authData := make([]byte, 37)
	copy(authData, []byte("rp-id-hash----------------------"))
	authData[32] = 0x01
	return &webauthnv1.AuthenticatorAssertionResponse{
		ClientDataJson:    clientDataJSON,
		AuthenticatorData: authData,
		Signature:         []byte("signature"),
		UserHandle:        []byte("emp-1"),
	}
}

func TestWebAuthnFinishRegistrationInvalidChallenge(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewWebAuthnService(repo, &webauthn.FakeVerifier{})

	_, err := svc.FinishRegistration(context.Background(), &webauthnv1.FinishRegistrationRequest{
		ActorId:      "emp-1",
		ActorType:    webauthnv1.ActorType_ACTOR_TYPE_EMPLOYEE,
		Challenge:    "challenge",
		CredentialId: "cred-1",
		Attestation:  validAttestation(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge not found")
}
