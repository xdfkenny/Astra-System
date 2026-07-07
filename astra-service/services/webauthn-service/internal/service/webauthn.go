// Package service implements WebAuthn registration and authentication flows.
package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-systems/astra-service/services/webauthn-service/internal/repository"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/webauthn"
	webauthnv1 "github.com/astra-systems/astra-service/proto/gen/go/webauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WebAuthnService implements credential registration and authentication ceremonies.
type WebAuthnService struct {
	webauthnv1.UnimplementedWebAuthnServiceServer
	repo     repository.Repository
	verifier webauthn.Verifier
}

// NewWebAuthnService returns a WebAuthnService backed by repo and verifier.
func NewWebAuthnService(repo repository.Repository, verifier webauthn.Verifier) *WebAuthnService {
	return &WebAuthnService{repo: repo, verifier: verifier}
}

// BeginRegistration returns PublicKeyCredentialCreationOptions for the actor.
func (s *WebAuthnService) BeginRegistration(ctx context.Context, req *webauthnv1.BeginRegistrationRequest) (*webauthnv1.BeginRegistrationResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}

	actorType := actorTypeFromWebAuthnProto(req.ActorType)
	user := webauthn.User{
		ID:          []byte(req.ActorId),
		Name:        req.ActorId,
		DisplayName: req.ActorId,
	}

	creation, session, err := s.verifier.BeginRegistration(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin registration: %v", err)
	}

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if err := s.repo.CreateSession(ctx, &repository.Session{
		SessionID:      uuid.New().String(),
		ActorID:        req.ActorId,
		ActorType:      actorType,
		Challenge:      session.Challenge,
		StoreID:        req.StoreId,
		TenantID:       req.TenantId,
		RelyingPartyID: session.RelyingPartyID,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "store session: %v", err)
	}

	return &webauthnv1.BeginRegistrationResponse{
		Challenge: session.Challenge,
		Options:   toProtoCreationOptions(creation),
	}, nil
}

// FinishRegistration validates an attestation and persists the credential.
func (s *WebAuthnService) FinishRegistration(ctx context.Context, req *webauthnv1.FinishRegistrationRequest) (*webauthnv1.FinishRegistrationResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}
	if req.Attestation == nil {
		return nil, status.Error(codes.InvalidArgument, "attestation is required")
	}

	actorType := actorTypeFromWebAuthnProto(req.ActorType)
	session, err := s.repo.GetSession(ctx, req.ActorId, actorType, req.Challenge)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "challenge not found or expired")
		}
		return nil, status.Errorf(codes.Internal, "load session: %v", err)
	}

	body, err := json.Marshal(attestationBody{
		ID:       credentialIDString(req.CredentialId),
		RawID:    credentialIDString(req.CredentialId),
		Type:     "public-key",
		Response: toAttestationResponseBody(req.Attestation),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal attestation: %v", err)
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(body))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attestation: %v", err)
	}

	user := webauthn.User{
		ID:          []byte(req.ActorId),
		Name:        req.ActorId,
		DisplayName: req.ActorId,
	}

	cred, err := s.verifier.FinishRegistration(user, webauthn.SessionData{
		Challenge:      session.Challenge,
		UserID:         []byte(req.ActorId),
		RelyingPartyID: session.RelyingPartyID,
	}, parsed)
	if err != nil {
		return &webauthnv1.FinishRegistrationResponse{Success: false}, nil
	}

	credentialID := base64.RawURLEncoding.EncodeToString(cred.ID)
	if err := s.repo.SaveCredential(ctx, &repository.Credential{
		ActorID:      req.ActorId,
		ActorType:    actorType,
		CredentialID: credentialID,
		PublicKey:    cred.PublicKey,
		IsActive:     true,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "save credential: %v", err)
	}

	if err := s.repo.DeleteSession(ctx, session.SessionID); err != nil {
		return nil, status.Errorf(codes.Internal, "delete session: %v", err)
	}

	return &webauthnv1.FinishRegistrationResponse{
		Success:      true,
		CredentialId: credentialID,
		ActorId:      req.ActorId,
		ActorType:    req.ActorType,
		CreatedAt:    timestamppb.New(time.Now().UTC()),
	}, nil
}

// BeginAuthentication returns assertion options for the actor.
func (s *WebAuthnService) BeginAuthentication(ctx context.Context, req *webauthnv1.BeginAuthenticationRequest) (*webauthnv1.BeginAuthenticationResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}

	actorType := actorTypeFromWebAuthnProto(req.ActorType)
	cred, err := s.repo.GetCredential(ctx, req.ActorId, actorType)
	if err != nil {
		if errors.Is(err, repository.ErrCredentialNotFound) {
			return nil, status.Errorf(codes.NotFound, "credential not found for actor %s", req.ActorId)
		}
		return nil, status.Errorf(codes.Internal, "load credential: %v", err)
	}
	if !cred.IsActive {
		return nil, status.Errorf(codes.PermissionDenied, "credential is inactive")
	}

	user := webauthn.User{
		ID:          []byte(req.ActorId),
		Name:        req.ActorId,
		DisplayName: req.ActorId,
		Creds: []webauthn.Credential{
			{ID: credentialBytes(cred.CredentialID), PublicKey: cred.PublicKey},
		},
	}

	assertion, session, err := s.verifier.BeginLogin(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin authentication: %v", err)
	}

	expiresAt := time.Now().UTC().Add(2 * time.Minute)
	if err := s.repo.CreateSession(ctx, &repository.Session{
		SessionID:      uuid.New().String(),
		ActorID:        req.ActorId,
		ActorType:      actorType,
		Challenge:      session.Challenge,
		StoreID:        req.StoreId,
		KioskID:        req.KioskId,
		TenantID:       req.TenantId,
		RelyingPartyID: session.RelyingPartyID,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "store session: %v", err)
	}

	return &webauthnv1.BeginAuthenticationResponse{
		Challenge:        session.Challenge,
		RpId:             session.RelyingPartyID,
		AllowCredentials: toWebAuthnProtoDescriptors(assertion.Response.AllowedCredentials),
		UserVerification: string(assertion.Response.UserVerification),
		TimeoutMs:        int64(assertion.Response.Timeout),
	}, nil
}

// FinishAuthentication validates a signed assertion.
func (s *WebAuthnService) FinishAuthentication(ctx context.Context, req *webauthnv1.FinishAuthenticationRequest) (*webauthnv1.FinishAuthenticationResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}
	if req.Assertion == nil {
		return nil, status.Error(codes.InvalidArgument, "assertion is required")
	}

	actorType := actorTypeFromWebAuthnProto(req.ActorType)
	session, err := s.repo.GetSession(ctx, req.ActorId, actorType, req.Challenge)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "challenge not found or expired")
		}
		return nil, status.Errorf(codes.Internal, "load session: %v", err)
	}

	cred, err := s.repo.GetCredential(ctx, req.ActorId, actorType)
	if err != nil {
		if errors.Is(err, repository.ErrCredentialNotFound) {
			return nil, status.Errorf(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "load credential: %v", err)
	}
	if !cred.IsActive {
		return nil, status.Errorf(codes.PermissionDenied, "credential is inactive")
	}

	body, err := json.Marshal(assertionBody{
		ID:       credentialIDString(req.CredentialId),
		RawID:    credentialIDString(req.CredentialId),
		Type:     "public-key",
		Response: toWebAuthnAssertionResponseBody(req.Assertion),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal assertion: %v", err)
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(body))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assertion: %v", err)
	}

	user := webauthn.User{
		ID:          []byte(req.ActorId),
		Name:        req.ActorId,
		DisplayName: req.ActorId,
		Creds: []webauthn.Credential{
			{ID: credentialBytes(cred.CredentialID), PublicKey: cred.PublicKey},
		},
	}

	_, err = s.verifier.FinishLogin(user, webauthn.SessionData{
		Challenge:            session.Challenge,
		UserID:               []byte(req.ActorId),
		AllowedCredentialIDs: [][]byte{credentialBytes(cred.CredentialID)},
		RelyingPartyID:       session.RelyingPartyID,
	}, parsed)
	if err != nil {
		return &webauthnv1.FinishAuthenticationResponse{Valid: false}, nil
	}

	if err := s.repo.DeleteSession(ctx, session.SessionID); err != nil {
		return nil, status.Errorf(codes.Internal, "delete session: %v", err)
	}
	if err := s.repo.TouchCredential(ctx, req.ActorId, actorType); err != nil {
		return nil, status.Errorf(codes.Internal, "touch credential: %v", err)
	}

	return &webauthnv1.FinishAuthenticationResponse{
		Valid:           true,
		ActorId:         req.ActorId,
		ActorType:       req.ActorType,
		CredentialId:    req.CredentialId,
		AuthenticatedAt: timestamppb.New(time.Now().UTC()),
	}, nil
}

func actorTypeFromWebAuthnProto(at webauthnv1.ActorType) repository.ActorType {
	switch at {
	case webauthnv1.ActorType_ACTOR_TYPE_USER:
		return repository.ActorTypeUser
	default:
		return repository.ActorTypeEmployee
	}
}

func toProtoCreationOptions(creation *protocol.CredentialCreation) *webauthnv1.PublicKeyCredentialCreationOptions {
	if creation == nil {
		return nil
	}
	opts := creation.Response
	out := &webauthnv1.PublicKeyCredentialCreationOptions{
		Challenge: opts.Challenge,
		RelyingParty: &webauthnv1.RelyingParty{
			Id:   opts.RelyingParty.ID,
			Name: opts.RelyingParty.Name,
		},
		User: &webauthnv1.UserEntity{
			Id:          userIDBytes(opts.User.ID),
			Name:        opts.User.Name,
			DisplayName: opts.User.DisplayName,
		},
		TimeoutMs:   int64(opts.Timeout),
		Attestation: string(opts.Attestation),
	}
	out.PubKeyCredParams = make([]*webauthnv1.PublicKeyCredentialParameters, len(opts.Parameters))
	for i, p := range opts.Parameters {
		out.PubKeyCredParams[i] = &webauthnv1.PublicKeyCredentialParameters{
			Type: string(p.Type),
			Alg:  int64(p.Algorithm),
		}
	}
	if opts.AuthenticatorSelection.AuthenticatorAttachment != "" {
		out.AuthenticatorSelection = &webauthnv1.AuthenticatorSelection{
			AuthenticatorAttachment: string(opts.AuthenticatorSelection.AuthenticatorAttachment),
			ResidentKey:             string(opts.AuthenticatorSelection.ResidentKey),
			UserVerification:        string(opts.AuthenticatorSelection.UserVerification),
		}
	}
	out.ExcludeCredentials = toWebAuthnProtoDescriptors(opts.CredentialExcludeList)
	return out
}

func userIDBytes(id any) []byte {
	if b, ok := id.([]byte); ok {
		return b
	}
	return []byte(fmt.Sprintf("%v", id))
}

func toWebAuthnProtoDescriptors(descs []protocol.CredentialDescriptor) []*webauthnv1.PublicKeyCredentialDescriptor {
	out := make([]*webauthnv1.PublicKeyCredentialDescriptor, len(descs))
	for i, d := range descs {
		out[i] = &webauthnv1.PublicKeyCredentialDescriptor{
			Type: string(d.Type),
			Id:   []byte(d.CredentialID),
		}
		if len(d.Transport) > 0 {
			out[i].Transports = make([]string, len(d.Transport))
			for j, t := range d.Transport {
				out[i].Transports[j] = string(t)
			}
		}
	}
	return out
}

type attestationBody struct {
	ID       string                `json:"id"`
	RawID    string                `json:"rawId"`
	Type     string                `json:"type"`
	Response attestationResponseBody `json:"response"`
}

type attestationResponseBody struct {
	ClientDataJSON    string   `json:"clientDataJSON"`
	AttestationObject string   `json:"attestationObject"`
	Transports        []string `json:"transports,omitempty"`
}

func toAttestationResponseBody(a *webauthnv1.AuthenticatorAttestationResponse) attestationResponseBody {
	return attestationResponseBody{
		ClientDataJSON:    base64.RawURLEncoding.EncodeToString(a.ClientDataJson),
		AttestationObject: base64.RawURLEncoding.EncodeToString(a.AttestationObject),
		Transports:        a.Transports,
	}
}

func toWebAuthnAssertionResponseBody(a *webauthnv1.AuthenticatorAssertionResponse) assertionResponseBody {
	return assertionResponseBody{
		ClientDataJSON:    base64.RawURLEncoding.EncodeToString(a.ClientDataJson),
		AuthenticatorData: base64.RawURLEncoding.EncodeToString(a.AuthenticatorData),
		Signature:         base64.RawURLEncoding.EncodeToString(a.Signature),
		UserHandle:        base64.RawURLEncoding.EncodeToString(a.UserHandle),
	}
}

var _ webauthnv1.WebAuthnServiceServer = (*WebAuthnService)(nil)
