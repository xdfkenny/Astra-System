// Package auth implements gRPC authentication for kiosk mesh leaders.
package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"

	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// KioskIDKey is the context key used to propagate the authenticated kiosk ID.
type KioskIDKey struct{}

// Interceptor returns a gRPC unary interceptor that authenticates every
// incoming request as an active kiosk leader. It expects an
// "authorization" metadata header formatted as "Bearer <token>" and a
// request message carrying a kiosk_id field. The token must equal the kiosk's
// stored signing key hash.
func Interceptor(store repository.Store) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		kioskID, err := extractKioskID(req)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "auth: %v", err)
		}

		kiosk, err := store.GetKiosk(ctx, kioskID)
		if err != nil {
			if errors.Is(err, model.ErrKioskNotFound) {
				return nil, status.Errorf(codes.Unauthenticated, "auth: kiosk not found")
			}
			return nil, status.Errorf(codes.Internal, "auth: lookup kiosk: %v", err)
		}
		if !kiosk.IsLeader {
			return nil, status.Errorf(codes.PermissionDenied, "auth: kiosk is not the mesh leader")
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "auth: %v", err)
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(kiosk.SigningKeyHash)) != 1 {
			return nil, status.Errorf(codes.Unauthenticated, "auth: invalid credentials")
		}

		ctx = context.WithValue(ctx, KioskIDKey{}, kioskID)
		return handler(ctx, req)
	}
}

func extractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(vals[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be Bearer token")
	}
	if parts[1] == "" {
		return "", errors.New("empty bearer token")
	}
	return parts[1], nil
}

type kioskIDCarrier interface {
	GetKioskId() string
}

func extractKioskID(req any) (string, error) {
	c, ok := req.(kioskIDCarrier)
	if !ok {
		return "", fmt.Errorf("request type %T does not expose kiosk_id", req)
	}
	if c.GetKioskId() == "" {
		return "", errors.New("kiosk_id is required")
	}
	return c.GetKioskId(), nil
}
