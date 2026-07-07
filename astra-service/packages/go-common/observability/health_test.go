package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type staticCheck struct {
	err error
}

func (s *staticCheck) Check(ctx context.Context) error { return s.err }

func TestCompositeChecker_AllHealthy(t *testing.T) {
	checker := NewCompositeChecker()
	checker.Register("a", &staticCheck{err: nil})
	checker.Register("b", &staticCheck{err: nil})

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected healthy checker, got %v", err)
	}
}

func TestCompositeChecker_FirstFailure(t *testing.T) {
	checker := NewCompositeChecker()
	checker.Register("db", &staticCheck{err: errors.New("db down")})
	checker.Register("cache", &staticCheck{err: errors.New("cache down")})

	err := checker.Check(context.Background())
	if err == nil {
		t.Fatal("expected unhealthy checker")
	}
	if err.Error() != "db: db down" && err.Error() != "cache: cache down" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCompositeChecker_Timeout(t *testing.T) {
	checker := NewCompositeChecker()
	checker.Register("slow", CheckFunc(func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := checker.Check(ctx); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if body := rr.Body.String(); body != `{"status":"ok"}`+"\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestLiveHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rr := httptest.NewRecorder()

	LiveHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestReadyHandler_Ready(t *testing.T) {
	checker := NewCompositeChecker()
	checker.Register("db", &staticCheck{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()

	ReadyHandler(checker)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if body := rr.Body.String(); body != `{"status":"ready"}`+"\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestReadyHandler_NotReady(t *testing.T) {
	checker := NewCompositeChecker()
	checker.Register("db", &staticCheck{err: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()

	ReadyHandler(checker)(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
	body := rr.Body.String()
	if body == "" {
		t.Fatal("expected error body")
	}
}
