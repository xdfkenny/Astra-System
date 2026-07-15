// Package server wires the gRPC and REST servers for the inventory service.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/astra-service/go-common/observability"
	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server holds the gRPC and HTTP listeners.
type Server struct {
	grpcServer *grpc.Server
	httpServer *http.Server
	grpcPort   string
	httpPort   string
}

// New creates a Server that exposes inventorySvc over gRPC and a thin REST
// mapping over HTTP.
func New(inventorySvc *service.Inventory, grpcPort, httpPort string, enableReflection bool) *Server {
	grpcServer := grpc.NewServer()
	inventoryv1.RegisterInventoryServiceServer(grpcServer, inventorySvc)
	if enableReflection {
		reflection.Register(grpcServer)
	}

	mux := http.NewServeMux()
	registerREST(mux, inventorySvc)

	return &Server{
		grpcServer: grpcServer,
		httpServer: &http.Server{
			Addr:              ":" + httpPort,
			Handler:           mux,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
		},
		grpcPort: grpcPort,
		httpPort: httpPort,
	}
}

// Start runs the gRPC and HTTP servers in separate goroutines.
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", ":"+s.grpcPort)
	if err != nil {
		return fmt.Errorf("server: listen grpc: %w", err)
	}

	errCh := make(chan error, 2)
	go func() {
		observability.Info(ctx, "inventory-service grpc listening", slog.String("port", s.grpcPort))
		if err := s.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("server: grpc serve: %w", err)
		}
	}()
	go func() {
		observability.Info(ctx, "inventory-service http listening", slog.String("port", s.httpPort))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server: http serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

// Shutdown gracefully stops both servers.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownHttp, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdownHttp); err != nil {
		return fmt.Errorf("server: http shutdown: %w", err)
	}
	s.grpcServer.GracefulStop()
	return nil
}

func registerREST(mux *http.ServeMux, svc *service.Inventory) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("GET /live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})
	mux.HandleFunc("GET /stock", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		storeID := r.URL.Query().Get("store_id")
		itemID := r.URL.Query().Get("item_id")
		stock, err := svc.GetStock(ctx, &inventoryv1.GetStockRequest{StoreId: storeID, ItemId: itemID})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, stock)
	})
	mux.HandleFunc("POST /reserve", func(w http.ResponseWriter, r *http.Request) {
		var req inventoryv1.ReservationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		stock, err := svc.ReserveStock(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, stock)
	})
	mux.HandleFunc("POST /release", func(w http.ResponseWriter, r *http.Request) {
		var req inventoryv1.ReleaseStockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		stock, err := svc.ReleaseStock(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, stock)
	})
	mux.HandleFunc("POST /adjust", func(w http.ResponseWriter, r *http.Request) {
		var req inventoryv1.AdjustStockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		stock, err := svc.AdjustStock(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, stock)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// parseInt32 converts a string to int32; on error it returns 0.
func parseInt32(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}
