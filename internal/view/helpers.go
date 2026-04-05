package view

import (
	"net/http"
)

// IsHTMXRequest checks if the request is from HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// SetHXRedirect sends an HX-Redirect header for client-side redirection.
func SetHXRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
}

// SetHXTrigger sends an HX-Trigger header to trigger a client-side event.
func SetHXTrigger(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger", event)
}
