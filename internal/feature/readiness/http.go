package readiness

import (
	"net/http"

	"github.com/esrid/mon-template-go/internal/platform/web"
)

// Mount registers this feature's routes on the router it is given, like
// Django's include(). Every feature exposes exactly one Mount, and
// internal/app calls it; the root router never knows what is inside.
//
// A feature declares its own full paths, so nothing outside decides where it
// lives — a prefixed feature would register "GET /billing/invoices" here.
// Leaving "/" free means the root router, not a feature, owns 404s.
func Mount(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("GET /healthz", live)
	mux.HandleFunc("GET /readyz", ready(service))
}

// live reports that the process is running. It must not touch dependencies:
// an orchestrator restarts the container when this fails.
func live(w http.ResponseWriter, _ *http.Request) {
	web.NoStore(w)
	web.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready reports whether dependencies are usable. Failing here removes the
// instance from the load balancer without restarting it.
func ready(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.NoStore(w)
		if err := service.Check(r.Context()); err != nil {
			web.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		web.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
