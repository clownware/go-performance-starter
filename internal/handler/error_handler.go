package handler

import (
	"encoding/json"
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
	json.NewEncoder(w).Encode(APIError{Error: err.Error()})
}
