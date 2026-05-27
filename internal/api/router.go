package api

import (
	"net/http"

	"limbo/internal/config"
	"limbo/internal/scanner"
	"limbo/internal/seerr"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

// Deps holds all API handler dependencies.
type Deps struct {
	Config   *config.Config
	DB       *gorm.DB
	Scanner  *scanner.Scanner
	Seerr    *seerr.Client
}

// NewRouter sets up the Chi router with all API routes and middleware.
func NewRouter(deps Deps, frontendFS http.FileSystem) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Api-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handleHealth(deps.DB, deps.Seerr))
		r.Get("/stats", handleStats(deps.DB, deps.Scanner))
		r.Get("/requests", handleRequests(deps.Config, deps.DB))
		r.Get("/triage/{seerrRequestId}", handleGetTriage(deps.DB))
		r.Post("/triage", handlePostTriage(deps.DB))
		r.Post("/sync", handleSync(deps.Scanner))
		r.Post("/maintenance/clean-older", handleCleanOlder(deps.DB, deps.Seerr))
		r.Post("/maintenance/refresh-cache", handleRefreshCache(deps.DB, deps.Seerr, deps.Config))
		r.Get("/maintenance/cache", handleGetCacheInfo(deps.DB, deps.Config))
		r.Post("/maintenance/test-notification", handleTestNotification(deps.DB, deps.Scanner, deps.Seerr))
		r.Get("/notifications/config", handleGetNotificationsConfig(deps.Config))
		r.Post("/notifications/subscribe", handleSubscribe(deps.DB))
	})

	// Frontend static files — serve SPA
	fileServer := http.FileServer(frontendFS)
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file directly
		f, err := frontendFS.Open(req.URL.Path)
		if err != nil {
			// If file not found, serve index.html for SPA routing
			req.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, req)
	})

	return r
}
