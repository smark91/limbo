package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/scanner"
	"github.com/smark91/limbo/internal/seerr"

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
	r.Use(requestLogger())
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
		r.Get("/maintenance/info", handleGetSystemInfo(deps.DB, deps.Seerr, deps.Config))
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
			_ = f.Close()
		}
		fileServer.ServeHTTP(w, req)
	})

	return r
}

// requestLogger logs HTTP requests using log/slog at the Debug level.
func requestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			
			next.ServeHTTP(ww, r)
			
			// #nosec G706
			slog.Debug("HTTP Request",
				slog.String("method", sanitizeLogInput(r.Method)),
				slog.String("path", sanitizeLogInput(r.URL.Path)),
				slog.Int("status", ww.Status()),
				slog.String("ip", sanitizeLogInput(r.RemoteAddr)),
				slog.Duration("duration", time.Since(start)),
				slog.Int("bytes", ww.BytesWritten()),
			)
		})
	}
}

// sanitizeLogInput removes newlines and carriage returns to prevent log injection.
func sanitizeLogInput(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
