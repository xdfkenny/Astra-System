package client

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	verifonepb "github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client/verifonepb/verifone/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestHTTPTransport_Authorize(t *testing.T) {
	var received any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/authorize" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED","verifone_token":"tok-123","auth_code":"auth-123"}`))
	}))
	defer server.Close()

	c := newHTTPTransport(server.URL)
	resp, err := c.Authorize(context.Background(), &AuthorizeRequest{
		PaymentID: "pay-1", OrderID: "ord-1", KioskID: "kiosk-1",
		AmountCents: 1000, Currency: "USD", Method: "credit_debit",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Status != "APPROVED" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if received == nil {
		t.Fatal("expected request body on server")
	}
}

func TestHTTPTransport_Capture_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/capture" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"DECLINED"}`))
	}))
	defer server.Close()

	c := newHTTPTransport(server.URL)
	err := c.Capture(context.Background(), "pay-1", "tok-1")
	if err == nil {
		t.Fatal("expected error for declined capture")
	}
}

func TestClient_GRPCFallbackToHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED","verifone_token":"tok-http"}`))
	}))
	defer server.Close()

	c, err := New("", server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	resp, err := c.Authorize(context.Background(), &AuthorizeRequest{
		PaymentID: "pay-1", OrderID: "ord-1", KioskID: "kiosk-1",
		AmountCents: 100, Currency: "USD", Method: "credit_debit",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.VerifoneToken != "tok-http" {
		t.Fatalf("expected tok-http, got %s", resp.VerifoneToken)
	}
}

func TestClient_GRPCTransport(t *testing.T) {
	grpcServer := grpc.NewServer()
	verifonepb.RegisterVerifoneFFIServer(grpcServer, &stubVerifoneFFIServer{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = grpcServer.Serve(ln) }()
	defer grpcServer.Stop()

	c, err := New(ln.Addr().String(), "")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	resp, err := c.Authorize(context.Background(), &AuthorizeRequest{
		PaymentID: "pay-1", OrderID: "ord-1", KioskID: "kiosk-1",
		AmountCents: 100, Currency: "USD", Method: "credit_debit",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Status != "APPROVED" {
		t.Fatalf("expected APPROVED, got %s", resp.Status)
	}
}

type stubVerifoneFFIServer struct {
	verifonepb.UnimplementedVerifoneFFIServer
}

func (s *stubVerifoneFFIServer) Authorize(_ context.Context, _ *verifonepb.AuthorizeRequest) (*verifonepb.AuthorizeResponse, error) {
	return &verifonepb.AuthorizeResponse{Status: "APPROVED", VerifoneToken: "tok-grpc", AuthCode: "auth-grpc"}, nil
}

func (s *stubVerifoneFFIServer) Capture(_ context.Context, _ *verifonepb.CaptureRequest) (*verifonepb.CaptureResponse, error) {
	return &verifonepb.CaptureResponse{Status: "CAPTURED"}, nil
}

func (s *stubVerifoneFFIServer) Settle(_ context.Context, _ *verifonepb.SettleRequest) (*verifonepb.SettleResponse, error) {
	return &verifonepb.SettleResponse{Status: "SETTLED"}, nil
}

// compile-time check that grpc.Dial still works.
var _ = insecure.NewCredentials
