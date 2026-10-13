package middleware

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/clownware/go-performance-starter/internal/performance"
)

func TestResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "200 OK", statusCode: http.StatusOK},
		{name: "404 Not Found", statusCode: http.StatusNotFound},
		{name: "500 Internal Server Error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			rw.WriteHeader(tt.statusCode)

			if rw.statusCode != tt.statusCode {
				t.Errorf("statusCode = %d, want %d", rw.statusCode, tt.statusCode)
			}
			if w.Code != tt.statusCode {
				t.Errorf("underlying ResponseWriter code = %d, want %d", w.Code, tt.statusCode)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	data := []byte("hello world")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned %d bytes, want %d", n, len(data))
	}
	if rw.bytesWritten != len(data) {
		t.Errorf("bytesWritten = %d, want %d", rw.bytesWritten, len(data))
	}

	// Write again to test accumulation
	n2, err := rw.Write(data)
	if err != nil {
		t.Fatalf("second Write() returned error: %v", err)
	}
	if rw.bytesWritten != len(data)+n2 {
		t.Errorf("bytesWritten after two writes = %d, want %d", rw.bytesWritten, len(data)+n2)
	}
}

func TestGetRoutePattern_NoChiContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	// Without chi route context, should fall back to URL path
	pattern := getRoutePattern(req)
	if pattern != "/api/test" {
		t.Errorf("getRoutePattern() = %q, want %q", pattern, "/api/test")
	}
}

func TestMetrics_HandlerChain(t *testing.T) {
	// Verify that the Metrics middleware passes through to the next handler
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := Metrics(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Metrics middleware did not call next handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("response status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestCheckPerformanceBudget pins the ADR-000 thresholds with exact-boundary
// rows: a request at exactly the budget is within budget. Unique paths per
// row keep the global counter readable.
func TestCheckPerformanceBudget(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantP50  float64
		wantP95  float64
		wantP99  float64
	}{
		{"under every budget", 10 * time.Millisecond, 0, 0, 0},
		{"exactly p50 budget is within budget", performance.MaxP50ResponseTime, 0, 0, 0},
		{"just over p50", performance.MaxP50ResponseTime + time.Millisecond, 1, 0, 0},
		{"exactly p95 budget violates only p50", performance.MaxP95ResponseTime, 1, 0, 0},
		{"just over p95", performance.MaxP95ResponseTime + time.Millisecond, 1, 1, 0},
		{"exactly p99 budget violates p50+p95", performance.MaxP99ResponseTime, 1, 1, 0},
		{"just over p99 violates all three", performance.MaxP99ResponseTime + time.Millisecond, 1, 1, 1},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/budget-test-%d", i)
			checkPerformanceBudget(tt.duration, path, http.MethodGet)

			got := map[string]float64{
				"p50": testutil.ToFloat64(budgetViolations.WithLabelValues("p50_response_time", path)),
				"p95": testutil.ToFloat64(budgetViolations.WithLabelValues("p95_response_time", path)),
				"p99": testutil.ToFloat64(budgetViolations.WithLabelValues("p99_response_time", path)),
			}
			want := map[string]float64{"p50": tt.wantP50, "p95": tt.wantP95, "p99": tt.wantP99}
			for k := range want {
				if got[k] != want[k] {
					t.Errorf("%s violations = %v, want %v", k, got[k], want[k])
				}
			}
		})
	}
}

func TestGetRoutePattern_ChiContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/posts/123", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/api/posts/{id}"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if got := getRoutePattern(req); got != "/api/posts/{id}" {
		t.Errorf("getRoutePattern = %q, want the chi pattern", got)
	}

	// An empty pattern (mounted-but-unmatched) falls back to the raw path.
	req2 := httptest.NewRequest(http.MethodGet, "/raw/path", nil)
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	if got := getRoutePattern(req2); got != "/raw/path" {
		t.Errorf("getRoutePattern(empty pattern) = %q, want /raw/path", got)
	}
}

// TestMetrics_RequestSizeOnlyWithBody pins that the request-size histogram
// records only requests that actually carry a body — a zero-length series
// per GET would skew the distribution toward zero.
func TestMetrics_RequestSizeOnlyWithBody(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	withBody := httptest.NewRequest(http.MethodPost, "/size-test-body", strings.NewReader("hello"))
	Metrics(next).ServeHTTP(httptest.NewRecorder(), withBody)
	noBody := httptest.NewRequest(http.MethodGet, "/size-test-empty", nil)
	Metrics(next).ServeHTTP(httptest.NewRecorder(), noBody)

	if got := testutil.CollectAndCount(httpRequestSize, "http_request_size_bytes"); got < 1 {
		t.Fatalf("request size series count = %d, want >= 1", got)
	}
	// The GET path must not have created a series; probe by checking that
	// observing it now increases the series count (i.e. it did not exist).
	before := testutil.CollectAndCount(httpRequestSize, "http_request_size_bytes")
	httpRequestSize.WithLabelValues(http.MethodGet, "/size-test-empty").Observe(0)
	after := testutil.CollectAndCount(httpRequestSize, "http_request_size_bytes")
	if after == before {
		t.Error("bodyless GET created a request-size series; ContentLength > 0 gate is broken")
	}
}

// TestMetrics_SlowRequestLogged pins the slow-request warning to its
// boundary: a request is slow only when it takes longer than the p95 budget,
// not when it takes exactly the budget. The clock is injected (#138) so the
// boundary rows are exact; with wall-clock sleeps a `>` to `>=` mutant
// survived (ADR-032).
func TestMetrics_SlowRequestLogged(t *testing.T) {
	budget := performance.MaxP95ResponseTime
	tests := []struct {
		name    string
		took    time.Duration
		wantLog bool
	}{
		{"instant request is not slow", 0, false},
		{"one nanosecond under the budget is not slow", budget - time.Nanosecond, false},
		{"exactly the budget is not slow", budget, false},
		{"one nanosecond over the budget is slow", budget + time.Nanosecond, true},
		{"well over the budget is slow", budget + 20*time.Millisecond, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			// A clock the handler moves: the request takes exactly tt.took.
			clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
			prevNow := now
			now = func() time.Time { return clock }
			t.Cleanup(func() { now = prevNow })

			req := httptest.NewRequest(http.MethodGet, "/slow-log", nil)
			Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				clock = clock.Add(tt.took)
			})).ServeHTTP(httptest.NewRecorder(), req)

			if got := strings.Contains(buf.String(), "slow request detected"); got != tt.wantLog {
				t.Errorf("took %v against a %v budget: slow-request warning logged = %v, want %v", tt.took, budget, got, tt.wantLog)
			}
		})
	}
}

// TestMetrics_ServerTiming pins the Server-Timing header (ADR-034): the
// handler's time to first byte, committed with the headers whether the
// handler calls WriteHeader explicitly or writes the body straight away.
func TestMetrics_ServerTiming(t *testing.T) {
	tests := []struct {
		name string
		next http.HandlerFunc
	}{
		{
			name: "explicit WriteHeader",
			next: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("ok"))
			},
		},
		{
			name: "implicit header via Write",
			next: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("ok"))
			},
		},
	}
	re := regexp.MustCompile(`^app;dur=\d+(\.\d+)?$`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Metrics(tt.next).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/timing", nil))
			got := w.Header().Get("Server-Timing")
			if !re.MatchString(got) {
				t.Errorf("Server-Timing = %q, want app;dur=<ms>", got)
			}
		})
	}
}

// TestMetrics_RecordsLatencyForObserver pins the feed for the landing page's
// observed column: every request through the middleware lands one sample
// in the process observer.
func TestMetrics_RecordsLatencyForObserver(t *testing.T) {
	before := performance.Default.Snapshot().Samples
	w := httptest.NewRecorder()
	Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/observed", nil))
	if after := performance.Default.Snapshot().Samples; after != before+1 {
		t.Errorf("observer samples = %d, want %d (one per request)", after, before+1)
	}
}

// TestMetrics_ServerTimingValue pins the number in Server-Timing, not only
// its shape: milliseconds to one decimal, measured to the moment the headers
// are committed. Uses the injected clock (#138); with the wall clock a
// mutant that multiplied instead of dividing survived (ADR-032).
func TestMetrics_ServerTimingValue(t *testing.T) {
	tests := []struct {
		name         string
		beforeHeader time.Duration
		afterHeader  time.Duration
		want         string
	}{
		{"instant", 0, 0, "app;dur=0.0"},
		{"whole milliseconds", 12 * time.Millisecond, 0, "app;dur=12.0"},
		{"fraction of a millisecond", 12500 * time.Microsecond, 0, "app;dur=12.5"},
		{"time spent after the headers is not counted", 3 * time.Millisecond, 40 * time.Millisecond, "app;dur=3.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
			prevNow := now
			now = func() time.Time { return clock }
			t.Cleanup(func() { now = prevNow })

			w := httptest.NewRecorder()
			Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				clock = clock.Add(tt.beforeHeader)
				w.WriteHeader(http.StatusOK)
				clock = clock.Add(tt.afterHeader)
			})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/timing-value", nil))

			if got := w.Header().Get("Server-Timing"); got != tt.want {
				t.Errorf("Server-Timing = %q, want %q", got, tt.want)
			}
		})
	}
}
