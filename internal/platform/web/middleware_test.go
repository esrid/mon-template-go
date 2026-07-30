package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRecoversAndSetsHeaders(t *testing.T) {
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	response := httptest.NewRecorder()

	Middleware(panicking).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID header is empty")
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers were not applied")
	}
}

// A panicking request must still be access-logged with its real status, which
// only holds while accessLog stays outside recoverPanic.
func TestPanickingRequestIsStillAccessLogged(t *testing.T) {
	var logged bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	Middleware(panicking).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	output := logged.String()
	if !strings.Contains(output, "msg=request") || !strings.Contains(output, "status=500") {
		t.Fatalf("no access-log line with status 500, got:\n%s", output)
	}
}

func TestRequestIDReachesTheHandler(t *testing.T) {
	var seen string
	handler := Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestID(r.Context())
	}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if seen == "" {
		t.Fatal("handler saw an empty request id")
	}
	if got := response.Header().Get("X-Request-ID"); got != seen {
		t.Fatalf("header id = %q, handler id = %q", got, seen)
	}
}
