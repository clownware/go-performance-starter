package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	startTime  time.Time
	startOnce  sync.Once
	healthDB   *pgxpool.Pool
	healthDBMu sync.RWMutex
)

// InitHealth records the server start time and stores the DB pool for health checks.
func InitHealth(db *pgxpool.Pool) {
	startOnce.Do(func() {
		startTime = time.Now()
	})
	healthDBMu.Lock()
	healthDB = db
	healthDBMu.Unlock()
}

// HealthHandler returns a simple liveness probe.
// Used by Dockerfile HEALTHCHECK — must return 200 with minimal overhead.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// HealthDetailHandler returns a detailed JSON health check with dependency status.
// Per ADR-013 Error Handling and Observability.
func HealthDetailHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "ok"

	healthDBMu.RLock()
	db := healthDB
	healthDBMu.RUnlock()

	if db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			dbStatus = "unreachable"
			status = "degraded"
		}
	} else {
		dbStatus = "not configured"
	}

	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				version = s.Value[:7]
				break
			}
		}
	}

	httpStatus := http.StatusOK
	if status != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}

	resp := map[string]interface{}{
		"status":  status,
		"version": version,
		"uptime":  time.Since(startTime).String(),
		"checks": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}
