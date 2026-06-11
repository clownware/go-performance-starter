package view

import (
	"net/http"
)

// HTMXRequest represents information from HTMX request headers
type HTMXRequest struct {
	IsHTMX             bool
	Target             string
	Trigger            string
	TriggerName        string
	TriggerID          string
	CurrentURL         string
	PromptResponse     string
	Boosted            bool
	HistoryRestoreType string
}

// GetHTMXRequest extracts HTMX information from the HTTP request headers
func GetHTMXRequest(r *http.Request) HTMXRequest {
	return HTMXRequest{
		IsHTMX:             r.Header.Get("HX-Request") == "true",
		Target:             r.Header.Get("HX-Target"),
		Trigger:            r.Header.Get("HX-Trigger"),
		TriggerName:        r.Header.Get("HX-Trigger-Name"),
		TriggerID:          r.Header.Get("HX-Trigger-Id"),
		CurrentURL:         r.Header.Get("HX-Current-URL"),
		PromptResponse:     r.Header.Get("HX-Prompt"),
		Boosted:            r.Header.Get("HX-Boosted") == "true",
		HistoryRestoreType: r.Header.Get("HX-History-Restore-Request"),
	}
}

// IsHTMXRequest checks if the request is from HTMX
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// AddHTMXResponseHeaders sets common HTMX response headers
func AddHTMXResponseHeaders(w http.ResponseWriter, options map[string]string) {
	// Common HTMX response headers
	if options["location"] != "" {
		w.Header().Set("HX-Location", options["location"])
	}
	if options["pushUrl"] != "" {
		w.Header().Set("HX-Push-Url", options["pushUrl"])
	}
	if options["redirect"] != "" {
		w.Header().Set("HX-Redirect", options["redirect"])
	}
	if options["refresh"] == "true" {
		w.Header().Set("HX-Refresh", "true")
	}
	if options["trigger"] != "" {
		w.Header().Set("HX-Trigger", options["trigger"])
	}
	if options["triggerAfterSwap"] != "" {
		w.Header().Set("HX-Trigger-After-Swap", options["triggerAfterSwap"])
	}
	if options["triggerAfterSettle"] != "" {
		w.Header().Set("HX-Trigger-After-Settle", options["triggerAfterSettle"])
	}
}
