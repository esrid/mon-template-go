package subscriber

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/esrid/mon-template-go/internal/platform/web"
)

// maxRequestBody caps sign-up payloads. An email does not need more.
const maxRequestBody = 4 << 10

// Mount registers this feature's routes, declaring its own paths.
func Mount(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("POST /subscribers", create(service))
	mux.HandleFunc("GET /subscribers", list(service))
	mux.HandleFunc("GET /subscribers/{id}", show(service))
}

func create(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Email string `json:"email"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			web.JSON(w, http.StatusBadRequest, errorBody("request body must be a JSON object with an email field"))
			return
		}

		created, err := service.Subscribe(r.Context(), payload.Email)
		if err != nil {
			fail(w, r, err)
			return
		}
		w.Header().Set("Location", "/subscribers/"+strconv.FormatInt(created.ID, 10))
		web.JSON(w, http.StatusCreated, created)
	}
}

func list(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subscribers, err := service.List(r.Context())
		if err != nil {
			fail(w, r, err)
			return
		}
		web.JSON(w, http.StatusOK, subscribers)
	}
}

func show(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			web.JSON(w, http.StatusNotFound, errorBody("subscriber not found"))
			return
		}

		found, err := service.ByID(r.Context(), id)
		if err != nil {
			fail(w, r, err)
			return
		}
		web.JSON(w, http.StatusOK, found)
	}
}

// fail maps the feature's errors to status codes. Expected rejections tell the
// client what to fix. Anything else is logged with the request id and answered
// with a bare 500: internals belong in the logs, not in the response.
func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		web.JSON(w, http.StatusBadRequest, errorBody("email is not a valid address"))
	case errors.Is(err, ErrDuplicate):
		web.JSON(w, http.StatusConflict, errorBody("email is already subscribed"))
	case errors.Is(err, ErrNotFound):
		web.JSON(w, http.StatusNotFound, errorBody("subscriber not found"))
	default:
		slog.Error("subscriber request failed",
			"request_id", web.RequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"err", err,
		)
		web.JSON(w, http.StatusInternalServerError, errorBody("internal server error"))
	}
}

func errorBody(message string) map[string]string {
	return map[string]string{"error": message}
}
