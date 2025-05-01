package handler

import (
    "net/http"
)

// HealthHandler returns a simple health check response
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
