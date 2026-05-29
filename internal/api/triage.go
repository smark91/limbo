package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"limbo/internal/database"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type triageRequest struct {
	SeerrRequestID int     `json:"seerrRequestId"`
	Status         string  `json:"status"`
	Reason         *string `json:"reason,omitempty"`
}

// handleGetTriage returns the triage entry for a specific Seerr request.
func handleGetTriage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		idStr := chi.URLParam(r, "seerrRequestId")
		seerrID, err := strconv.Atoi(idStr)
		if err != nil {
			slog.WarnContext(ctx, "Invalid request ID format in GET triage", slog.String("idStr", idStr))
			http.Error(w, "Invalid request ID", http.StatusBadRequest)
			return
		}

		var entry database.TriageEntry
		result := db.WithContext(ctx).Where("seerr_request_id = ?", seerrID).First(&entry)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if result.Error != nil {
			slog.ErrorContext(ctx, "Database error looking up triage entry", slog.Int("seerrId", seerrID), slog.Any("error", result.Error))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}
}

// handlePostTriage creates or updates a triage entry.
func handlePostTriage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req triageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			slog.WarnContext(ctx, "Invalid JSON body in POST triage", slog.Any("error", err))
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.SeerrRequestID == 0 {
			http.Error(w, "seerrRequestId is required", http.StatusBadRequest)
			return
		}

		// Validate status
		if req.Status != "" {
			if req.Status == database.StatusCompleted {
				slog.WarnContext(ctx, "Manual transition to COMPLETED is forbidden", slog.Int("seerrId", req.SeerrRequestID))
				http.Error(w, "Cannot manually transition to COMPLETED status", http.StatusBadRequest)
				return
			}
			valid := false
			for _, s := range database.AllStatuses() {
				if s == req.Status {
					valid = true
					break
				}
			}
			if !valid {
				slog.WarnContext(ctx, "Invalid triage status submitted", slog.String("status", req.Status))
				http.Error(w, "Invalid status", http.StatusBadRequest)
				return
			}
		}

		var entry database.TriageEntry
		result := db.WithContext(ctx).Where("seerr_request_id = ?", req.SeerrRequestID).First(&entry)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new entry
			entry = database.TriageEntry{
				SeerrRequestID: req.SeerrRequestID,
				Status:         req.Status,
				Reason:         req.Reason,
			}
			if entry.Status == "" || entry.Status == database.StatusCompleted {
				entry.Status = database.StatusPending
			}
			if err := db.WithContext(ctx).Create(&entry).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to create triage entry", slog.Int("seerrId", req.SeerrRequestID), slog.Any("error", err))
				http.Error(w, "Failed to create triage entry", http.StatusInternalServerError)
				return
			}
			slog.InfoContext(ctx, "Created new triage entry", slog.Int("seerrId", req.SeerrRequestID), slog.String("status", entry.Status))
		} else if result.Error == nil {
			// Prevent changing status away from or to COMPLETED
			if entry.Status == database.StatusCompleted && req.Status != "" && req.Status != database.StatusCompleted {
				slog.WarnContext(ctx, "Cannot manually change status of a completed request", slog.Int("seerrId", req.SeerrRequestID))
				http.Error(w, "Cannot change status of a completed request", http.StatusBadRequest)
				return
			}

			// Update existing
			updates := map[string]interface{}{}
			if req.Status != "" {
				updates["status"] = req.Status
			}
			if req.Reason != nil {
				updates["reason"] = req.Reason
			}

			if len(updates) > 0 {
				if err := db.WithContext(ctx).Model(&entry).Updates(updates).Error; err != nil {
					slog.ErrorContext(ctx, "Failed to update triage entry", slog.Int("seerrId", req.SeerrRequestID), slog.Any("error", err))
					http.Error(w, "Failed to update triage entry", http.StatusInternalServerError)
					return
				}
			}

			// Reload
			if err := db.WithContext(ctx).Where("seerr_request_id = ?", req.SeerrRequestID).First(&entry).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to reload triage entry after update", slog.Int("seerrId", req.SeerrRequestID), slog.Any("error", err))
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			slog.InfoContext(ctx, "Updated triage entry", slog.Int("seerrId", req.SeerrRequestID), slog.Any("updates", updates))
		} else {
			slog.ErrorContext(ctx, "Database error finding triage entry in POST triage", slog.Int("seerrId", req.SeerrRequestID), slog.Any("error", result.Error))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(entry)
	}
}
