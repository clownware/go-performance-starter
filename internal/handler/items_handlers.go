package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
	"github.com/clownware/alpine-go-performance-starter/internal/view/partials"
	"github.com/go-chi/chi/v5"
)

const itemsPerPage = 5

// itemStore simulates a data store for favorite status (replace with real DB later)
var itemStore = make(map[string]bool)

// ItemsPage renders the items page, which will load the list via HTMX
func ItemsPage(w http.ResponseWriter, r *http.Request) {
	// Load first page of items for baseline (no JS) fallback
	page := 1
	items := make([]view.Item, 0, itemsPerPage)
	start := (page - 1) * itemsPerPage
	for i := 1; i <= itemsPerPage; i++ {
		id := start + i
		items = append(items, view.Item{
			ID:         strconv.Itoa(id),
			Name:       fmt.Sprintf("Item %d", id),
			IsFavorite: itemStore[strconv.Itoa(id)],
		})
	}
	props := pages.ItemsPageProps{
		BaseProps: view.NewBaseProps("Items List"),
		Items:     items,
		NextPage:  page + 1,
	}
	if err := view.Render(w, r, http.StatusOK, pages.ItemsPage(props)); err != nil {
		slog.Error("Failed to render items page", "error", err)
	}
}

// ItemsList returns a fragment of items for HTMX requests
func ItemsList(w http.ResponseWriter, r *http.Request) {
	// Typeahead search support
	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) != "" {
		var results []view.Item
		for id := 1; id <= 50; id++ {
			name := fmt.Sprintf("Item %d", id)
			if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
				results = append(results, view.Item{
					ID:         strconv.Itoa(id),
					Name:       name,
					IsFavorite: itemStore[strconv.Itoa(id)],
				})
			}
		}
		listProps := partials.ItemsListProps{Items: results}
		if err := view.Render(w, r, http.StatusOK, partials.ItemsList(listProps)); err != nil {
			slog.Error("Failed to render items list partial", "error", err)
		}
		return
	}

	// Determine page number from query
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	// Stub items
	items := make([]view.Item, 0, itemsPerPage)
	start := (page - 1) * itemsPerPage
	for i := 1; i <= itemsPerPage; i++ {
		id := start + i
		items = append(items, view.Item{
			ID:         strconv.Itoa(id),
			Name:       fmt.Sprintf("Item %d", id),
			IsFavorite: itemStore[strconv.Itoa(id)],
		})
	}

	listProps := partials.ItemsListProps{
		Items:    items,
		NextPage: page + 1,
	}

	// Render fragment or full page depending on HTMX
	if view.IsHTMXRequest(r) {
		if err := view.Render(w, r, http.StatusOK, partials.ItemsList(listProps)); err != nil {
			slog.Error("Failed to render items list partial", "error", err)
		}
	} else {
		props := pages.ItemsPageProps{
			BaseProps: view.NewBaseProps("Items List"),
			Items:     items,
			NextPage:  page + 1,
		}
		if err := view.Render(w, r, http.StatusOK, pages.ItemsPage(props)); err != nil {
			slog.Error("Failed to render items page", "error", err)
		}
	}
}

// ItemToggle handles toggling the favorite status of an item
func ItemToggle(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Toggle favorite status in our stub store
	itemStore[itemID] = !itemStore[itemID]
	isFavorite := itemStore[itemID]

	item := view.Item{
		ID:         itemID,
		Name:       fmt.Sprintf("Item %s", itemID),
		IsFavorite: isFavorite,
	}

	if err := view.Render(w, r, http.StatusOK, partials.ItemCard(item)); err != nil {
		slog.Error("Failed to render item partial", "error", err)
	}
}
