package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET returns 200 OK",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name:       "HEAD returns 200",
			method:     http.MethodHead,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			w := httptest.NewRecorder()

			HealthHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HealthHandler() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("HealthHandler() body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

// pingerFunc adapts a function to the Pinger seam so tests can script the
// dependency's answer without a database driver.
type pingerFunc func(ctx context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

// TestHealthDetailHandler_Pinger proves the readiness probe reads dependency
// health through the Pinger seam (ADR-003: handlers stay driver-free) and
// maps each outcome to the ADR-013 status contract.
func TestHealthDetailHandler_Pinger(t *testing.T) {
	tests := []struct {
		name       string
		pinger     Pinger
		wantStatus int
		wantState  string
		wantDB     string
	}{
		{
			name:       "reachable dependency is ok",
			pinger:     pingerFunc(func(context.Context) error { return nil }),
			wantStatus: http.StatusOK,
			wantState:  "ok",
			wantDB:     "ok",
		},
		{
			name:       "failing ping degrades to 503",
			pinger:     pingerFunc(func(context.Context) error { return errors.New("connection refused") }),
			wantStatus: http.StatusServiceUnavailable,
			wantState:  "degraded",
			wantDB:     "unreachable",
		},
		{
			name:       "no dependency configured is healthy",
			pinger:     nil,
			wantStatus: http.StatusOK,
			wantState:  "ok",
			wantDB:     "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitHealth(tt.pinger)
			t.Cleanup(func() { InitHealth(nil) })

			rec := httptest.NewRecorder()
			HealthDetailHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var resp struct {
				Status string            `json:"status"`
				Checks map[string]string `json:"checks"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Status != tt.wantState {
				t.Errorf("status = %q, want %q", resp.Status, tt.wantState)
			}
			if resp.Checks["database"] != tt.wantDB {
				t.Errorf("checks.database = %q, want %q", resp.Checks["database"], tt.wantDB)
			}
		})
	}
}

// TestHealthDetailHandler_PingTimeout proves the probe bounds the dependency
// call: a ping that honours ctx must be cut off, not hang the readiness check.
func TestHealthDetailHandler_PingTimeout(t *testing.T) {
	var sawDeadline bool
	InitHealth(pingerFunc(func(ctx context.Context) error {
		_, sawDeadline = ctx.Deadline()
		return nil
	}))
	t.Cleanup(func() { InitHealth(nil) })

	rec := httptest.NewRecorder()
	HealthDetailHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if !sawDeadline {
		t.Error("ping context carried no deadline — a slow dependency could hang the probe")
	}
}
