package view

import (
	"net/http"
	"time"

	"github.com/a-h/templ"
)

// Render renders a templ component as an HTTP response with the given status code.
func Render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// WriteHeader must come before Render: once streaming starts,
	// headers can no longer be changed. Errors during rendering are
	// logged by the caller but cannot change the response status.
	w.WriteHeader(status)
	return component.Render(r.Context(), w)
}

// CurrentYear returns the current calendar year.
func CurrentYear() int {
	return time.Now().Year()
}
