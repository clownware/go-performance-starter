package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
	"github.com/clownware/go-performance-starter/internal/view/partials"
)

// /patterns is ADR-024 surface 2: the public HTMX/Alpine pattern showcase.
// Self-contained stub data, no database, no auth — the page must render with
// Supabase credentials unset. Each section is a demo panel plus tabbed
// (templ | handler) source, one per pattern in the catalogue retained from
// docs/updates/ux-overhaul-spec.md.

// PatternsRoutes registers the /patterns showcase page and its stub demo API.
func PatternsRoutes(r chi.Router) {
	r.Get("/patterns", PatternsPage)
	r.Route("/patterns/api", func(api chi.Router) {
		api.Get("/swap", PatternSwap)
		api.Get("/search", PatternSearch)
		api.Get("/edit/{id}", PatternEditForm)
		api.Put("/edit/{id}", PatternEditSave)
		api.Post("/favorite/{id}", PatternFavorite)
		api.Get("/scroll", PatternScroll)
		api.Get("/typeahead", PatternTypeahead)
		api.Post("/toast", PatternToast)
		api.Get("/skeleton", PatternSkeleton)
		api.Get("/tab/{name}", PatternTab)
		api.Post("/bulk", PatternBulk)
	})
}

// patternStubData is the shared dataset the search, typeahead, and scroll
// demos filter and page over — the pieces of this starter's own stack.
var patternStubData = []string{
	"Go 1.26", "Chi router", "templ", "HTMX", "Alpine.js", "Tailwind CSS",
	"PostgreSQL", "sqlc", "pgx", "Supabase Auth", "Row Level Security",
	"golang-migrate", "Taskfile", "golangci-lint", "Prometheus", "Fly.io",
	"Docker", "GitHub Actions", "air (live reload)", "gofmt",
}

const patternScrollPerPage = 5

// patternsCatalog drives the showcase sections. Source snippets are abridged
// from the real handlers and templates below (kept in sync by hand — see
// ux-overhaul-spec.md for the generated-source future iteration).
var patternsCatalog = []view.PatternSection{
	{
		Slug:         "partial-swap",
		Title:        "Partial swap",
		Summary:      "Click a button, replace a fragment. The simplest HTMX pattern — a GET returns rendered HTML and hx-target places it.",
		HTMXFeatures: []string{"hx-get", "hx-target", "hx-swap"},
		TemplSource: `<button hx-get="/patterns/api/swap"
  hx-target="#swap-demo" hx-swap="innerHTML">
  Swap this fragment
</button>
<div id="swap-demo">...</div>`,
		HandlerSource: `func PatternSwap(w http.ResponseWriter, r *http.Request) {
  view.Render(w, r, http.StatusOK,
    partials.PatternSwapResult())
}`,
	},
	{
		Slug:         "live-search",
		Title:        "Live search",
		Summary:      "Type in a search box; results filter server-side with a debounce, so the server stays the single source of truth.",
		HTMXFeatures: []string{"hx-get", `hx-trigger="keyup changed delay:300ms"`, "hx-target"},
		TemplSource: `<input type="search" name="q"
  hx-get="/patterns/api/search"
  hx-trigger="keyup changed delay:300ms"
  hx-target="#search-results"/>`,
		HandlerSource: `q := r.URL.Query().Get("q")
results := filterStubData(q)
view.Render(w, r, http.StatusOK,
  partials.PatternSearchResults(props))`,
	},
	{
		Slug:         "click-to-edit",
		Title:        "Click to edit",
		Summary:      "Click text to turn it into an edit form; saving swaps the display view back in. Two endpoints, zero client state.",
		HTMXFeatures: []string{"hx-get", "hx-put", "hx-swap"},
		TemplSource: `<div hx-get="/patterns/api/edit/1"
  hx-swap="outerHTML">{ value }</div>
<form hx-put="/patterns/api/edit/1"
  hx-swap="outerHTML">...</form>`,
		HandlerSource: `// GET returns the form; PUT saves and
// returns the display view.
value := r.FormValue("value")
view.Render(w, r, http.StatusOK,
  partials.PatternEditDisplay(props))`,
	},
	{
		Slug:           "inline-validation",
		Title:          "Inline validation",
		Summary:        "Alpine validates as you type; the server re-checks on submit and returns field errors into the same markup.",
		HTMXFeatures:   []string{"hx-post", "hx-target"},
		AlpineFeatures: []string{"x-data", "x-model", "x-show"},
		TemplSource: `<form x-data="{ email: '' }">
  <input type="email" x-model="email"/>
  <p x-show="email && !email.includes('@')">
    That doesn't look like an email.
  </p>
</form>`,
		HandlerSource: `// Server-side mirror of the client rules —
// see ProfileUpdate for the full pattern:
// errors map rendered at 422 into the form.`,
	},
	{
		Slug:           "optimistic-ui",
		Title:          "Optimistic UI",
		Summary:        "A favorite toggle that swaps instantly — current state rides along in hx-vals, so the endpoint stays stateless.",
		HTMXFeatures:   []string{"hx-post", `hx-swap="outerHTML"`, "hx-vals"},
		AlpineFeatures: []string{"x-transition"},
		TemplSource: `<button hx-post="/patterns/api/favorite/1"
  hx-vals={ favoritedState }
  hx-swap="outerHTML" x-transition>
  &#9734; Favorite
</button>`,
		HandlerSource: `favorited := r.FormValue("favorited") == "true"
props := partials.PatternFavoriteProps{
  ID: id, Favorited: !favorited}
view.Render(w, r, http.StatusOK,
  partials.PatternFavoriteButton(props))`,
	},
	{
		Slug:         "infinite-scroll",
		Title:        "Infinite scroll",
		Summary:      "Scroll to the bottom and the next page loads automatically; the sentinel row replaces itself with each new page.",
		HTMXFeatures: []string{"hx-get", `hx-trigger="revealed"`, `hx-swap="outerHTML"`},
		TemplSource: `<li hx-get="/patterns/api/scroll?page=2"
  hx-trigger="revealed" hx-swap="outerHTML">
  Loading more...
</li>`,
		HandlerSource: `page, _ := strconv.Atoi(r.URL.Query().Get("page"))
items, next := stubPage(page)
view.Render(w, r, http.StatusOK,
  partials.PatternScrollPage(props))`,
	},
	{
		Slug:         "typeahead",
		Title:        "Active search (typeahead)",
		Summary:      "Like live search, plus a loading indicator that HTMX toggles for you while the request is in flight.",
		HTMXFeatures: []string{"hx-get", `hx-trigger="keyup changed delay:200ms"`, "hx-indicator"},
		TemplSource: `<input hx-get="/patterns/api/typeahead"
  hx-trigger="keyup changed delay:200ms"
  hx-indicator="#typeahead-indicator"/>
<span id="typeahead-indicator"
  class="htmx-indicator">Searching...</span>`,
		HandlerSource: `// Identical server shape to live search —
// the indicator is pure client wiring.`,
	},
	{
		Slug:           "toasts",
		Title:          "Toast notifications",
		Summary:        "The response body swaps a tiny status line; the toast itself fires from the HX-Trigger header into an Alpine listener.",
		HTMXFeatures:   []string{"HX-Trigger", "HX-Toast-Type"},
		AlpineFeatures: []string{"x-data", "x-for", "x-transition"},
		TemplSource: `<button hx-post="/patterns/api/toast"
  hx-vals={ toastType }>Success toast</button>
<!-- base layout listens for HX-Trigger
     and dispatches a toast event -->`,
		HandlerSource: `view.SetHXTrigger(w, "Toast from the server!")
w.Header().Set("HX-Toast-Type", toastType)
view.Render(w, r, http.StatusOK,
  partials.PatternToastResult())`,
	},
	{
		Slug:           "dark-mode",
		Title:          "Dark mode",
		Summary:        "Alpine state seeded from the system preference, persisted to localStorage, applied as a class on the root element.",
		AlpineFeatures: []string{"x-data", "$watch", "localStorage"},
		TemplSource: `<body x-data="{ dark: localStorage.getItem('dark')
    === 'true' || prefersDark }"
  x-init="$watch('dark', v => ...)">`,
		HandlerSource: `// No handler — this pattern is entirely
// client-side; the server ships one class
// toggle and no JavaScript framework.`,
	},
	{
		Slug:           "skeleton-loading",
		Title:          "Skeleton loading",
		Summary:        "A CSS skeleton renders instantly; HTMX replaces it with real content on a delayed load trigger.",
		HTMXFeatures:   []string{"hx-get", `hx-trigger="load delay:1.5s"`},
		AlpineFeatures: []string{},
		TemplSource: `<div hx-get="/patterns/api/skeleton"
  hx-trigger="load delay:1.5s">
  <div class="animate-pulse">...</div>
</div>`,
		HandlerSource: `// The server returns immediately — the
// delay is client-side, so no goroutine
// sleeps on a demo's behalf.
view.Render(w, r, http.StatusOK,
  partials.PatternSkeletonContent())`,
	},
	{
		Slug:           "tabs",
		Title:          "Tabs",
		Summary:        "Tab buttons fetch server-rendered panels; Alpine handles the client-only variant (see the source panel you're using).",
		HTMXFeatures:   []string{"hx-get", "hx-target"},
		AlpineFeatures: []string{"x-data", "x-show"},
		TemplSource: `<button hx-get="/patterns/api/tab/templ"
  hx-target="#tab-demo-panel">templ</button>
<div id="tab-demo-panel"></div>`,
		HandlerSource: `name := chi.URLParam(r, "name")
content, ok := patternTabs[name]
if !ok { http.NotFound(w, r); return }
view.Render(w, r, http.StatusOK,
  partials.PatternTabPanel(props))`,
	},
	{
		Slug:           "bulk-operations",
		Title:          "Bulk operations",
		Summary:        "Alpine tracks the checkbox selection client-side; one HTMX POST submits the whole batch as form values.",
		HTMXFeatures:   []string{"hx-post", "hx-target"},
		AlpineFeatures: []string{"x-data", "x-model", "x-text"},
		TemplSource: `<form x-data="{ selected: [] }"
  hx-post="/patterns/api/bulk"
  hx-target="#bulk-demo-result">
  <input type="checkbox" name="selected"
    x-model="selected"/>
  <button :disabled="selected.length === 0">
    Archive (<span x-text="selected.length"/>)
  </button>
</form>`,
		HandlerSource: `r.ParseForm()
count := len(r.Form["selected"])
view.Render(w, r, http.StatusOK,
  partials.PatternBulkResult(count))`,
	},
}

// patternTabs is the content served by the server-loaded tabs demo.
var patternTabs = map[string]string{
	"templ":  "templ compiles these panels to type-checked Go — a bad prop is a build error, not a runtime surprise.",
	"htmx":   "HTMX fetched this panel with one attribute. The server rendered it; no client router, no JSON glue.",
	"alpine": "Alpine covers what stays client-only — the source tabs next to each demo are an x-show, no round trip.",
}

// PatternsPage renders the full showcase.
func PatternsPage(w http.ResponseWriter, r *http.Request) {
	props := pages.PatternsPageProps{
		BaseProps: view.NewBaseProps("Pattern Showcase"),
		Sections:  patternsCatalog,
	}
	if err := view.Render(w, r, http.StatusOK, pages.PatternsPage(props)); err != nil {
		slog.Error("Failed to render patterns page", "error", err)
	}
}

// PatternSwap serves the partial-swap demo fragment.
func PatternSwap(w http.ResponseWriter, r *http.Request) {
	renderPattern(w, r, "swap", partials.PatternSwapResult())
}

// PatternSearch serves the live-search demo results.
func PatternSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	props := partials.PatternSearchProps{Query: q, Results: filterPatternStubData(q)}
	renderPattern(w, r, "search", partials.PatternSearchResults(props))
}

// PatternTypeahead serves the typeahead demo results (same server shape as
// live search; the loading indicator is client wiring).
func PatternTypeahead(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	props := partials.PatternSearchProps{Query: q, Results: filterPatternStubData(q)}
	renderPattern(w, r, "typeahead", partials.PatternSearchResults(props))
}

// PatternEditForm serves the click-to-edit form.
func PatternEditForm(w http.ResponseWriter, r *http.Request) {
	props := partials.PatternEditProps{ID: chi.URLParam(r, "id"), Value: "Click me, then save your edit"}
	renderPattern(w, r, "edit form", partials.PatternEditForm(props))
}

// PatternEditSave accepts the edit and returns the display view.
func PatternEditSave(w http.ResponseWriter, r *http.Request) {
	value := strings.TrimSpace(r.FormValue("value"))
	if value == "" {
		value = "(nothing entered)"
	}
	props := partials.PatternEditProps{ID: chi.URLParam(r, "id"), Value: value}
	renderPattern(w, r, "edit save", partials.PatternEditDisplay(props))
}

// PatternFavorite flips the favorite state carried in hx-vals — stateless on
// purpose, so the demo needs no store to retire later (ADR-024 Slice C).
func PatternFavorite(w http.ResponseWriter, r *http.Request) {
	favorited := r.FormValue("favorited") == "true"
	props := partials.PatternFavoriteProps{ID: chi.URLParam(r, "id"), Favorited: !favorited}
	renderPattern(w, r, "favorite", partials.PatternFavoriteButton(props))
}

// PatternScroll serves one page of the infinite-scroll demo.
func PatternScroll(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	start := (page - 1) * patternScrollPerPage
	if start >= len(patternStubData) {
		start = len(patternStubData)
	}
	end := start + patternScrollPerPage
	if end > len(patternStubData) {
		end = len(patternStubData)
	}
	next := page + 1
	if end == len(patternStubData) {
		next = 0 // last page: no sentinel
	}
	props := partials.PatternScrollProps{Items: patternStubData[start:end], NextPage: next}
	renderPattern(w, r, "scroll", partials.PatternScrollPage(props))
}

// PatternToast fires a toast through the HX-Trigger header contract the base
// layout listens for; the body is just a small status line.
func PatternToast(w http.ResponseWriter, r *http.Request) {
	toastType := r.FormValue("type")
	switch toastType {
	case "error", "warning":
	default:
		toastType = "success"
	}
	// HX-Trigger rides an HTTP header (latin-1): keep the message ASCII or
	// browsers render mojibake.
	view.SetHXTrigger(w, "Toast from the server - no client toast library involved.")
	w.Header().Set("HX-Toast-Type", toastType)
	renderPattern(w, r, "toast", partials.PatternToastResult())
}

// PatternSkeleton serves the content behind the skeleton-loading demo. The
// perceived delay is the client-side load trigger — the server returns
// immediately rather than sleeping on a demo's behalf.
func PatternSkeleton(w http.ResponseWriter, r *http.Request) {
	renderPattern(w, r, "skeleton", partials.PatternSkeletonContent())
}

// PatternTab serves a server-loaded tab panel.
func PatternTab(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	content, ok := patternTabs[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	props := partials.PatternTabProps{Name: name, Content: content}
	renderPattern(w, r, "tab", partials.PatternTabPanel(props))
}

// PatternBulk reports how many selected items the batch contained.
func PatternBulk(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	renderPattern(w, r, "bulk", partials.PatternBulkResult(len(r.Form["selected"])))
}

// renderPattern renders a demo fragment, logging render failures like every
// other handler in this package.
func renderPattern(w http.ResponseWriter, r *http.Request, name string, c templ.Component) {
	if err := view.Render(w, r, http.StatusOK, c); err != nil {
		slog.Error("Failed to render pattern fragment", "pattern", name, "error", err)
	}
}

// filterPatternStubData does the case-insensitive contains filtering shared
// by the search demos.
func filterPatternStubData(q string) []string {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	var results []string
	for _, item := range patternStubData {
		if strings.Contains(strings.ToLower(item), q) {
			results = append(results, item)
		}
	}
	return results
}
