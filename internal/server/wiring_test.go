package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clownware/go-performance-starter/internal/config"
)

// Wiring contracts for internal/server (#88). These tests assert what the
// router does — middleware order as observable behaviour, static cache
// headers, the registered route table — rather than line coverage. No
// production code is introspected beyond chi.Walk over the route tree.

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testAnonKey is a placeholder Supabase anon key for tests — not a secret.
const testAnonKey = "test-anon-key"

// fakeGoTrue is a minimal stand-in for the Supabase GoTrue REST API. It
// records every request it receives (method, path, Authorization) so tests
// can assert which middleware talked to it and in what order. Anonymous
// sign-up succeeds; token validation (GET /user) always fails, so no test
// path ever needs a users row from the database.
type fakeGoTrue struct {
	*httptest.Server
	mu       sync.Mutex
	requests []fakeGoTrueCall
}

type fakeGoTrueCall struct {
	Method, Path, Authorization string
}

func newFakeGoTrue(t *testing.T) *fakeGoTrue {
	t.Helper()
	f := &fakeGoTrue{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, fakeGoTrueCall{r.Method, r.URL.Path, r.Header.Get("Authorization")})
		f.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/signup":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"guest-access-token","refresh_token":"guest-refresh-token",` +
				`"expires_in":3600,"user":{"id":"00000000-0000-0000-0000-000000000001","is_anonymous":true}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/auth/v1/user":
			http.Error(w, `{"msg":"invalid token"}`, http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGoTrue) calls() []fakeGoTrueCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeGoTrueCall(nil), f.requests...)
}

// authTestConfig returns a development config with Supabase auth enabled
// against the fake GoTrue. New() only constructs the client — no network
// call happens until a request reaches an auth-aware middleware/handler.
func authTestConfig(gotrueURL string) *config.Config {
	cfg := testConfig("development")
	cfg.SupabaseURL = gotrueURL
	cfg.SupabaseAnonKey = testAnonKey
	return cfg
}

// deadPool returns a pgxpool that never dials: pgxpool.New is lazy (no
// MinConns), so New() can register the DB-gated route groups without a
// database. Tests using it must not send requests that reach a repository.
func deadPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:1/test?sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New() error: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// routeSet walks the router tree and returns "METHOD /pattern" keys.
func routeSet(t *testing.T, r chi.Routes) map[string]bool {
	t.Helper()
	set := map[string]bool{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		set[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk() error: %v", err)
	}
	return set
}

// syncBuffer guards the log buffer: slog handlers may be invoked from any
// goroutine, and the race detector is on.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLogs routes slog's default logger into a buffer for the test's
// lifetime and returns a finder for the first record with the given msg
// (e.g. RequestLogger's "request completed").
func captureLogs(t *testing.T) func(msg string) map[string]any {
	t.Helper()
	buf := &syncBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return func(msg string) map[string]any {
		t.Helper()
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			var rec map[string]any
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatalf("log line is not JSON: %q (%v)", line, err)
			}
			if rec["msg"] == msg {
				return rec
			}
		}
		t.Fatalf("no %q log record found; log output:\n%s", msg, buf.String())
		return nil
	}
}

func doRequest(srv http.Handler, method, target string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Static asset helpers: isFileType / cacheControlWrapper / fileServer
// ---------------------------------------------------------------------------

func TestIsFileType(t *testing.T) {
	tests := []struct {
		name string
		path string
		exts []string
		want bool
	}{
		{"matches listed extension", "/static/css/app.css", []string{".css", ".js"}, true},
		{"match is case-insensitive", "/static/img/LOGO.PNG", []string{".png"}, true},
		{"only the final extension counts", "/static/js/htmx.min.js", []string{".js"}, true},
		{"non-matching extension", "/static/fonts/inter.woff2", []string{".css", ".js"}, false},
		{"no extension never matches a real extension", "/static/README", []string{".css", ".js"}, false},
		{"dot in directory is not an extension", "/static/v1.2/app", []string{".2"}, false},
		{"empty extension list never matches", "/static/css/app.css", nil, false},
		{"empty path never matches", "", []string{".css"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFileType(tt.path, tt.exts...); got != tt.want {
				t.Errorf("isFileType(%q, %v) = %v, want %v", tt.path, tt.exts, got, tt.want)
			}
		})
	}
}

func TestCacheControlWrapper(t *testing.T) {
	const (
		oneYear = "public, max-age=31536000"
		oneHour = "public, max-age=3600"
	)
	tests := []struct {
		name string
		path string
		want string
	}{
		{"css cached one year", "/css/app.css", oneYear},
		{"js cached one year", "/js/app.js", oneYear},
		{"png cached one year", "/img/logo.png", oneYear},
		{"jpg cached one year", "/img/photo.jpg", oneYear},
		{"jpeg cached one year", "/img/photo.jpeg", oneYear},
		{"gif cached one year", "/img/anim.gif", oneYear},
		{"svg cached one year", "/img/mark.svg", oneYear},
		{"webp cached one year", "/img/hero.webp", oneYear},
		{"woff cached one year", "/fonts/inter.woff", oneYear},
		{"woff2 cached one year", "/fonts/inter.woff2", oneYear},
		{"ttf cached one year", "/fonts/inter.ttf", oneYear},
		{"otf cached one year", "/fonts/inter.otf", oneYear},
		{"eot cached one year", "/fonts/inter.eot", oneYear},
		{"uppercase extension still long-lived", "/img/LOGO.SVG", oneYear},
		{"html cached one hour", "/pages/offline.html", oneHour},
		{"json cached one hour", "/manifest.json", oneHour},
		{"no extension cached one hour", "/robots", oneHour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			rec := httptest.NewRecorder()
			cacheControlWrapper(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if !called {
				t.Fatal("wrapped handler was not invoked")
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d (wrapper must not alter the handler's status)", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.want {
				t.Errorf("Cache-Control for %s = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestCacheControlWrapper_ErrorsNotCached pins the 2026-08-19 found-work
// fix: a 404 for /static/app.css used to carry the one-year header, so an
// intermediary could cache the miss and keep serving it after the file
// appeared. Error responses now go out as no-store; 304 revalidations keep
// the extension header (they're the cache working as intended); implicit
// WriteHeader via Write still succeeds.
func TestCacheControlWrapper_ErrorsNotCached(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		status    int
		wantCache string
	}{
		{"404 for a css path is no-store, not one-year", "/css/missing.css", http.StatusNotFound, "no-store"},
		{"500 for a font path is no-store", "/fonts/inter.woff2", http.StatusInternalServerError, "no-store"},
		{"403 for an html path is no-store", "/secret.html", http.StatusForbidden, "no-store"},
		{"304 revalidation keeps the one-year header", "/css/app.css", http.StatusNotModified, "public, max-age=31536000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})
			rec := httptest.NewRecorder()
			cacheControlWrapper(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
		})
	}

	t.Run("implicit WriteHeader via Write keeps the success header", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("body{}")) // no explicit WriteHeader
		})
		rec := httptest.NewRecorder()
		cacheControlWrapper(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/css/app.css", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000" {
			t.Errorf("Cache-Control = %q, want the one-year header", got)
		}
		if rec.Body.String() != "body{}" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "body{}")
		}
	})
}

// TestFileServer mounts fileServer on a bare chi router over a temp dir and
// checks the mount contract: the bare prefix redirects to the trailing-slash
// form, files under it are served with the extension-based cache header.
func TestFileServer(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("css/app.css", "body{margin:0}")
	mustWrite("fonts/inter.woff2", "wOF2")
	mustWrite("offline.html", "<h1>offline</h1>")

	r := chi.NewRouter()
	fileServer(r, "/static", http.Dir(dir))

	tests := []struct {
		name         string
		target       string
		wantStatus   int
		wantLocation string
		wantCache    string
		wantBody     string
	}{
		{"bare prefix redirects to slash form", "/static", http.StatusMovedPermanently, "/static/", "", ""},
		{"css served with one-year cache", "/static/css/app.css", http.StatusOK, "", "public, max-age=31536000", "body{margin:0}"},
		{"font served with one-year cache", "/static/fonts/inter.woff2", http.StatusOK, "", "public, max-age=31536000", "wOF2"},
		{"html served with one-hour cache", "/static/offline.html", http.StatusOK, "", "public, max-age=3600", "<h1>offline</h1>"},
		{"missing file is 404 and not cacheable", "/static/css/missing.css", http.StatusNotFound, "", "no-store", ""},
		{"path outside the mount is not served", "/css/app.css", http.StatusNotFound, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(r, http.MethodGet, tt.target, nil)
			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.target, rec.Code, tt.wantStatus)
			}
			if tt.wantLocation != "" {
				if got := rec.Header().Get("Location"); got != tt.wantLocation {
					t.Errorf("Location = %q, want %q", got, tt.wantLocation)
				}
			}
			if tt.wantCache != "" {
				if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
					t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
				}
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestServer_StaticMount proves the real router serves ./web/static at
// /static with the cache header AND the global security headers — the
// static mount is registered inside setupMiddleware, so it must still sit
// behind the header middleware.
func TestServer_StaticMount(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "web", "static", "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "web", "static", "css", "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // http.Dir("./web/static") resolves against the CWD at Open time
	srv := newTestServer(t, "development")

	tests := []struct {
		name         string
		target       string
		wantStatus   int
		wantLocation string
		wantCache    string
	}{
		{"bare /static redirects", "/static", http.StatusMovedPermanently, "/static/", ""},
		{"asset served with one-year cache", "/static/css/app.css", http.StatusOK, "", "public, max-age=31536000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(srv, http.MethodGet, tt.target, nil)
			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.target, rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("static response missing security headers: X-Content-Type-Options = %q", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Route table
// ---------------------------------------------------------------------------

// TestServer_RouteTable walks the router and pins the registered
// method+pattern set per configuration: the public surface is always there,
// the /auth credential endpoints appear only with a Supabase client, and the
// /profile, /first-run and /learn groups additionally require a DB pool.
func TestServer_RouteTable(t *testing.T) {
	public := []string{
		"GET /",
		"GET /dashboard",
		"GET /terms",
		"GET /privacy",
		"GET /healthz",
		"GET /health",
		"GET /metrics",
		"GET /auth/page",
		"GET /auth/logout",
		"GET /static",
		"GET /static/*",
		// handler.PatternsRoutes (ADR-024 surface 2)
		"GET /patterns",
		"GET /patterns/api/swap",
		"GET /patterns/api/search",
		"GET /patterns/api/edit/{id}",
		"PUT /patterns/api/edit/{id}",
		"POST /patterns/api/favorite/{id}",
		"GET /patterns/api/scroll",
		"GET /patterns/api/typeahead",
		"POST /patterns/api/toast",
		"GET /patterns/api/skeleton",
		"GET /patterns/api/tab/{name}",
		"POST /patterns/api/bulk",
		"GET /patterns/api/time",
		"POST /patterns/api/counter",
		"POST /patterns/api/confirm",
		"GET /patterns/api/transition",
		"GET /patterns/api/slow",
	}
	authRoutes := []string{
		"GET /auth/recover",
		"GET /auth/reset",
		"POST /auth/login",
		"POST /auth/signup",
		"POST /auth/recover",
		"POST /auth/reset",
		"POST /auth/logout",
	}
	dbGatedRoutes := []string{
		"GET /profile",
		"POST /profile",
		// handler.FirstRunHandlers
		"GET /first-run",
		"GET /first-run/profile",
		"GET /first-run/ctas",
		// handler.QuizRoutes / FlashcardRoutes / UpgradeRoutes (ADR-024 surface 3)
		"GET /learn/quiz",
		"POST /learn/quiz/{slug}/answer",
		"GET /learn/flashcards",
		"POST /learn/flashcards",
		"POST /learn/flashcards/{id}/known",
		"POST /learn/flashcards/{id}/delete",
		"GET /learn/upgrade",
		"POST /learn/upgrade",
	}
	// Retired pre-ADR-024 stubs must never come back (see TestServer_StubDemosRetired).
	retired := []string{"GET /items", "GET /items/list", "GET /api/users/{id}", "GET /api/organizations", "GET /api"}

	tests := []struct {
		name        string
		newServer   func(t *testing.T) *Server
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "auth disabled, no DB",
			newServer:   func(t *testing.T) *Server { return newTestServer(t, "development") },
			wantPresent: public,
			wantAbsent:  concat(authRoutes, dbGatedRoutes, retired),
		},
		{
			name: "auth enabled, no DB",
			newServer: func(t *testing.T) *Server {
				return newServer(t, authTestConfig(newFakeGoTrue(t).URL), nil)
			},
			wantPresent: concat(public, authRoutes),
			wantAbsent:  concat(dbGatedRoutes, retired),
		},
		{
			name: "auth enabled with DB",
			newServer: func(t *testing.T) *Server {
				return newServer(t, authTestConfig(newFakeGoTrue(t).URL), deadPool(t))
			},
			wantPresent: concat(public, authRoutes, dbGatedRoutes),
			wantAbsent:  retired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := routeSet(t, tt.newServer(t).router)
			for _, want := range tt.wantPresent {
				if !routes[want] {
					t.Errorf("route %q not registered", want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if routes[absent] {
					t.Errorf("route %q registered but must not be in this configuration", absent)
				}
			}
			if t.Failed() {
				t.Logf("registered routes:\n  %s", strings.Join(sortedKeys(routes), "\n  "))
			}
		})
	}
}

func concat(lists ...[]string) []string {
	var out []string
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Middleware wiring as observable behaviour
// ---------------------------------------------------------------------------

// TestServer_SecurityHeadersWired proves SecurityHeaders is the outermost
// layer: every response carries the ADR-014 headers, including 404s for
// unmatched paths (the middleware runs before routing resolves).
func TestServer_SecurityHeadersWired(t *testing.T) {
	srv := newTestServer(t, "development")
	wantHeaders := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":       "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{"liveness probe", "/healthz", http.StatusOK},
		{"rendered page", "/", http.StatusOK},
		{"unmatched path", "/does-not-exist", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(srv, http.MethodGet, tt.target, nil)
			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.target, rec.Code, tt.wantStatus)
			}
			for name, want := range wantHeaders {
				if got := rec.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if csp := rec.Header().Get("Content-Security-Policy"); !strings.HasPrefix(csp, "default-src 'self';") {
				t.Errorf("Content-Security-Policy = %q, want a default-src 'self' policy", csp)
			}
		})
	}
}

// TestServer_RequestLogCarriesIdentity pins two orderings through the only
// place they are observable — RequestLogger's structured record:
//
//   - RequestID runs before RequestLogger: every request logs a request_id
//     (generated, or the inbound X-Request-ID when one is supplied).
//   - RealIP runs before RequestLogger: remote_addr is the resolved client
//     IP — port stripped, and forwarded headers honoured only from a peer
//     inside TrustedProxyCIDRs (ADR-027).
func TestServer_RequestLogCarriesIdentity(t *testing.T) {
	const forwarded = "203.0.113.9"
	tests := []struct {
		name           string
		trustedCIDRs   []string
		headers        map[string]string
		wantRemoteAddr string
		wantRequestID  string // "" means "any non-empty"
	}{
		{
			name:           "request id generated when absent",
			wantRemoteAddr: "192.0.2.1", // httptest.NewRequest's RemoteAddr, port stripped
		},
		{
			name:           "inbound X-Request-ID is carried through",
			headers:        map[string]string{"X-Request-ID": "req-from-edge-1"},
			wantRemoteAddr: "192.0.2.1",
			wantRequestID:  "req-from-edge-1",
		},
		{
			name:           "forwarded header ignored from untrusted peer",
			headers:        map[string]string{"X-Forwarded-For": forwarded},
			wantRemoteAddr: "192.0.2.1",
		},
		{
			name:           "forwarded client resolved behind trusted proxy",
			trustedCIDRs:   []string{"192.0.2.0/24"},
			headers:        map[string]string{"X-Forwarded-For": forwarded},
			wantRemoteAddr: forwarded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig("development")
			cfg.TrustedProxyCIDRs = tt.trustedCIDRs
			srv := newServer(t, cfg, nil)
			record := captureLogs(t)

			rec := doRequest(srv, http.MethodGet, "/healthz", func(r *http.Request) {
				for k, v := range tt.headers {
					r.Header.Set(k, v)
				}
			})
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /healthz status = %d, want 200", rec.Code)
			}

			got := record("request completed")
			if got["remote_addr"] != tt.wantRemoteAddr {
				t.Errorf("logged remote_addr = %v, want %q", got["remote_addr"], tt.wantRemoteAddr)
			}
			id, _ := got["request_id"].(string)
			switch {
			case tt.wantRequestID != "" && id != tt.wantRequestID:
				t.Errorf("logged request_id = %q, want %q", id, tt.wantRequestID)
			case tt.wantRequestID == "" && id == "":
				t.Error("logged request_id is empty — RequestID middleware not wired before RequestLogger")
			}
		})
	}
}

// TestServer_RateLimiterKeysOnResolvedClientIP proves RealIP runs before the
// global RateLimiter: behind a trusted proxy, distinct forwarded clients get
// independent buckets (no 429 across many requests), while one forwarded
// client exhausts its own burst. If the limiter ran first it would key on the
// proxy's address and the distinct-client case would start returning 429.
func TestServer_RateLimiterKeysOnResolvedClientIP(t *testing.T) {
	const requests = 40 // > burst(10); at 50 rps all 40 cannot refill in the time this takes
	tests := []struct {
		name       string
		forwarded  func(i int) string
		wantLimits bool
	}{
		{"same forwarded client exhausts its bucket", func(int) string { return "203.0.113.9" }, true},
		{"distinct forwarded clients are limited independently", func(i int) string { return "203.0.113." + strconv.Itoa(i+1) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig("development")
			cfg.TrustedProxyCIDRs = []string{"192.0.2.0/24"}
			srv := newServer(t, cfg, nil)

			limited := 0
			for i := 0; i < requests; i++ {
				rec := doRequest(srv, http.MethodGet, "/healthz", func(r *http.Request) {
					r.Header.Set("X-Forwarded-For", tt.forwarded(i))
				})
				switch rec.Code {
				case http.StatusOK:
				case http.StatusTooManyRequests:
					limited++
				default:
					t.Fatalf("request %d: status = %d, want 200 or 429", i, rec.Code)
				}
			}
			if tt.wantLimits && limited == 0 {
				t.Errorf("no 429 in %d requests from one client — global rate limiter not wired", requests)
			}
			if !tt.wantLimits && limited > 0 {
				t.Errorf("%d of %d requests from distinct clients were 429 — limiter keys on the proxy, not the RealIP-resolved client", limited, requests)
			}
		})
	}
}

// TestServer_CompressWired proves response compression is negotiated on a
// compressible (text/html) route and absent when the client does not ask.
func TestServer_CompressWired(t *testing.T) {
	srv := newTestServer(t, "development")
	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
	}{
		{"gzip negotiated", "gzip", "gzip"},
		{"identity when not requested", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(srv, http.MethodGet, "/", func(r *http.Request) {
				if tt.acceptEncoding != "" {
					r.Header.Set("Accept-Encoding", tt.acceptEncoding)
				}
			})
			if rec.Code != http.StatusOK {
				t.Fatalf("GET / status = %d, want 200", rec.Code)
			}
			if got := rec.Header().Get("Content-Encoding"); got != tt.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, tt.wantEncoding)
			}

			body := rec.Body.Bytes()
			if tt.wantEncoding == "gzip" {
				if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
					t.Errorf("Vary = %q, want it to include Accept-Encoding", rec.Header().Get("Vary"))
				}
				zr, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Fatalf("body is not valid gzip: %v", err)
				}
				if body, err = io.ReadAll(zr); err != nil {
					t.Fatalf("gunzip: %v", err)
				}
			}
			if !strings.Contains(string(body), "Go Performance Starter") {
				t.Error("decoded body does not contain the page brand — compression corrupted or wrong route")
			}
		})
	}
}

// TestServer_CSRFGatesUnsafeMethods extends TestServer_CSRFWiring: the
// double-submit check sits before routing AND before every handler, so an
// unsafe method without a token is rejected on any route — including the
// auth-enabled credential endpoints, which must never reach GoTrue without
// it.
func TestServer_CSRFGatesUnsafeMethods(t *testing.T) {
	tests := []struct {
		name   string
		auth   bool
		method string
		target string
	}{
		{"PUT on a patterns route", false, http.MethodPut, "/patterns/api/edit/1"},
		{"POST on a patterns route", false, http.MethodPost, "/patterns/api/favorite/1"},
		{"DELETE on an unregistered path is refused before routing", false, http.MethodDelete, "/does-not-exist"},
		{"POST /auth/login never reaches the credential handler", true, http.MethodPost, "/auth/login"},
		{"POST /auth/signup never reaches the credential handler", true, http.MethodPost, "/auth/signup"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv *Server
			var gotrue *fakeGoTrue
			if tt.auth {
				gotrue = newFakeGoTrue(t)
				srv = newServer(t, authTestConfig(gotrue.URL), nil)
			} else {
				srv = newTestServer(t, "development")
			}

			rec := doRequest(srv, tt.method, tt.target, nil)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s %s without CSRF token: status = %d, want 403", tt.method, tt.target, rec.Code)
			}
			if gotrue != nil {
				if calls := gotrue.calls(); len(calls) != 0 {
					t.Errorf("GoTrue received %d request(s) for a CSRF-rejected POST: %+v", len(calls), calls)
				}
			}
		})
	}
}

// TestServer_GuestSessionBeforeOptionalAuth pins the /learn identity chain
// order (ADR-024): GuestSession mints an anonymous session first, then
// OptionalAuth validates the token that GuestSession just injected into the
// same request. Observable proof: GoTrue sees POST /signup followed by
// GET /user carrying the freshly issued bearer token. The fake rejects that
// validation, so OptionalAuth clears the cookies and the page still renders
// the browse-first teaser — never touching the (dead) DB.
func TestServer_GuestSessionBeforeOptionalAuth(t *testing.T) {
	gotrue := newFakeGoTrue(t)
	cfg := authTestConfig(gotrue.URL)
	cfg.GuestModeEnabled = true
	srv := newServer(t, cfg, deadPool(t))

	rec := doRequest(srv, http.MethodGet, "/learn/quiz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /learn/quiz status = %d, want 200; body:\n%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `data-testid="quiz-teaser"`) {
		t.Error("anonymous GET /learn/quiz did not render the browse-first teaser")
	}

	want := []fakeGoTrueCall{
		{http.MethodPost, "/auth/v1/signup", "Bearer " + testAnonKey},
		{http.MethodGet, "/auth/v1/user", "Bearer guest-access-token"},
	}
	got := gotrue.calls()
	if len(got) != len(want) {
		t.Fatalf("GoTrue calls = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GoTrue call %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// GuestSession issued the cookie; OptionalAuth then cleared it on failed
	// validation. Both Set-Cookie headers prove the order within one request.
	var issued, cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name != "sb-access-token" {
			continue
		}
		switch {
		case c.Value == "guest-access-token" && c.MaxAge > 0:
			issued = true
		case c.Value == "" && c.MaxAge < 0:
			cleared = issued // only counts if it came after the issue
		}
	}
	if !issued {
		t.Error("GuestSession did not issue the sb-access-token cookie")
	}
	if !cleared {
		t.Error("OptionalAuth did not clear the rejected guest cookie after GuestSession issued it")
	}
}

// ---------------------------------------------------------------------------
// Server surface: ServeHTTP delegation, AuthClient accessor
// ---------------------------------------------------------------------------

var _ http.Handler = (*Server)(nil)

// TestServer_ServeHTTPDelegates proves the http.Handler implementation is a
// pure pass-through to the router.
func TestServer_ServeHTTPDelegates(t *testing.T) {
	srv := newTestServer(t, "development")
	tests := []struct {
		name   string
		target string
	}{
		{"matched route", "/healthz"},
		{"unmatched route", "/does-not-exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viaServer := doRequest(srv, http.MethodGet, tt.target, nil)
			viaRouter := doRequest(srv.router, http.MethodGet, tt.target, nil)
			if viaServer.Code != viaRouter.Code {
				t.Errorf("Server status = %d, router status = %d", viaServer.Code, viaRouter.Code)
			}
			if viaServer.Body.String() != viaRouter.Body.String() {
				t.Errorf("Server body differs from router body for %s", tt.target)
			}
		})
	}
}

// failingWriter simulates a client that disconnects mid-render: headers are
// accepted, every body write fails.
type failingWriter struct {
	http.ResponseWriter
}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestHomeHandler_RenderFailureIsLogged pins the error path of the home page:
// a write failure after headers are committed is logged, not panicked or
// silently dropped.
func TestHomeHandler_RenderFailureIsLogged(t *testing.T) {
	record := captureLogs(t)
	rec := httptest.NewRecorder()

	homeHandler(failingWriter{rec}, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (headers are committed before the body streams)", rec.Code)
	}
	entry := record("Failed to render home page")
	if entry["level"] != "ERROR" {
		t.Errorf("render failure logged at %v, want ERROR", entry["level"])
	}
}

func TestServer_AuthClient(t *testing.T) {
	tests := []struct {
		name           string
		cfg            func(t *testing.T) *config.Config
		wantClient     bool
		wantServiceKey bool
	}{
		{
			name:       "nil when Supabase credentials are absent",
			cfg:        func(*testing.T) *config.Config { return testConfig("development") },
			wantClient: false,
		},
		{
			name:       "constructed from URL and anon key without a service role key",
			cfg:        func(t *testing.T) *config.Config { return authTestConfig(newFakeGoTrue(t).URL) },
			wantClient: true,
		},
		{
			name: "service role key attached when configured",
			cfg: func(t *testing.T) *config.Config {
				cfg := authTestConfig(newFakeGoTrue(t).URL)
				cfg.SupabaseServiceRoleKey = "test-service-role-key"
				return cfg
			},
			wantClient:     true,
			wantServiceKey: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(t, tt.cfg(t), nil)
			client := srv.AuthClient()
			if (client != nil) != tt.wantClient {
				t.Fatalf("AuthClient() = %v, want non-nil=%v", client, tt.wantClient)
			}
			if client != nil && client.HasServiceRoleKey() != tt.wantServiceKey {
				t.Errorf("HasServiceRoleKey() = %v, want %v", client.HasServiceRoleKey(), tt.wantServiceKey)
			}
		})
	}
}
