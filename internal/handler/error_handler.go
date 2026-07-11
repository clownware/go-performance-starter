package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// APIError represents a JSON error response
type APIError struct {
	Error string `json:"error"`
}

// JSONError writes a JSON error response with the given status code.
//
// Server errors (5xx) may carry internal detail — database/RLS messages, driver
// text — so they are logged server-side and the client receives only the
// generic status text (2026-07-06 audit). Client errors (4xx) describe the
// caller's own mistake and are echoed as-is.
func JSONError(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "status", status, "error", err)
		msg = http.StatusText(status)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(APIError{Error: msg}); encodeErr != nil {
		slog.Error("failed to encode error response", "error", encodeErr)
	}
}
