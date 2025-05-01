package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/go-alpine-saas-starter/internal/view"
	"github.com/yourusername/go-alpine-saas-starter/internal/webutil"
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
			IsFavorite: itemStore[strconv.Itoa(id)], // Check store
		})
	}
	data := map[string]interface{}{ "Items": items, "NextPage": page + 1 }
	webutil.RenderTemplate(w, r, http.StatusOK, "pages/items.html", data)
}

// ItemsList returns a fragment of items for HTMX requests
func ItemsList(w http.ResponseWriter, r *http.Request) {
	// Typeahead search support
	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) != "" {
		// Generate stub items up to 50 and filter
		var results []view.Item
		for id := 1; id <= 50; id++ {
			name := fmt.Sprintf("Item %d", id)
			if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
				results = append(results, view.Item{
					ID:         strconv.Itoa(id),
					Name:       name,
					IsFavorite: itemStore[strconv.Itoa(id)], // Check store
				})
			}
		}
		listData := map[string]interface{}{"Items": results}
		webutil.RenderTemplate(w, r, http.StatusOK, "partials/items_list.html", listData)
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
			IsFavorite: itemStore[strconv.Itoa(id)], // Check store
		})
	}

	data := map[string]interface{}{
		"Items":    items,
		"NextPage": page + 1,
	}

	// Render fragment or full page depending on HTMX
	if webutil.IsHTMXRequest(r) {
		webutil.RenderTemplate(w, r, http.StatusOK, "partials/items_list.html", data)
	} else {
		webutil.RenderTemplate(w, r, http.StatusOK, "pages/items.html", data)
	}
}

// ItemToggle handles toggling the favorite status of an item
func ItemToggle(w http.ResponseWriter, r *http.Request) {
	// Get item ID from path parameter
	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Toggle favorite status in our stub store
	itemStore[itemID] = !itemStore[itemID]
	isFavorite := itemStore[itemID]

	// Prepare data for the partial template
	// In a real app, you'd fetch the full item details here
	itemName := fmt.Sprintf("Item %s", itemID) // Stub name
	data := view.Item{
		ID:         itemID,
		Name:       itemName,
		IsFavorite: isFavorite, // Assuming you add this field to view.Item
	}

	// Return the updated item partial
	// Pass the item itself as data context for the partial
	webutil.RenderTemplate(w, r, http.StatusOK, "partials/item.html", map[string]interface{}{"Item": data})
}
