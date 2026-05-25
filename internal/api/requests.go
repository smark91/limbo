package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"limbo/internal/config"
	"limbo/internal/database"

	"gorm.io/gorm"
)

type enrichedRequest struct {
	database.TriageEntry
	PosterURL  string `json:"posterUrl"`
	SeerrURL   string `json:"seerrUrl"`
	ServiceURL string `json:"serviceUrl"`
}

// handleRequests returns all triage entries enriched with links.
// Supports query params: ?status=, ?type=, ?search=, ?sort=
func handleRequests(cfg *config.Config, db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := db.WithContext(ctx).Model(&database.TriageEntry{})

		// Filter by status
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("status = ?", strings.ToUpper(status))
		}

		// Filter by type
		if mediaType := r.URL.Query().Get("type"); mediaType != "" {
			query = query.Where("media_type = ?", strings.ToLower(mediaType))
		}

		// Search by title
		if search := r.URL.Query().Get("search"); search != "" {
			query = query.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(search)+"%")
		}

		// Sort
		sort := r.URL.Query().Get("sort")
		switch sort {
		case "title":
			query = query.Order("title ASC")
		case "release":
			query = query.Order("release_date ASC NULLS LAST")
		case "oldest":
			query = query.Order("seerr_created_at ASC")
		default:
			query = query.Order("seerr_created_at DESC")
		}

		var entries []database.TriageEntry
		if err := query.Find(&entries).Error; err != nil {
			slog.ErrorContext(ctx, "Failed to query database for triage entries", slog.Any("error", err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Enrich with poster URLs and Seerr links
		results := make([]enrichedRequest, len(entries))
		for i, entry := range entries {
			posterURL := ""
			if entry.PosterPath != "" {
				posterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w300%s", entry.PosterPath)
			}

			seerrURL := ""
			if cfg.SeerrPublicURL != "" {
				seerrURL = fmt.Sprintf("%s/%s/%d", cfg.SeerrPublicURL, entry.MediaType, entry.TmdbID)
			}

			results[i] = enrichedRequest{
				TriageEntry: entry,
				PosterURL:   posterURL,
				SeerrURL:    seerrURL,
				ServiceURL:  entry.ServiceURL,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
