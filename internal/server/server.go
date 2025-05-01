package server

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	middleware "github.com/yourusername/go-alpine-saas-starter/internal/middleware"
	"github.com/yourusername/go-alpine-saas-starter/internal/handler"
	"github.com/yourusername/go-alpine-saas-starter/internal/webutil"
)

// New creates and configures a new Chi router with middleware and routes
func New() *chi.Mux {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Static file server with cache control
	fileServer(r, "/static", http.Dir("./web/static"))

	// API routes
	r.Route("/api", func(api chi.Router) {
		api.Get("/", handler.APIPlaceholder)
		api.Get("/users/{userID}", handler.GetUserProfile)
		api.Get("/organizations", handler.ListOrganizations)
	})

	// Health check endpoint (liveness)
	r.Get("/healthz", handler.HealthHandler)

	// Profile page (HTMX-enabled)
	r.Get("/profile", handler.ProfileView)
	r.Post("/profile", handler.ProfileUpdate)

	// Home page route
	r.Get("/", homeHandler)

	return r
}

// fileServer conveniently sets up a http.FileServer handler to serve static files
// from a http.FileSystem with proper cache control headers
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		// Define cache durations based on file type
		rctx := chi.RouteContext(r.Context())
		pathPrefix := rctx.RoutePattern()[:len(rctx.RoutePattern())-2] // strip the /*
		fs := http.StripPrefix(pathPrefix, cacheControlWrapper(http.FileServer(root)))
		fs.ServeHTTP(w, r)
	})
}

// cacheControlWrapper adds cache control headers based on file extension
func cacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add cache control headers based on file extension
		path := r.URL.Path

		// Set Cache-Control headers based on file type
		switch {
		// CSS, JS, and images can be cached for longer periods
		case isFileType(path, ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"):
			w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		// Fonts can be cached for a long time
		case isFileType(path, ".woff", ".woff2", ".ttf", ".otf", ".eot"):
			w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		// HTML and other files should have shorter cache times
		default:
			w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		}

		h.ServeHTTP(w, r)
	})
}

// isFileType checks if the file path has any of the specified extensions
func isFileType(filePath string, extensions ...string) bool {
	ext := path.Ext(filePath)
	for _, e := range extensions {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// homeHandler renders the home page using the base layout template
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate a server-side validation error for demonstration
	errors := webutil.FormErrors{
		"test_field": "Server: This value is invalid!",
	}

	// Prepare data for the template (can be nil if no extra data needed)
	data := map[string]interface{}{}

	// Render the home page template with the errors
	webutil.RenderTemplateWithErrors(w, r, http.StatusOK, "pages/home.html", data, errors)
}
