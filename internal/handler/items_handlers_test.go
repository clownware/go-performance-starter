package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
)

// withURLParam attaches a chi URL parameter to the request so handlers that call
// chi.URLParam resolve it, without standing up a full router.
func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestItemToggle(t *testing.T) {
	tests := []struct {
		name       string
		itemID     string
		seed       bool // initial favorite state in the store
		wantStatus int
		wantFav    bool // expected store state after the toggle
	}{
		{name: "missing id returns 400", itemID: "", wantStatus: http.StatusBadRequest},
		{name: "unfavorited item becomes favorite", itemID: "toggle-a", seed: false, wantStatus: http.StatusOK, wantFav: true},
		{name: "favorited item becomes unfavorite", itemID: "toggle-b", seed: true, wantStatus: http.StatusOK, wantFav: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.itemID != "" {
				itemStore[tt.itemID] = tt.seed
				t.Cleanup(func() { delete(itemStore, tt.itemID) })
			}

			req := httptest.NewRequest(http.MethodPost, "/items/"+tt.itemID+"/toggle", nil)
			if tt.itemID != "" {
				req = withURLParam(req, "id", tt.itemID)
			}
			w := httptest.NewRecorder()

			ItemToggle(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("ItemToggle() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				if got := itemStore[tt.itemID]; got != tt.wantFav {
					t.Errorf("itemStore[%q] = %v, want %v", tt.itemID, got, tt.wantFav)
				}
			}
		})
	}
}

func TestItemToggleConcurrent(t *testing.T) {
	// Regression guard for the itemStore data race: handlers mutate and read
	// the package-global map from concurrent requests, so toggles and list
	// renders must be safe under -race.
	const workers = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "race-" + strconv.Itoa(n%2) // force key contention
			req := withURLParam(httptest.NewRequest(http.MethodPost, "/items/"+id+"/toggle", nil), "id", id)
			ItemToggle(httptest.NewRecorder(), req)

			ItemsList(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items?page=1", nil))
		}(i)
	}
	wg.Wait()

	for n := 0; n < 2; n++ {
		delete(itemStore, "race-"+strconv.Itoa(n))
	}
}

func TestItemsList(t *testing.T) {
	tests := []struct {
		name   string
		target string
		htmx   bool
	}{
		{name: "default first page", target: "/items"},
		{name: "explicit page param", target: "/items?page=3"},
		{name: "typeahead search", target: "/items?query=Item+1"},
		{name: "htmx fragment request", target: "/items?page=2", htmx: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			w := httptest.NewRecorder()

			ItemsList(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("ItemsList() status = %d, want %d", w.Code, http.StatusOK)
			}
			if w.Body.Len() == 0 {
				t.Error("ItemsList() rendered an empty body")
			}
		})
	}
}

func TestItemsPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	w := httptest.NewRecorder()

	ItemsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ItemsPage() status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.Len() == 0 {
		t.Error("ItemsPage() rendered an empty body")
	}
}
