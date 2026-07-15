// Package server exposes the OrderService over gRPC and HTTP/REST. The gRPC
// server implements the generated OrderServiceServer interface, while the REST
// server offers JSON endpoints that call the same service methods.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	commonv1 "github.com/astra-systems/astra-service/proto/gen/go/common"
	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/order-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const idempotencyKeyHeader = "Idempotency-Key"

// Server hosts both gRPC and HTTP listeners.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	grpcPort   string
	httpPort   string
	grpcServer *grpc.Server
	httpServer *http.Server
	service    *service.OrderService
}

// New constructs a server bound to the supplied service.
func New(grpcPort, httpPort string, svc *service.OrderService) *Server {
	return &Server{
		grpcPort: grpcPort,
		httpPort: httpPort,
		service:  svc,
	}
}

// Start registers handlers and begins serving gRPC and HTTP traffic. It blocks
// until the provided context is cancelled, then performs a graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 2)

	s.grpcServer = grpc.NewServer()
	orderv1.RegisterOrderServiceServer(s.grpcServer, s)

	lis, err := net.Listen("tcp", ":"+s.grpcPort)
	if err != nil {
		return fmt.Errorf("server: listen grpc: %w", err)
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("server: grpc serve: %w", err)
		}
	}()

	s.httpServer = &http.Server{
		Addr:         ":" + s.httpPort,
		Handler:      s.mux(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server: http serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-errCh:
		_ = s.shutdown()
		return err
	}
}

func (s *Server) shutdown() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: http shutdown: %w", err)
		}
	}
	return nil
}

// mux returns the HTTP router.
func (s *Server) mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/orders", s.handleOrders)
	mux.HandleFunc("/v1/orders/", s.handleOrder)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleOrders routes GET /v1/orders (list) and POST /v1/orders (create).
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	case http.MethodGet:
		s.handleListOrders(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleOrder routes GET /v1/orders/{id}, PATCH /v1/orders/{id}/status and
// POST /v1/orders/{id}/fulfill.
func (s *Server) handleOrder(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/orders/")
	parts := strings.SplitN(path, "/", 2)
	orderID := parts[0]
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetOrder(w, r, orderID)
	case http.MethodPatch:
		if len(parts) == 2 && parts[1] == "status" {
			s.handleUpdateOrderStatus(w, r, orderID)
			return
		}
		writeError(w, http.StatusNotFound, "not found")
	case http.MethodPost:
		if len(parts) == 2 && parts[1] == "fulfill" {
			s.handleFulfillOrder(w, r, orderID)
			return
		}
		writeError(w, http.StatusNotFound, "not found")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req orderv1.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	idempotencyKey := r.Header.Get(idempotencyKeyHeader)

	order, err := s.service.CreateOrder(r.Context(), &req, idempotencyKey)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	order, err := s.service.GetOrder(r.Context(), &orderv1.GetOrderRequest{OrderId: orderID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	req := &orderv1.ListOrdersRequest{
		StoreId:    query.Get("store_id"),
		KioskId:    query.Get("kiosk_id"),
		Pagination: &commonv1.PaginationRequest{},
	}
	if status := query.Get("status"); status != "" {
		req.Status = statusFromString(status)
	}
	if page := query.Get("page"); page != "" {
		if v, err := strconv.ParseInt(page, 10, 32); err == nil {
			req.Pagination.Page = int32(v)
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if v, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			req.Pagination.PageSize = int32(v)
		}
	}

	resp, err := s.service.ListOrders(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request, orderID string) {
	var req orderv1.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	req.OrderId = orderID

	order, err := s.service.UpdateOrderStatus(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleFulfillOrder(w http.ResponseWriter, r *http.Request, orderID string) {
	var req orderv1.FulfillOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	req.OrderId = orderID

	order, err := s.service.FulfillOrder(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// gRPC implementation of OrderServiceServer.

// CreateOrder implements the generated OrderServiceServer interface.
func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.Order, error) {
	idempotencyKey := idempotencyKeyFromGRPC(ctx)
	return s.service.CreateOrder(ctx, req, idempotencyKey)
}

// GetOrder implements the generated OrderServiceServer interface.
func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.Order, error) {
	return s.service.GetOrder(ctx, req)
}

// ListOrders implements the generated OrderServiceServer interface.
func (s *Server) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	return s.service.ListOrders(ctx, req)
}

// UpdateOrderStatus implements the generated OrderServiceServer interface.
func (s *Server) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.Order, error) {
	return s.service.UpdateOrderStatus(ctx, req)
}

// FulfillOrder implements the generated OrderServiceServer interface.
func (s *Server) FulfillOrder(ctx context.Context, req *orderv1.FulfillOrderRequest) (*orderv1.Order, error) {
	return s.service.FulfillOrder(ctx, req)
}

func idempotencyKeyFromGRPC(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(idempotencyKeyHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func statusFromString(s string) orderv1.OrderStatus {
	switch strings.ToUpper(s) {
	case "PENDING":
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case "PAID":
		return orderv1.OrderStatus_ORDER_STATUS_PAID
	case "FULFILLED":
		return orderv1.OrderStatus_ORDER_STATUS_FULFILLED
	case "CANCELLED":
		return orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	case "REFUNDED":
		return orderv1.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		_, _ = w.Write([]byte(`{"error":"failed to encode response"}`))
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpStatus := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.FailedPrecondition:
		httpStatus = http.StatusPreconditionFailed
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	}
	writeError(w, httpStatus, st.Message())
}

// Ensure Server satisfies the generated OrderServiceServer interface.
var _ orderv1.OrderServiceServer = (*Server)(nil)
