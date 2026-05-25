package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/seerr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Scanner runs periodic background scans of Seerr requests.
type Scanner struct {
	cfg      *config.Config
	db       *gorm.DB
	seerr    *seerr.Client
	notifier *Notifier
	mu       sync.Mutex
	lastScan time.Time
}

// New creates a new Scanner instance.
func New(cfg *config.Config, db *gorm.DB, seerrClient *seerr.Client) *Scanner {
	var notifier *Notifier
	if cfg.DiscordWebhookURL != "" {
		notifier = NewNotifier(cfg.DiscordWebhookURL)
	}

	var lastScan time.Time
	var meta database.SystemMetadata
	// Use background context for initialization
	ctx := context.Background()
	if err := db.WithContext(ctx).First(&meta, "key = ?", "last_scan_at").Error; err == nil {
		if t, err := time.Parse(time.RFC3339, meta.Value); err == nil {
			lastScan = t
		}
	}

	return &Scanner{
		cfg:      cfg,
		db:       db,
		seerr:    seerrClient,
		notifier: notifier,
		lastScan: lastScan,
	}
}

// Run starts the background scan loop. Blocks until context is cancelled.
func (s *Scanner) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.ScanInterval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "Scanner starting", slog.Duration("interval", s.cfg.ScanInterval))

	// Run immediately on startup
	s.scan(ctx)

	for {
		select {
		case <-ticker.C:
			s.scan(ctx)
		case <-ctx.Done():
			slog.InfoContext(ctx, "Scanner stopped")
			return
		}
	}
}

// TriggerScan runs an immediate scan (for the /api/sync endpoint).
func (s *Scanner) TriggerScan(ctx context.Context) {
	s.scan(ctx)
}

// LastScanTime returns when the last scan completed.
func (s *Scanner) LastScanTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastScan
}

func (s *Scanner) scan(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.InfoContext(ctx, "Starting scan cycle...")
	startTime := time.Now()

	requests, err := s.seerr.GetApprovedRequests(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching requests from Seerr", slog.Any("error", err))
		return
	}

	slog.InfoContext(ctx, "Fetched approved requests", slog.Int("count", len(requests)))

	// Track seen IDs for reconciliation (only for active requests)
	var currentIDs []int
	if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).Where("status != ?", database.StatusCompleted).Pluck("seerr_request_id", &currentIDs).Error; err != nil {
		slog.ErrorContext(ctx, "Error fetching current request IDs from database", slog.Any("error", err))
		return
	}

	seenIDs := make(map[int]bool)
	for _, id := range currentIDs {
		seenIDs[id] = false
	}

	processed := 0
	for _, req := range requests {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Scan cycle interrupted by context cancellation")
			return
		default:
		}

		seenIDs[req.ID] = true

		// Skip if actively downloading
		if len(req.Media.DownloadStatus) > 0 {
			continue
		}

		if err := s.processRequest(ctx, req); err != nil {
			slog.ErrorContext(ctx, "Error processing request",
				slog.Int("requestId", req.ID),
				slog.Int("tmdbId", req.Media.TmdbID),
				slog.Any("error", err),
			)
			continue
		}
		processed++
	}

	// Reconcile and purge stale entries (IDs that weren't seen in the Seerr response)
	staleCount := 0
	completedCount := 0
	for id, seen := range seenIDs {
		if !seen {
			// Query Seerr API to check if it was completed or deleted
			seerrReq, err := s.seerr.GetRequest(ctx, id)
			if err != nil {
				// If the request returns 404, it was deleted in Seerr
				if strings.Contains(err.Error(), "returned 404") {
					slog.InfoContext(ctx, "Reconciling missing request: request not found in Seerr, deleting locally", slog.Int("requestId", id))
					if err := s.db.WithContext(ctx).Where("seerr_request_id = ?", id).Delete(&database.TriageEntry{}).Error; err != nil {
						slog.ErrorContext(ctx, "Error deleting stale triage entry", slog.Int("requestId", id), slog.Any("error", err))
					} else {
						staleCount++
					}
				} else {
					slog.ErrorContext(ctx, "Error fetching missing request from Seerr during reconciliation, skipping", slog.Int("requestId", id), slog.Any("error", err))
				}
				continue
			}

			// If the media status is completed, transition it to COMPLETED in database
			if seerrReq.Media.Status == 4 || seerrReq.Media.Status == 5 {
				slog.InfoContext(ctx, "Reconciling missing request: media available in Seerr, marking as completed locally", slog.Int("requestId", id))
				fulfilledAt := parseTime(seerrReq.UpdatedAt)
				updates := map[string]interface{}{
					"status":       database.StatusCompleted,
					"fulfilled_at": &fulfilledAt,
				}
				if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).Where("seerr_request_id = ?", id).Updates(updates).Error; err != nil {
					slog.ErrorContext(ctx, "Error updating completed triage entry during reconciliation", slog.Int("requestId", id), slog.Any("error", err))
				} else {
					completedCount++
				}
			} else {
				// If it was declined or changed status to something not approved, delete it
				slog.InfoContext(ctx, "Reconciling missing request: request no longer approved in Seerr, deleting locally", slog.Int("requestId", id))
				if err := s.db.WithContext(ctx).Where("seerr_request_id = ?", id).Delete(&database.TriageEntry{}).Error; err != nil {
					slog.ErrorContext(ctx, "Error deleting stale triage entry", slog.Int("requestId", id), slog.Any("error", err))
				} else {
					staleCount++
				}
			}
		}
	}

	if staleCount > 0 {
		slog.InfoContext(ctx, "Purged stale entries", slog.Int("count", staleCount))
	}
	if completedCount > 0 {
		slog.InfoContext(ctx, "Marked missing entries as completed", slog.Int("count", completedCount))
	}

	s.lastScan = time.Now()

	// Persist to database
	if err := s.db.WithContext(ctx).Save(&database.SystemMetadata{
		Key:   "last_scan_at",
		Value: s.lastScan.Format(time.RFC3339),
	}).Error; err != nil {
		slog.ErrorContext(ctx, "Error saving last scan time to database", slog.Any("error", err))
	}

	slog.InfoContext(ctx, "Scan cycle complete",
		slog.Int("processed", processed),
		slog.Duration("duration", time.Since(startTime).Round(time.Millisecond)),
	)
}

func (s *Scanner) processRequest(ctx context.Context, req seerr.SeerrRequest) error {
	tmdbID := req.Media.TmdbID
	mediaType := req.MediaType

	var title string
	var posterPath string
	var releaseInfo ReleaseInfo

	switch mediaType {
	case "movie":
		detail, err := s.seerr.GetMovieDetail(ctx, tmdbID)
		if err != nil {
			return fmt.Errorf("fetching movie %d: %w", tmdbID, err)
		}
		title = detail.Title
		posterPath = detail.PosterPath
		releaseInfo = EvaluateMovieRelease(detail, s.cfg)

	case "tv":
		detail, err := s.seerr.GetTVDetail(ctx, tmdbID)
		if err != nil {
			return fmt.Errorf("fetching TV %d: %w", tmdbID, err)
		}
		title = detail.Name
		posterPath = detail.PosterPath

		// Extract requested season numbers
		var requestedSeasons []int
		for _, season := range req.Seasons {
			requestedSeasons = append(requestedSeasons, season.SeasonNumber)
		}
		releaseInfo = EvaluateTVRelease(detail, requestedSeasons)

	default:
		return fmt.Errorf("unknown media type: %s", mediaType)
	}

	// Upsert triage entry
	entry := database.TriageEntry{
		SeerrRequestID: req.ID,
		MediaID:        req.Media.ID,
		TmdbID:         tmdbID,
		PosterPath:     posterPath,
		MediaType:      mediaType,
		Title:          title,
		ReleaseDate:    releaseInfo.Date,
		ReleaseSource:  &releaseInfo.Source,
		SeerrCreatedAt: parseTime(req.CreatedAt),
		ServiceURL:     req.Media.ServiceURL,
	}

	// Check existing entry
	var existing database.TriageEntry
	result := s.db.WithContext(ctx).Where("seerr_request_id = ?", req.ID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		if req.Media.Status == 4 || req.Media.Status == 5 {
			entry.Status = database.StatusCompleted
			fulfilledAt := parseTime(req.UpdatedAt)
			entry.FulfilledAt = &fulfilledAt
		} else if releaseInfo.Date != nil && !releaseInfo.IsReleased() {
			entry.Status = database.StatusWaitingRelease
		} else {
			entry.Status = database.StatusPending
		}
		if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
			return fmt.Errorf("creating triage entry: %w", err)
		}
	} else if result.Error == nil {
		// Update cached fields, preserve user-set status
		updates := map[string]interface{}{
			"title":            title,
			"tmdb_id":          tmdbID,
			"media_id":         req.Media.ID,
			"poster_path":      posterPath,
			"release_date":     releaseInfo.Date,
			"release_source":   releaseInfo.Source,
			"service_url":      req.Media.ServiceURL,
			"seerr_created_at": parseTime(req.CreatedAt),
		}

		// Auto-transition to COMPLETED if fulfilled
		if req.Media.Status == 4 || req.Media.Status == 5 {
			updates["status"] = database.StatusCompleted
			if existing.FulfilledAt == nil {
				fulfilledAt := parseTime(req.UpdatedAt)
				updates["fulfilled_at"] = &fulfilledAt
			}
		} else {
			// If it was completed, but now it's no longer completed (re-requested/deleted)
			if existing.Status == database.StatusCompleted {
				if releaseInfo.Date != nil && !releaseInfo.IsReleased() {
					updates["status"] = database.StatusWaitingRelease
				} else {
					updates["status"] = database.StatusPending
				}
				updates["fulfilled_at"] = nil
				updates["notified_at"] = nil
			} else if existing.Status == database.StatusPending && releaseInfo.Date != nil && !releaseInfo.IsReleased() {
				updates["status"] = database.StatusWaitingRelease
			}
		}

		if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return fmt.Errorf("updating triage entry: %w", err)
		}

		// Sync GORM updates map to in-memory existing struct for subsequent logic
		if status, ok := updates["status"].(string); ok {
			existing.Status = status
		}
		if val, exists := updates["fulfilled_at"]; exists {
			if fulfilledAt, ok := val.(*time.Time); ok {
				existing.FulfilledAt = fulfilledAt
			} else {
				existing.FulfilledAt = nil
			}
		}
		if val, exists := updates["notified_at"]; exists {
			if notifiedAt, ok := val.(*time.Time); ok {
				existing.NotifiedAt = notifiedAt
			} else {
				existing.NotifiedAt = nil
			}
		}

		entry = existing
	} else {
		return fmt.Errorf("querying triage entry: %w", result.Error)
	}

	// Discord notification for qualifying requests
	if s.notifier != nil && s.shouldNotify(entry, req) {
		posterURL := ""
		if posterPath != "" {
			posterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w300%s", posterPath)
		}

		requestedBy := req.RequestedBy.DisplayName
		if requestedBy == "" {
			requestedBy = "Unknown"
		}

		requestAge := time.Since(parseTime(req.CreatedAt))

		if err := s.notifier.NotifyUnfulfilled(title, mediaType, posterURL, entry.ServiceURL, releaseInfo, requestedBy, requestAge); err != nil {
			slog.ErrorContext(ctx, "Notification error", slog.String("title", title), slog.Any("error", err))
		} else {
			now := time.Now()
			if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).
				Where("seerr_request_id = ?", req.ID).
				Update("notified_at", &now).Error; err != nil {
				slog.ErrorContext(ctx, "Error updating notified_at timestamp", slog.Int("requestId", req.ID), slog.Any("error", err))
			}
		}
	}

	return nil
}

func (s *Scanner) autoResolve(ctx context.Context, req seerr.SeerrRequest) {
	if err := s.db.WithContext(ctx).Where("seerr_request_id = ?", req.ID).Delete(&database.TriageEntry{}).Error; err != nil {
		slog.ErrorContext(ctx, "Error in autoResolve deleting triage entry", slog.Int("requestId", req.ID), slog.Any("error", err))
	}
}

// shouldNotify determines if a Discord notification should be sent.
// - Only for released media (or unknown release)
// - Only if not already notified
// - Only if within the alert window (aged past threshold but within threshold+window)
func (s *Scanner) shouldNotify(entry database.TriageEntry, req seerr.SeerrRequest) bool {
	// Already notified
	if entry.NotifiedAt != nil {
		return false
	}

	// Don't notify for manually triaged items (except PENDING)
	if entry.Status != database.StatusPending {
		return false
	}

	// Only notify for released content
	if entry.ReleaseDate != nil && entry.ReleaseDate.After(time.Now()) {
		return false
	}

	// Check age window: request must be old enough (threshold) but not too old (threshold + window)
	requestTime := parseTime(req.CreatedAt)
	age := time.Since(requestTime)

	if age < s.cfg.AlertThreshold {
		return false
	}
	if age > s.cfg.AlertThreshold+s.cfg.AlertWindow {
		return false
	}

	return true
}

func parseTime(s string) time.Time {
	// Seerr uses ISO 8601 with milliseconds (e.g., 2026-04-15T14:59:24.056Z)
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Fallback to simpler format if needed
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			slog.Error("Error parsing time string", slog.String("timeString", s), slog.Any("error", err))
			return time.Time{}
		}
	}
	return t
}

// Ensure clause import is used (for potential future upsert usage)
var _ = clause.OnConflict{}

// Notifier returns the scanner's notifier.
func (s *Scanner) Notifier() *Notifier {
	return s.notifier
}
