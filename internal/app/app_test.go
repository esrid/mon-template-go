package app

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
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

func TestPasswordResetRevokesExistingSessions(t *testing.T) {
	logs := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	app, err := New(context.Background(), testConfig(filepath.Join(t.TempDir(), "app.db")))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	handler := app.server.Handler

	registration := post(t, handler, nil, "/auth/register", `{"Email":"user@example.com","Password":"correct-horse-battery"}`)
	if registration.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", registration.Code, http.StatusCreated)
	}
	session := registration.Result().Cookies()

	if code := post(t, handler, session, "/auth/password-reset", `{"Email":"user@example.com"}`).Code; code != http.StatusAccepted {
		t.Fatalf("password reset status = %d, want %d", code, http.StatusAccepted)
	}
	token := regexp.MustCompile(`token=([A-Za-z0-9_-]+)`).FindStringSubmatch(logs.String())
	if token == nil {
		t.Fatalf("no reset token logged, got: %s", logs.String())
	}

	confirm := post(t, handler, nil, "/auth/password-reset/confirm", `{"Token":"`+token[1]+`","Password":"brand-new-passphrase"}`)
	if confirm.Code != http.StatusNoContent {
		t.Fatalf("reset confirm status = %d, want %d", confirm.Code, http.StatusNoContent)
	}

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	for _, cookie := range session {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("session survived the password reset: status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func post(t *testing.T, handler http.Handler, cookies []*http.Cookie, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
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
