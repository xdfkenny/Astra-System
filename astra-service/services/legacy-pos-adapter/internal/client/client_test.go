package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Submit_Success(t *testing.T) {
	var captured *LegacyPOSRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/orders" {
			t.Fatalf("expected /v1/orders, got %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("expected Bearer test-key, got %s", auth)
		}
		body, _ := io.ReadAll(r.Body)
		var req LegacyPOSRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		captured = &req
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"pos_order_id":"pos-123","accepted":true}`))
	}))
	defer server.Close()

	c := New(server.URL, "test-key", 5*time.Second)
	resp, err := c.Submit(context.Background(), LegacyPOSRequest{
		OrderID:  "order-1",
		CartID:   "cart-1",
		StoreID:  "store-1",
		KioskID:  "kiosk-1",
		Total:    1000,
		Currency: "USD",
		Items:    []POSItem{{ItemID: "item-1", Name: "Burger", Quantity: 1, UnitPrice: 1000, LineTotal: 1000}},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("expected accepted")
	}
	if resp.POSOrderID != "pos-123" {
		t.Fatalf("expected pos-123, got %s", resp.POSOrderID)
	}
	if captured == nil || captured.OrderID != "order-1" {
		t.Fatalf("request not captured")
	}
}

func TestClient_Submit_Rejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"accepted":false,"error":"duplicate order"}`))
	}))
	defer server.Close()

	c := New(server.URL, "", 5*time.Second)
	resp, err := c.Submit(context.Background(), LegacyPOSRequest{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if resp.Accepted {
		t.Fatalf("expected not accepted")
	}
	if resp.Error != "duplicate order" {
		t.Fatalf("expected duplicate order, got %s", resp.Error)
	}
}

func TestClient_BaseURL(t *testing.T) {
	c := New("http://legacy-pos.example", "", time.Second)
	if c.BaseURL() != "http://legacy-pos.example" {
		t.Fatalf("unexpected base url: %s", c.BaseURL())
	}
}
