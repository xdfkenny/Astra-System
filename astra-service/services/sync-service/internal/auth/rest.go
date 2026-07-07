package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RESTAuth authenticates REST requests the same way the gRPC interceptor
// authenticates incoming RPCs: the kiosk must exist, be the mesh leader, and
// supply a bearer token equal to its signing key hash.
type RESTAuth struct {
	store   repository.Store
	timeout time.Duration
}

// NewRESTAuth returns an authenticator for HTTP handlers.
func NewRESTAuth(store repository.Store, timeout time.Duration) *RESTAuth {
	return &RESTAuth{store: store, timeout: timeout}
}

// Authenticate verifies the request authorization for the supplied kioskID
// and returns a context carrying the authenticated kiosk identity.
func (a *RESTAuth) Authenticate(ctx context.Context, r *http.Request, kioskID string) (context.Context, error) {
	if kioskID == "" {
		return nil, status.Error(codes.InvalidArgument, "kiosk_id is required")
	}

	authCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	kiosk, err := a.store.GetKiosk(authCtx, kioskID)
	if err != nil {
		if errors.Is(err, model.ErrKioskNotFound) {
			return nil, status.Error(codes.Unauthenticated, "kiosk not found")
		}
		return nil, status.Errorf(codes.Internal, "lookup kiosk: %v", err)
	}
	if !kiosk.IsLeader {
		return nil, status.Error(codes.PermissionDenied, "kiosk is not the mesh leader")
	}

	token, err := extractBearerFromHeader(r.Header.Get("Authorization"))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(kiosk.SigningKeyHash)) != 1 {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return context.WithValue(ctx, KioskIDKey{}, kioskID), nil
}

func extractBearerFromHeader(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be Bearer token")
	}
	if parts[1] == "" {
		return "", errors.New("empty bearer token")
	}
	return parts[1], nil
}

// Unused helpers kept to mirror the gRPC metadata extraction pattern.
func _() {
	_ = fmt.Sprintf("")
}
