package web

import (
	"encoding/json"
	"net/http"
)

// JSON writes value as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// NoStore marks a response as uncacheable. Health and authenticated pages
// should call it before writing.
func NoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}
