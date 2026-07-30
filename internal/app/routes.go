package app

import (
	"net/http"

	"github.com/esrid/mon-template-go/internal/feature/readiness"
	"github.com/esrid/mon-template-go/internal/feature/subscriber"
	"github.com/esrid/mon-template-go/internal/platform/web"
)

// routes mounts every feature on the root router. This is the only file that
// knows the full list of features — the Django urls.py of this template.
// It composes and nothing else: no business logic lives here.
//
// Each feature owns the paths it registers, so adding one is a single line:
//
//	billing.Mount(root, a.billing)
//
// Nothing registers "/", so unknown paths get ServeMux's 404 rather than a
// 404 that belongs to whichever feature happened to take the root pattern.
func (a *App) routes() http.Handler {
	root := http.NewServeMux()

	readiness.Mount(root, a.readiness)
	subscriber.Mount(root, a.subscribers)

	return web.Middleware(root)
}
