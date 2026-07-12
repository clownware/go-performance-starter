package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// newPatternsRouter mounts the showcase routes exactly as production will, so
// the tests encode the route shape itself, not just handler behaviour.
func newPatternsRouter() http.Handler {
	r := chi.NewRouter()
	PatternsRoutes(r)
	return r
}

// patternSlugs is the catalogue retained from ux-overhaul-spec.md — every
// pattern the showcase must demonstrate, each anchored as a section on the
// page (id={slug}) so patterns are individually linkable.
var patternSlugs = []string{
	"partial-swap",
	"live-search",
	"click-to-edit",
	"inline-validation",
	"optimistic-ui",
	"infinite-scroll",
	"typeahead",
	"toasts",
	"dark-mode",
	"skeleton-loading",
	"tabs",
	"bulk-operations",
	// wave 2: the rest of the bells and whistles
	"polling",
	"oob-swap",
	"confirm-delete",
	"view-transitions",
	"loading-states",
	"modal",
	"global-store",
}

func TestPatternsPage(t *testing.T) {
	// The showcase is public and DB-free (ADR-024): no auth context, no
	// repository — the page must still render every pattern section.
	req := httptest.NewRequest(http.MethodGet, "/patterns", nil)
	w := httptest.NewRecorder()

	newPatternsRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /patterns status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<!doctype") && !strings.Contains(body, "<!DOCTYPE") {
		t.Error("GET /patterns should render a full page through the base layout")
	}
	for _, slug := range patternSlugs {
		if !strings.Contains(body, `id="`+slug+`"`) {
			t.Errorf("GET /patterns missing section anchor id=%q", slug)
		}
	}
}

func TestPatternsAPI(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		target        string
		wantStatus    int
		wantContains  []string
		wantHXTrigger bool // response must carry an HX-Trigger header
	}{
		{
			name:       "partial swap returns fragment",
			method:     http.MethodGet,
			target:     "/patterns/api/swap",
			wantStatus: http.StatusOK,
		},
		{
			name:       "live search returns results",
			method:     http.MethodGet,
			target:     "/patterns/api/search?q=a",
			wantStatus: http.StatusOK,
		},
		{
			name:         "click-to-edit fetch returns a form",
			method:       http.MethodGet,
			target:       "/patterns/api/edit/1",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<form"},
		},
		{
			name:       "click-to-edit save returns display view",
			method:     http.MethodPut,
			target:     "/patterns/api/edit/1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "optimistic favorite toggle",
			method:     http.MethodPost,
			target:     "/patterns/api/favorite/1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "infinite scroll page",
			method:     http.MethodGet,
			target:     "/patterns/api/scroll?page=2",
			wantStatus: http.StatusOK,
		},
		{
			name:       "typeahead results",
			method:     http.MethodGet,
			target:     "/patterns/api/typeahead?q=a",
			wantStatus: http.StatusOK,
		},
		{
			name:          "toast is triggered via response header",
			method:        http.MethodPost,
			target:        "/patterns/api/toast",
			wantStatus:    http.StatusOK,
			wantHXTrigger: true,
		},
		{
			name:       "toast rejects GET",
			method:     http.MethodGet,
			target:     "/patterns/api/toast",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "skeleton demo content",
			method:     http.MethodGet,
			target:     "/patterns/api/skeleton",
			wantStatus: http.StatusOK,
		},
		{
			name:       "server-loaded tab content",
			method:     http.MethodGet,
			target:     "/patterns/api/tab/templ",
			wantStatus: http.StatusOK,
		},
		{
			name:       "bulk operation result",
			method:     http.MethodPost,
			target:     "/patterns/api/bulk",
			wantStatus: http.StatusOK,
		},
		{
			name:       "polling tick",
			method:     http.MethodGet,
			target:     "/patterns/api/time",
			wantStatus: http.StatusOK,
		},
		{
			name:         "counter response carries an out-of-band swap",
			method:       http.MethodPost,
			target:       "/patterns/api/counter",
			wantStatus:   http.StatusOK,
			wantContains: []string{`hx-swap-oob`},
		},
		{
			name:       "confirmed delete returns the emptied state",
			method:     http.MethodPost,
			target:     "/patterns/api/confirm",
			wantStatus: http.StatusOK,
		},
		{
			name:       "view-transition card",
			method:     http.MethodGet,
			target:     "/patterns/api/transition?step=1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "loading-states content (delay capped server-side)",
			method:     http.MethodGet,
			target:     "/patterns/api/slow?delay=0",
			wantStatus: http.StatusOK,
		},
	}

	router := newPatternsRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			req.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.target, w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				body := w.Body.String()
				if len(strings.TrimSpace(body)) == 0 {
					t.Errorf("%s %s rendered an empty fragment", tt.method, tt.target)
				}
				if strings.Contains(body, "<!doctype") || strings.Contains(body, "<!DOCTYPE") {
					t.Errorf("%s %s is a demo endpoint and must return a fragment, not a full page", tt.method, tt.target)
				}
				for _, want := range tt.wantContains {
					if !strings.Contains(body, want) {
						t.Errorf("%s %s body missing %q", tt.method, tt.target, want)
					}
				}
			}
			if tt.wantHXTrigger && w.Header().Get("HX-Trigger") == "" {
				t.Errorf("%s %s must set an HX-Trigger header to fire the toast", tt.method, tt.target)
			}
		})
	}
}
