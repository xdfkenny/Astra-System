// Package server wires the gRPC SyncService and its JSON REST fallback into a
// single runnable server group.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/proto/gen/go/sync"
	"github.com/astra-systems/astra-service/services/sync-service/internal/auth"
	"github.com/astra-systems/astra-service/services/sync-service/internal/eventbus"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository"
	"github.com/astra-systems/astra-service/services/sync-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Server holds the gRPC and HTTP listeners and their shared dependencies.
type Server struct {
	grpcServer *grpc.Server
	httpServer *http.Server
	listener   net.Listener
}

// Config defines the listening ports and request timeout.
type Config struct {
	GRPCPort       string
	HTTPPort       string
	RequestTimeout time.Duration
}

// New creates a Server that serves gRPC on GRPCPort and JSON REST on HTTPPort.
func New(cfg Config, store repository.Store, publisher eventbus.Publisher, health *observability.CompositeChecker, log *slog.Logger) (*Server, error) {
	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return nil, fmt.Errorf("server: listen grpc: %w", err)
	}

	syncSvc := service.NewSync(store, publisher)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.Interceptor(store)),
		grpc.ConnectionTimeout(cfg.RequestTimeout),
	)
	sync.RegisterSyncServiceServer(grpcServer, syncSvc)
	reflection.Register(grpcServer)

	rest := newRESTHandler(store, publisher, cfg.RequestTimeout)
	mux := http.NewServeMux()
	mux.Handle("/v1/sync/upload", httpPost(rest.uploadBatch))
	mux.Handle("/v1/sync/download", httpPost(rest.downloadBatch))
	mux.Handle("/v1/sync/heartbeat", httpPost(rest.heartbeat))
	mux.HandleFunc("/health", observability.HealthHandler)
	mux.HandleFunc("/live", observability.LiveHandler)
	mux.HandleFunc("/ready", observability.ReadyHandler(health))

	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  cfg.RequestTimeout + 5*time.Second,
		WriteTimeout: cfg.RequestTimeout + 5*time.Second,
	}

	return &Server{
		grpcServer: grpcServer,
		httpServer: httpServer,
		listener:   grpcListener,
	}, nil
}

// Start runs the gRPC and HTTP servers concurrently until ctx is cancelled.
func (s *Server) Start(ctx context.Context, log *slog.Logger) error {
	errCh := make(chan error, 2)

	go func() {
		log.Info("grpc server listening", slog.String("addr", s.listener.Addr().String()))
		if err := s.grpcServer.Serve(s.listener); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	go func() {
		log.Info("http server listening", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully stops both servers.
func (s *Server) Shutdown(ctx context.Context) error {
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	httpErr := s.httpServer.Shutdown(ctx)

	select {
	case <-stopped:
	case <-ctx.Done():
		s.grpcServer.Stop()
	}

	if httpErr != nil {
		return fmt.Errorf("server: http shutdown: %w", httpErr)
	}
	return nil
}

type restHandler struct {
	svc  *service.Sync
	auth *auth.RESTAuth
}

func newRESTHandler(store repository.Store, publisher eventbus.Publisher, timeout time.Duration) *restHandler {
	return &restHandler{
		svc:  service.NewSync(store, publisher),
		auth: auth.NewRESTAuth(store, timeout),
	}
}

func (h *restHandler) uploadBatch(w http.ResponseWriter, r *http.Request) {
	var req sync.UploadBatchRequest
	if !decodeProtoJSON(w, r, &req) {
		return
	}
	ctx, err := h.auth.Authenticate(r.Context(), r, req.GetBatch().GetKioskId())
	if err != nil {
		writeProtoError(w, err)
		return
	}
	resp, err := h.svc.UploadBatch(ctx, &req)
	writeProtoResponse(w, resp, err)
}

func (h *restHandler) downloadBatch(w http.ResponseWriter, r *http.Request) {
	var req sync.DownloadBatchRequest
	if !decodeProtoJSON(w, r, &req) {
		return
	}
	ctx, err := h.auth.Authenticate(r.Context(), r, req.GetKioskId())
	if err != nil {
		writeProtoError(w, err)
		return
	}
	resp, err := h.svc.DownloadBatch(ctx, &req)
	writeProtoResponse(w, resp, err)
}

func (h *restHandler) heartbeat(w http.ResponseWriter, r *http.Request) {
	var req sync.HeartbeatRequest
	if !decodeProtoJSON(w, r, &req) {
		return
	}
	ctx, err := h.auth.Authenticate(r.Context(), r, req.GetKioskId())
	if err != nil {
		writeProtoError(w, err)
		return
	}
	resp, err := h.svc.Heartbeat(ctx, &req)
	writeProtoResponse(w, resp, err)
}

func httpPost(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	})
}

var protoMarshal = protojson.MarshalOptions{EmitDefaultValues: true}
var protoUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}

func decodeProtoJSON(w http.ResponseWriter, r *http.Request, msg proto.Message) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return false
	}
	defer r.Body.Close()
	if err := protoUnmarshal.Unmarshal(body, msg); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return false
	}
	return true
}

func writeProtoResponse(w http.ResponseWriter, msg proto.Message, err error) {
	if err != nil {
		writeProtoError(w, err)
		return
	}
	data, err := protoMarshal.Marshal(msg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("encode response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func writeProtoError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		st = status.New(codes.Unknown, err.Error())
	}
	code := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.PermissionDenied:
		code = http.StatusForbidden
	case codes.NotFound:
		code = http.StatusNotFound
	}
	writeJSONError(w, code, st.Message())
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// metadataFromContext is unused but kept to mirror the gRPC interceptor shape.
func metadataFromContext(ctx context.Context) metadata.MD {
	md, _ := metadata.FromIncomingContext(ctx)
	return md
}
