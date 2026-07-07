// Package webauthn wraps the third-party go-webauthn library so the service
// layer can remain testable without real authenticator hardware.
package webauthn

import (
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	gwebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// User represents an actor that owns WebAuthn credentials.
type User struct {
	ID          []byte
	Name        string
	DisplayName string
	Creds       []Credential
}

// WebAuthnID returns the user handle.
func (u User) WebAuthnID() []byte { return u.ID }

// WebAuthnName returns the user identifier (e.g. email).
func (u User) WebAuthnName() string { return u.Name }

// WebAuthnDisplayName returns the human-readable name.
func (u User) WebAuthnDisplayName() string { return u.DisplayName }

// WebAuthnCredentials returns the user's credentials.
func (u User) WebAuthnCredentials() []gwebauthn.Credential {
	out := make([]gwebauthn.Credential, len(u.Creds))
	for i := range u.Creds {
		out[i] = gwebauthn.Credential{
			ID:        u.Creds[i].ID,
			PublicKey: u.Creds[i].PublicKey,
			Transport: u.Creds[i].Transport,
		}
	}
	return out
}

// Credential is a stored WebAuthn credential.
type Credential struct {
	ID        []byte
	PublicKey []byte
	Transport []protocol.AuthenticatorTransport
}

// SessionData is the challenge state returned by BeginLogin and consumed by
// FinishLogin.
type SessionData struct {
	Challenge            string
	UserID               []byte
	AllowedCredentialIDs [][]byte
	UserVerification     protocol.UserVerificationRequirement
	Extensions           protocol.AuthenticationExtensions
	RelyingPartyID       string
}

// Verifier defines the operations required by the service layer.
type Verifier interface {
	BeginRegistration(user User, opts ...gwebauthn.RegistrationOption) (*protocol.CredentialCreation, *SessionData, error)
	FinishRegistration(user User, session SessionData, response *protocol.ParsedCredentialCreationData) (*Credential, error)
	BeginLogin(user User, opts ...gwebauthn.LoginOption) (*protocol.CredentialAssertion, *SessionData, error)
	FinishLogin(user User, session SessionData, response *protocol.ParsedCredentialAssertionData) (*Credential, error)
}

// LibraryVerifier implements Verifier using go-webauthn/webauthn.
type LibraryVerifier struct {
	wa *gwebauthn.WebAuthn
}

// NewLibraryVerifier creates a verifier configured with the supplied relying party.
func NewLibraryVerifier(rpID, rpOrigin, rpName string) (*LibraryVerifier, error) {
	wa, err := gwebauthn.New(&gwebauthn.Config{
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
		RPDisplayName: rpName,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn: new: %w", err)
	}
	return &LibraryVerifier{wa: wa}, nil
}

// BeginRegistration starts a credential creation ceremony.
func (v *LibraryVerifier) BeginRegistration(user User, opts ...gwebauthn.RegistrationOption) (*protocol.CredentialCreation, *SessionData, error) {
	creation, session, err := v.wa.BeginRegistration(user, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("webauthn: begin registration: %w", err)
	}
	return creation, &SessionData{
		Challenge:            session.Challenge,
		UserID:               session.UserID,
		AllowedCredentialIDs: session.AllowedCredentialIDs,
		UserVerification:     session.UserVerification,
		Extensions:           session.Extensions,
		RelyingPartyID:       session.RelyingPartyID,
	}, nil
}

// FinishRegistration validates an attestation and returns the created credential.
func (v *LibraryVerifier) FinishRegistration(user User, session SessionData, response *protocol.ParsedCredentialCreationData) (*Credential, error) {
	cred, err := v.wa.CreateCredential(user, gwebauthn.SessionData{
		Challenge:            session.Challenge,
		UserID:               session.UserID,
		AllowedCredentialIDs: session.AllowedCredentialIDs,
		UserVerification:     session.UserVerification,
		Extensions:           session.Extensions,
		RelyingPartyID:       session.RelyingPartyID,
	}, response)
	if err != nil {
		return nil, fmt.Errorf("webauthn: finish registration: %w", err)
	}
	return &Credential{
		ID:        cred.ID,
		PublicKey: cred.PublicKey,
		Transport: cred.Transport,
	}, nil
}

// BeginLogin starts an assertion ceremony.
func (v *LibraryVerifier) BeginLogin(user User, opts ...gwebauthn.LoginOption) (*protocol.CredentialAssertion, *SessionData, error) {
	assertion, session, err := v.wa.BeginLogin(user, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("webauthn: begin login: %w", err)
	}
	return assertion, &SessionData{
		Challenge:            session.Challenge,
		UserID:               session.UserID,
		AllowedCredentialIDs: session.AllowedCredentialIDs,
		UserVerification:     session.UserVerification,
		Extensions:           session.Extensions,
		RelyingPartyID:       session.RelyingPartyID,
	}, nil
}

// FinishLogin validates a parsed assertion and returns the verified credential.
func (v *LibraryVerifier) FinishLogin(user User, session SessionData, response *protocol.ParsedCredentialAssertionData) (*Credential, error) {
	cred, err := v.wa.ValidateLogin(user, gwebauthn.SessionData{
		Challenge:            session.Challenge,
		UserID:               session.UserID,
		AllowedCredentialIDs: session.AllowedCredentialIDs,
		UserVerification:     session.UserVerification,
		Extensions:           session.Extensions,
		RelyingPartyID:       session.RelyingPartyID,
	}, response)
	if err != nil {
		return nil, fmt.Errorf("webauthn: finish login: %w", err)
	}
	return &Credential{
		ID:        cred.ID,
		PublicKey: cred.PublicKey,
		Transport: cred.Transport,
	}, nil
}

// Ensure LibraryVerifier satisfies Verifier.
var _ Verifier = (*LibraryVerifier)(nil)
