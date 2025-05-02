package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/go-alpine-saas-starter/internal/auth"
	"github.com/yourusername/go-alpine-saas-starter/internal/config"
	"github.com/yourusername/go-alpine-saas-starter/internal/handler"
	mw "github.com/yourusername/go-alpine-saas-starter/internal/middleware"
	"github.com/yourusername/go-alpine-saas-starter/internal/repository"
	"github.com/yourusername/go-alpine-saas-starter/internal/webutil"
)

// Server represents the main application server.
type Server struct {
	router     chi.Router
	db         *pgxpool.Pool
	authClient *auth.AuthClient
	// Add other dependencies like database connections here as needed
}

// New creates a new Server instance.
func New(cfg *config.Config, db *pgxpool.Pool) (*Server, error) {
	// Initialize Supabase Auth Client
	authClient, err := auth.NewAuthClient(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	if err != nil {
		slog.Error("Failed to create Supabase auth client", "error", err)
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}
	slog.Info("Supabase auth client initialized")

	r := chi.NewRouter()

	s := &Server{
		router:     r,
		db:         db,
		authClient: authClient,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s, nil
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

func (s *Server) setupMiddleware() {
	// Basic middleware
	s.router.Use(mw.RequestID)
	s.router.Use(mw.RealIP)
	s.router.Use(mw.Logger)
	s.router.Use(mw.Recoverer)
	s.router.Use(mw.Timeout(30 * time.Second))

	// Inject UserRepository into context for all routes
	s.router.Use(mw.UserRepoMiddleware(repository.NewUserRepository(s.db)))

	// Static file server with cache control
	fileServer(s.router, "/static", http.Dir("./web/static"))
}

func (s *Server) setupRoutes() {
	r := s.router // Use the router from the Server struct

	// Static & Informational Pages
	r.Get("/dashboard", handler.DashboardPage)
	r.Get("/terms", handler.TermsPage)
	r.Get("/privacy", handler.PrivacyPage)

	// Logout GET route for UX (renders confirm form)
	r.Get("/auth/logout", handler.LogoutPage)

	// First-run onboarding (after signup)
	handler.FirstRunHandlers(r)

	// API routes
	s.router.Route("/api", func(api chi.Router) {
		api.Get("/", handler.APIPlaceholder)
		api.Get("/users/{userID}", handler.GetUserProfile)
		api.Get("/organizations", handler.ListOrganizations)
	})

	// Health check endpoint (liveness)
	s.router.Get("/healthz", handler.HealthHandler)

	// Profile page (HTMX-enabled)
	r.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(mw.AuthMiddleware(s.authClient))
		protectedRouter.Get("/profile", handler.ProfileView)
		protectedRouter.Post("/profile", handler.ProfileUpdate)
	})

	// Items list (HTMX-powered)
	s.router.Get("/items", handler.ItemsPage)
	s.router.Get("/items/list", handler.ItemsList)
	// Toggle favorite (optimistic UI)
	s.router.Post("/items/{id}/toggle", handler.ItemToggle)

	// --- Authentication Routes --- 
	r.Route("/auth", func(authRouter chi.Router) {
		authRouter.Get("/page", handler.AuthPage) // Show login/signup form
		authRouter.Post("/login", handler.AuthLoginPost(s.authClient)) // Handle login
		authRouter.Post("/signup", handler.AuthSignupPost(s.authClient)) // Handle signup
		authRouter.Post("/logout", handler.AuthLogoutPost(s.authClient)) // Handle logout
	})

	// Home page route
	r.Get("/", homeHandler) // Use the router variable 'r'
}

// ServeHTTP implements the http.Handler interface, making Server usable with http.ListenAndServe.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

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
