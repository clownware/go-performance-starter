package webutil

import (
	"net/http"
)

// IsHTMXRequest checks if the request includes the HX-Request header.
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

// SetHXTriggerAfterSwap sends an HX-Trigger-After-Swap header to trigger a client-side event after the swap step.
func SetHXTriggerAfterSwap(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger-After-Swap", event)
}

// SetHXTriggerAfterSettle sends an HX-Trigger-After-Settle header to trigger a client-side event after the settle step.
func SetHXTriggerAfterSettle(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger-After-Settle", event)
}
