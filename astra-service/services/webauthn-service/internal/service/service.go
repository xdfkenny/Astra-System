// Package service implements the WebAuthn verification use cases: beginning an
// assertion ceremony, verifying a signed assertion, and validating the issued
// override token.
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
	authv1 "github.com/astra-systems/astra-service/proto/gen/go/auth"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const overrideTokenIssuer = "astra-webauthn-service"

// AuthService implements the WebAuthn verification use cases.
type AuthService struct {
	repo     repository.Repository
	verifier webauthn.Verifier
	secret   []byte
	ttl      time.Duration
}

// NewAuthService returns a service backed by the supplied repository and verifier.
func NewAuthService(repo repository.Repository, verifier webauthn.Verifier, secret []byte, ttl time.Duration) *AuthService {
	return &AuthService{
		repo:     repo,
		verifier: verifier,
		secret:   secret,
		ttl:      ttl,
	}
}

// BeginVerification returns WebAuthn assertion options for the requested actor.
func (s *AuthService) BeginVerification(ctx context.Context, req *authv1.BeginVerificationRequest) (*authv1.BeginVerificationResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}

	actorType := actorTypeFromProto(req.ActorType)
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
		return nil, status.Errorf(codes.Internal, "begin login: %v", err)
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
		Reason:         req.Reason,
		RelyingPartyID: session.RelyingPartyID,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "store session: %v", err)
	}

	return &authv1.BeginVerificationResponse{
		Challenge:        session.Challenge,
		RpId:             session.RelyingPartyID,
		AllowCredentials: toProtoDescriptors(assertion.Response.AllowedCredentials),
		UserVerification: string(assertion.Response.UserVerification),
		TimeoutMs:        int64(assertion.Response.Timeout),
	}, nil
}

// VerifyAssertion validates a signed assertion and issues an override token.
func (s *AuthService) VerifyAssertion(ctx context.Context, req *authv1.VerifyAssertionRequest) (*authv1.VerifyAssertionResponse, error) {
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id is required")
	}
	if req.Assertion == nil {
		return nil, status.Error(codes.InvalidArgument, "assertion is required")
	}

	actorType := actorTypeFromProto(req.ActorType)
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
		Response: toAssertionResponseBody(req.Assertion),
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
		return &authv1.VerifyAssertionResponse{Valid: false}, nil
	}

	if err := s.repo.DeleteSession(ctx, session.SessionID); err != nil {
		return nil, status.Errorf(codes.Internal, "delete session: %v", err)
	}
	if err := s.repo.TouchCredential(ctx, req.ActorId, actorType); err != nil {
		return nil, status.Errorf(codes.Internal, "touch credential: %v", err)
	}

	token, expiresAt, err := s.issueOverrideToken(req.ActorId, actorType, session.StoreID, session.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "issue token: %v", err)
	}

	return &authv1.VerifyAssertionResponse{
		Valid:         true,
		OverrideToken: token,
		ExpiresAt:     timestamppb.New(expiresAt),
		ActorId:       req.ActorId,
		ActorType:     actorTypeToProto(actorType),
	}, nil
}

// ValidateOverrideToken checks whether a previously issued override token is valid.
func (s *AuthService) ValidateOverrideToken(ctx context.Context, req *authv1.ValidateOverrideTokenRequest) (*authv1.ValidateOverrideTokenResponse, error) {
	if req.OverrideToken == "" {
		return nil, status.Error(codes.InvalidArgument, "override_token is required")
	}

	token, err := jwt.Parse(req.OverrideToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithIssuer(overrideTokenIssuer))
	if err != nil || !token.Valid {
		return &authv1.ValidateOverrideTokenResponse{Valid: false}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &authv1.ValidateOverrideTokenResponse{Valid: false}, nil
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil || expiresAt.Before(time.Now().UTC()) {
		return &authv1.ValidateOverrideTokenResponse{Valid: false}, nil
	}

	return &authv1.ValidateOverrideTokenResponse{
		Valid:     true,
		ActorId:   stringValue(claims, "sub"),
		ActorType: actorTypeFromString(stringValue(claims, "actor_type")),
		StoreId:   stringValue(claims, "store_id"),
		TenantId:  stringValue(claims, "tenant_id"),
		ExpiresAt: timestamppb.New(expiresAt.Time),
	}, nil
}

func (s *AuthService) issueOverrideToken(actorID string, actorType repository.ActorType, storeID, tenantID string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(s.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":        overrideTokenIssuer,
		"sub":        actorID,
		"actor_type": string(actorType),
		"store_id":   storeID,
		"tenant_id":  tenantID,
		"iat":        time.Now().UTC().Unix(),
		"exp":        expiresAt.Unix(),
	})
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func actorTypeFromProto(at authv1.ActorType) repository.ActorType {
	switch at {
	case authv1.ActorType_ACTOR_TYPE_USER:
		return repository.ActorTypeUser
	default:
		return repository.ActorTypeEmployee
	}
}

func actorTypeFromString(s string) authv1.ActorType {
	switch s {
	case "user":
		return authv1.ActorType_ACTOR_TYPE_USER
	default:
		return authv1.ActorType_ACTOR_TYPE_EMPLOYEE
	}
}

func actorTypeToProto(at repository.ActorType) authv1.ActorType {
	switch at {
	case repository.ActorTypeUser:
		return authv1.ActorType_ACTOR_TYPE_USER
	default:
		return authv1.ActorType_ACTOR_TYPE_EMPLOYEE
	}
}

func credentialBytes(credentialID string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(credentialID)
	if err != nil {
		return []byte(credentialID)
	}
	return b
}

func credentialIDString(credentialID string) string {
	b := credentialBytes(credentialID)
	return base64.RawURLEncoding.EncodeToString(b)
}

func toProtoDescriptors(descs []protocol.CredentialDescriptor) []*authv1.PublicKeyCredentialDescriptor {
	out := make([]*authv1.PublicKeyCredentialDescriptor, len(descs))
	for i, d := range descs {
		out[i] = &authv1.PublicKeyCredentialDescriptor{
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

type assertionBody struct {
	ID       string                     `json:"id"`
	RawID    string                     `json:"rawId"`
	Type     string                     `json:"type"`
	Response assertionResponseBody      `json:"response"`
}

type assertionResponseBody struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle,omitempty"`
}

func toAssertionResponseBody(a *authv1.AuthenticatorAssertionResponse) assertionResponseBody {
	return assertionResponseBody{
		ClientDataJSON:    base64.RawURLEncoding.EncodeToString(a.ClientDataJson),
		AuthenticatorData: base64.RawURLEncoding.EncodeToString(a.AuthenticatorData),
		Signature:         base64.RawURLEncoding.EncodeToString(a.Signature),
		UserHandle:        base64.RawURLEncoding.EncodeToString(a.UserHandle),
	}
}

func stringValue(claims jwt.MapClaims, key string) string {
	v, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return v
}
