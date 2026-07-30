package subscriber

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mounted(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	Mount(mux, New(testStore(t), func() time.Time { return frozen }))
	return mux
}

func TestMountMapsErrorsToStatusCodes(t *testing.T) {
	handler := mounted(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"sign-up", http.MethodPost, "/subscribers", `{"email":"bob@example.com"}`, http.StatusCreated},
		{"duplicate", http.MethodPost, "/subscribers", `{"email":"BOB@example.com"}`, http.StatusConflict},
		{"invalid address", http.MethodPost, "/subscribers", `{"email":"nope"}`, http.StatusBadRequest},
		{"malformed json", http.MethodPost, "/subscribers", `{`, http.StatusBadRequest},
		{"list", http.MethodGet, "/subscribers", "", http.StatusOK},
		{"show", http.MethodGet, "/subscribers/1", "", http.StatusOK},
		{"unknown id", http.MethodGet, "/subscribers/999", "", http.StatusNotFound},
		{"non-numeric id", http.MethodGet, "/subscribers/abc", "", http.StatusNotFound},
		{"wrong method", http.MethodDelete, "/subscribers", "", http.StatusMethodNotAllowed},
	}

	// Ordered on purpose: "duplicate" depends on "sign-up" having run.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}

func TestCreateReturnsTheSubscriberAndItsLocation(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/subscribers", strings.NewReader(`{"email":"bob@example.com"}`))

	mounted(t).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	var created Subscriber
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Email != "bob@example.com" {
		t.Fatalf("email = %q", created.Email)
	}
	if !created.CreatedAt.Equal(frozen) {
		t.Fatalf("created_at = %v, want %v", created.CreatedAt, frozen)
	}
	if got := response.Header().Get("Location"); got != "/subscribers/1" {
		t.Fatalf("Location = %q, want /subscribers/1", got)
	}
}

func TestEmptyListRendersAsArrayNotNull(t *testing.T) {
	response := httptest.NewRecorder()
	mounted(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/subscribers", nil))

	if body := strings.TrimSpace(response.Body.String()); body != "[]" {
		t.Fatalf("body = %s, want []", body)
	}
}

// An unexpected store failure must not leak internals to the client.
func TestUnexpectedErrorIsAPlain500(t *testing.T) {
	boom := errors.New("connection reset by peer")
	mux := http.NewServeMux()
	Mount(mux, New(&storeStub{createErr: boom}, func() time.Time { return frozen }))

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/subscribers", strings.NewReader(`{"email":"bob@example.com"}`)))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), boom.Error()) {
		t.Fatalf("response leaked the internal error: %s", response.Body)
	}
}
