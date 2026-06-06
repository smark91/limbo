package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/smark91/limbo/internal/database"
	"github.com/smark91/limbo/internal/scanner"

	"gorm.io/gorm"
)

var Version = "dev"

type statsResponse struct {
	Pending        int64     `json:"pending"`
	Unavailable    int64     `json:"unavailable"`
	WaitingRelease int64     `json:"waitingRelease"`
	Completed      int64     `json:"completed"`
	Total          int64     `json:"total"`
	LastScan       time.Time `json:"lastScan"`
	Version        string    `json:"version"`
}

// handleStats returns aggregated triage counters by status.
func handleStats(db *gorm.DB, scan *scanner.Scanner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var stats statsResponse

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status = ?", database.StatusPending).Count(&stats.Pending).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count pending status", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status = ?", database.StatusUnavailable).Count(&stats.Unavailable).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count unavailable status", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status = ?", database.StatusWaitingRelease).Count(&stats.WaitingRelease).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count waiting release status", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status = ?", database.StatusCompleted).Count(&stats.Completed).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count completed status", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Count(&stats.Total).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count total status", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		stats.LastScan = scan.LastScanTime()
		stats.Version = Version

		writeJSON(w, r, http.StatusOK, stats)
	}
}
