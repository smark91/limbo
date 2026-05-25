package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/scanner"
	"limbo/internal/seerr"

	"gorm.io/gorm"
)

type cleanOlderRequest struct {
	OlderThan time.Time `json:"olderThan"`
	Statuses  []string  `json:"statuses"`
}

// handleCleanOlder removes all unfulfilled requests older than a specific date.
func handleCleanOlder(db *gorm.DB, seerrClient *seerr.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req cleanOlderRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			slog.WarnContext(ctx, "Invalid JSON payload in clean-older", slog.Any("error", err))
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.OlderThan.IsZero() {
			http.Error(w, "olderThan date is required", http.StatusBadRequest)
			return
		}

		// Default to PENDING if no statuses provided
		if len(req.Statuses) == 0 {
			req.Statuses = []string{database.StatusPending}
		}

		// Validate statuses and ensure we don't allow COMPLETED
		for _, s := range req.Statuses {
			statusUpper := strings.ToUpper(s)
			if statusUpper == database.StatusCompleted {
				http.Error(w, "Cannot delete completed requests via maintenance", http.StatusBadRequest)
				return
			}
			valid := false
			for _, st := range database.AllStatuses() {
				if st == statusUpper {
					valid = true
					break
				}
			}
			if !valid {
				http.Error(w, "Invalid status: "+s, http.StatusBadRequest)
				return
			}
		}

		// Normalize statuses to uppercase
		for i, s := range req.Statuses {
			req.Statuses[i] = strings.ToUpper(s)
		}

		// Query matching database entries
		var entries []database.TriageEntry
		if err := db.WithContext(ctx).
			Where("seerr_created_at < ? AND status IN ?", req.OlderThan, req.Statuses).
			Find(&entries).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to query database for clean-older", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		deletedCount := 0
		for _, entry := range entries {
			// 1. Delete request in Seerr
			if err := seerrClient.DeleteRequest(ctx, entry.SeerrRequestID); err != nil {
				slog.ErrorContext(ctx, "Failed to delete request from Seerr",
					slog.Int("requestId", entry.SeerrRequestID),
					slog.String("title", entry.Title),
					slog.Any("error", err),
				)
				// Continue to clean other requests even if one fails
			}

			// 2. Best-effort media cache removal in Seerr
			if entry.MediaID != 0 {
				if err := seerrClient.DeleteMedia(ctx, entry.MediaID); err != nil {
					// This is common if another request is still linked to the media, so we log at debug/info
					slog.InfoContext(ctx, "Clean-older: Seerr media cache not removed (might be monitored or linked to other requests)",
						slog.Int("mediaId", entry.MediaID),
						slog.String("title", entry.Title),
						slog.Any("error", err),
					)
				}
			}

			// 3. Delete from local database
			if err := db.WithContext(ctx).Delete(&entry).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to delete local triage entry in clean-older",
					slog.Int("requestId", entry.SeerrRequestID),
					slog.Any("error", err),
				)
			} else {
				deletedCount++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"deletedCount": deletedCount,
		})
	}
}

// handleRefreshCache force-reloads poster paths and release dates for all active requests.
func handleRefreshCache(db *gorm.DB, seerrClient *seerr.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var entries []database.TriageEntry

		if err := db.WithContext(ctx).
			Where("status != ?", database.StatusCompleted).
			Find(&entries).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to query active database entries for cache refresh", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		refreshedCount := 0
		for _, entry := range entries {
			if entry.MediaType == "movie" {
				detail, err := seerrClient.GetMovieDetail(ctx, entry.TmdbID)
				if err != nil {
					slog.WarnContext(ctx, "Failed to fetch movie detail for cache refresh", slog.Int("tmdbId", entry.TmdbID), slog.Any("error", err))
					continue
				}

				releaseInfo := scanner.EvaluateMovieRelease(detail, cfg)
				updates := map[string]interface{}{
					"title":          detail.Title,
					"poster_path":    detail.PosterPath,
					"release_date":   releaseInfo.Date,
					"release_source": releaseInfo.Source,
				}
				db.WithContext(ctx).Model(&entry).Updates(updates)
				refreshedCount++
			} else if entry.MediaType == "tv" {
				detail, err := seerrClient.GetTVDetail(ctx, entry.TmdbID)
				if err != nil {
					slog.WarnContext(ctx, "Failed to fetch TV detail for cache refresh", slog.Int("tmdbId", entry.TmdbID), slog.Any("error", err))
					continue
				}

				releaseInfo := scanner.EvaluateTVRelease(detail, nil)
				updates := map[string]interface{}{
					"title":          detail.Name,
					"poster_path":    detail.PosterPath,
					"release_date":   releaseInfo.Date,
					"release_source": releaseInfo.Source,
				}
				db.WithContext(ctx).Model(&entry).Updates(updates)
				refreshedCount++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        true,
			"refreshedCount": refreshedCount,
		})
	}
}

// cacheInfoResponse represents the response containing cache counts and database size.
type cacheInfoResponse struct {
	ActiveCount int64 `json:"activeCount"`
	TotalCount  int64 `json:"totalCount"`
	Size        int64 `json:"size"`
}

// handleGetCacheInfo returns stats about the database cache size and item counts.
func handleGetCacheInfo(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var activeCount, totalCount int64

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status != ?", database.StatusCompleted).Count(&activeCount).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count active triage entries for cache info", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := db.WithContext(ctx).Model(&database.TriageEntry{}).Count(&totalCount).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to count total triage entries for cache info", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		var size int64
		switch cfg.DBDriver {
		case "postgres":
			if err := db.WithContext(ctx).Raw("SELECT pg_database_size(current_database())").Scan(&size).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to query PostgreSQL database size", slog.Any("error", err))
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		case "sqlite":
			var pageCount, pageSize int64
			if err := db.WithContext(ctx).Raw("PRAGMA page_count").Scan(&pageCount).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to query SQLite page_count", slog.Any("error", err))
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			if err := db.WithContext(ctx).Raw("PRAGMA page_size").Scan(&pageSize).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to query SQLite page_size", slog.Any("error", err))
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			size = pageCount * pageSize
		default:
			slog.WarnContext(ctx, "Unsupported DB_DRIVER for cache size lookup", slog.String("driver", cfg.DBDriver))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cacheInfoResponse{
			ActiveCount: activeCount,
			TotalCount:  totalCount,
			Size:        size,
		})
	}
}
