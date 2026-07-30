package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/esrid/mon-template-go/internal/platform/config"
)

func TestNewWiresTheReadinessSlice(t *testing.T) {
	cfg := testConfig(filepath.Join(t.TempDir(), "app.db"))
	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	if app.server.MaxHeaderBytes != cfg.MaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", app.server.MaxHeaderBytes, cfg.MaxHeaderBytes)
	}

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("readiness status = %d, want %d", response.Code, http.StatusOK)
	}
	// The middleware stack must wrap the mounted features, not bypass them.
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID header is empty: middleware is not wrapping the routes")
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	app, err := New(context.Background(), testConfig(filepath.Join(t.TempDir(), "app.db")))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })

	response := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func testConfig(dsn string) config.Config {
	return config.Config{
		HTTPAddr:          "127.0.0.1:8080",
		DatabaseDSN:       dsn,
		MaxHeaderBytes:    64 << 10,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}
}
