package api

import (
	"log/slog"
	"net/http"

	"limbo/internal/seerr"

	"gorm.io/gorm"
)

// handleHealth returns a simple healthcheck endpoint.
func handleHealth(db *gorm.DB, seerrClient *seerr.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dbStatus := "up"
		seerrStatus := "up"
		hasError := false

		// 1. Check Database
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = "down"
			hasError = true
			slog.ErrorContext(ctx, "Health check failed: database instance unavailable", slog.Any("error", err))
		} else if err := sqlDB.PingContext(ctx); err != nil {
			dbStatus = "down"
			hasError = true
			slog.ErrorContext(ctx, "Health check failed: database ping failed", slog.Any("error", err))
		}

		// 2. Check Seerr API
		if err := seerrClient.Ping(ctx); err != nil {
			seerrStatus = "down"
			hasError = true
			slog.ErrorContext(ctx, "Health check failed: Seerr API ping failed", slog.Any("error", err))
		}

		response := map[string]string{
			"status":   "ok",
			"database": dbStatus,
			"seerr":    seerrStatus,
		}

		status := http.StatusOK
		if hasError {
			response["status"] = "error"
			status = http.StatusServiceUnavailable
		}

		writeJSON(w, r, status, response)
	}
}
