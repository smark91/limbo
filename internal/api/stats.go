package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"limbo/internal/database"
	"limbo/internal/scanner"

	"gorm.io/gorm"
)

type statsResponse struct {
	Pending        int64     `json:"pending"`
	NotAvailable   int64     `json:"notAvailable"`
	WaitingRelease int64     `json:"waitingRelease"`
	Completed      int64     `json:"completed"`
	Total          int64     `json:"total"`
	LastScan       time.Time `json:"lastScan"`
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

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status = ?", database.StatusNotAvailable).Count(&stats.NotAvailable).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count not available status", slog.Any("error", err))
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}
