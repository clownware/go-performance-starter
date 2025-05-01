package view

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// Template helper functions for use in templates

// TemplateFuncs returns a map of template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// Convert a string to raw HTML (use with caution)
		"rawHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		// Format a time.Time
		"formatTime": func(t time.Time, format string) string {
			return t.Format(format)
		},
		// Check if a string is in a slice
		"contains": func(s []string, e string) bool {
			for _, a := range s {
				if a == e {
					return true
				}
			}
			return false
		},
		// Convert a string to lowercase
		"toLowerCase": strings.ToLower,
		// Convert a string to uppercase
		"toUpperCase": strings.ToUpper,
		// Convert a value to string
		"toString": func(v interface{}) string {
			switch v := v.(type) {
			case string:
				return v
			case int:
				return strconv.Itoa(v)
			case int64:
				return strconv.FormatInt(v, 10)
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64)
			case bool:
				return strconv.FormatBool(v)
			default:
				return ""
			}
		},
		// Add values
		"add": func(a, b int) int {
			return a + b
		},
		// Get current year (for footer copyright)
		"currentYear": func() int {
			return time.Now().Year()
		},
	}
}
