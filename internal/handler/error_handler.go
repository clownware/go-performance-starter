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

// JSONError writes a JSON error response with the given status code
func JSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(APIError{Error: err.Error()}); encodeErr != nil {
		slog.Error("failed to encode error response", "error", encodeErr)
	}
}
