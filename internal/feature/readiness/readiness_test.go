package readiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type storeStub struct {
	err error
}

func (s storeStub) Ping(context.Context) error { return s.err }

func TestCheck(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		if err := New(storeStub{}).Check(context.Background()); err != nil {
			t.Fatalf("Check() error = %v", err)
		}
	})

	t.Run("persistence unavailable", func(t *testing.T) {
		databaseErr := errors.New("database unavailable")
		err := New(storeStub{err: databaseErr}).Check(context.Background())
		if !errors.Is(err, databaseErr) {
			t.Fatalf("Check() error = %v, want wrapped database error", err)
		}
	})
}

func mounted(store Store) http.Handler {
	mux := http.NewServeMux()
	Mount(mux, New(store))
	return mux
}

func TestMount(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		storeErr   error
		wantStatus int
		wantBody   string
	}{
		{"live ignores the store", http.MethodGet, "/healthz", errors.New("down"), http.StatusOK, `"status":"ok"`},
		{"ready", http.MethodGet, "/readyz", nil, http.StatusOK, `"status":"ready"`},
		{"not ready", http.MethodGet, "/readyz", errors.New("down"), http.StatusServiceUnavailable, `"status":"unavailable"`},
		{"wrong method", http.MethodPost, "/healthz", nil, http.StatusMethodNotAllowed, ""},
		// The feature must not claim "/", so 404s stay the root router's job.
		{"leaves the root free", http.MethodGet, "/nope", nil, http.StatusNotFound, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)

			mounted(storeStub{err: test.storeErr}).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
			if test.wantStatus < 400 && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("health response is cacheable")
			}
		})
	}
}
