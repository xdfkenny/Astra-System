package webauthn

import (
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	gwebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// FakeVerifier is a test double that returns predetermined responses.
type FakeVerifier struct {
	BeginRegistrationFunc  func(user User, opts ...gwebauthn.RegistrationOption) (*protocol.CredentialCreation, *SessionData, error)
	FinishRegistrationFunc func(user User, session SessionData, response *protocol.ParsedCredentialCreationData) (*Credential, error)
	BeginLoginFunc         func(user User, opts ...gwebauthn.LoginOption) (*protocol.CredentialAssertion, *SessionData, error)
	FinishLoginFunc        func(user User, session SessionData, response *protocol.ParsedCredentialAssertionData) (*Credential, error)
}

// BeginRegistration delegates to BeginRegistrationFunc or returns a canned challenge.
func (f *FakeVerifier) BeginRegistration(user User, opts ...gwebauthn.RegistrationOption) (*protocol.CredentialCreation, *SessionData, error) {
	if f.BeginRegistrationFunc != nil {
		return f.BeginRegistrationFunc(user, opts...)
	}
	return &protocol.CredentialCreation{
		Response: protocol.PublicKeyCredentialCreationOptions{
			Challenge: []byte("challenge"),
			RelyingParty: protocol.RelyingPartyEntity{
				CredentialEntity: protocol.CredentialEntity{Name: "Astra"},
				ID:               "localhost",
			},
			User: protocol.UserEntity{
				CredentialEntity: protocol.CredentialEntity{Name: "actor"},
				DisplayName:      "actor",
				ID:               []byte("actor"),
			},
			Parameters: []protocol.CredentialParameter{
				{Type: "public-key", Algorithm: -7},
			},
			Timeout: 60000,
		},
	}, &SessionData{Challenge: "challenge", RelyingPartyID: "localhost"}, nil
}

// FinishRegistration delegates to FinishRegistrationFunc or returns success.
func (f *FakeVerifier) FinishRegistration(user User, session SessionData, response *protocol.ParsedCredentialCreationData) (*Credential, error) {
	if f.FinishRegistrationFunc != nil {
		return f.FinishRegistrationFunc(user, session, response)
	}
	return &Credential{ID: []byte("credential-id"), PublicKey: []byte("public-key")}, nil
}

// BeginLogin delegates to BeginLoginFunc or returns a canned challenge.
func (f *FakeVerifier) BeginLogin(user User, opts ...gwebauthn.LoginOption) (*protocol.CredentialAssertion, *SessionData, error) {
	if f.BeginLoginFunc != nil {
		return f.BeginLoginFunc(user, opts...)
	}
	return &protocol.CredentialAssertion{
		Response: protocol.PublicKeyCredentialRequestOptions{
			Challenge:        []byte("challenge"),
			RelyingPartyID:   "localhost",
			UserVerification: protocol.VerificationPreferred,
			Timeout:          60000,
		},
	}, &SessionData{Challenge: "challenge", RelyingPartyID: "localhost"}, nil
}

// FinishLogin delegates to FinishLoginFunc or returns success.
func (f *FakeVerifier) FinishLogin(user User, session SessionData, response *protocol.ParsedCredentialAssertionData) (*Credential, error) {
	if f.FinishLoginFunc != nil {
		return f.FinishLoginFunc(user, session, response)
	}
	return &Credential{ID: []byte("credential-id")}, nil
}

// Ensure FakeVerifier satisfies Verifier.
var _ Verifier = (*FakeVerifier)(nil)

// ErrVerificationFailed is a convenience error for tests.
var ErrVerificationFailed = fmt.Errorf("verification failed")
