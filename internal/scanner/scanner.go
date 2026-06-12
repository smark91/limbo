package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"
	"github.com/smark91/limbo/internal/seerr"

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
	notifier := NewNotifier(cfg, db)

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
	_ = s.scan(ctx)

	for {
		select {
		case <-ticker.C:
			_ = s.scan(ctx)
		case <-ctx.Done():
			slog.InfoContext(ctx, "Scanner stopped")
			return
		}
	}
}

// TriggerScan runs an immediate scan (for the /api/sync endpoint).
func (s *Scanner) TriggerScan(ctx context.Context) error {
	return s.scan(ctx)
}

// LastScanTime returns when the last scan completed.
func (s *Scanner) LastScanTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastScan
}

// preparedUpdate holds the parsed and prepared metadata updates for a request,
// to be committed as part of a batch database transaction.
type preparedUpdate struct {
	seerrReq     seerr.SeerrRequest
	title        string
	posterPath   string
	releaseInfo  ReleaseInfo
	entry        database.TriageEntry
	isNew        bool
	updates      map[string]interface{}
	shouldNotify bool
}

func (s *Scanner) scan(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.InfoContext(ctx, "Starting scan cycle...")
	startTime := time.Now()

	requests, err := s.seerr.GetApprovedRequests(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching requests from Seerr", slog.Any("error", err))
		return err
	}

	slog.InfoContext(ctx, "Fetched approved requests", slog.Int("count", len(requests)))

	// Backfill requested seasons for older TV entries that don't have it set (including completed ones)
	var emptyTVEntries []database.TriageEntry
	if err := s.db.WithContext(ctx).Where("media_type = ? AND (requested_seasons = ? OR requested_seasons IS NULL)", "tv", "").Find(&emptyTVEntries).Error; err == nil && len(emptyTVEntries) > 0 {
		slog.InfoContext(ctx, "Backfilling requested seasons for older TV entries", slog.Int("count", len(emptyTVEntries)))
		for _, entry := range emptyTVEntries {
			seerrReq, err := s.seerr.GetRequest(ctx, entry.SeerrRequestID)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to fetch request details for backfill", slog.Int("requestId", entry.SeerrRequestID), slog.Any("error", err))
				continue
			}
			seasonsStr := formatSeasons(seerrReq.Seasons)
			updates := map[string]interface{}{
				"requested_seasons": seasonsStr,
				"is4_k":             seerrReq.Is4K,
			}
			if err := s.db.WithContext(ctx).Model(&entry).Updates(updates).Error; err != nil {
				slog.ErrorContext(ctx, "Failed to update requested seasons and is4k during backfill", slog.Int("requestId", entry.SeerrRequestID), slog.Any("error", err))
			}
		}
	}

	// Fetch all existing entries at once to populate our in-memory map
	var existingEntries []database.TriageEntry
	if err := s.db.WithContext(ctx).Find(&existingEntries).Error; err != nil {
		slog.ErrorContext(ctx, "Error fetching existing database entries", slog.Any("error", err))
		return err
	}
	existingMap := make(map[int]database.TriageEntry)
	for _, entry := range existingEntries {
		existingMap[entry.SeerrRequestID] = entry
	}

	// Track seen IDs for reconciliation (only for active requests)
	seenIDs := make(map[int]bool)
	for id, entry := range existingMap {
		if entry.Status != database.StatusCompleted {
			seenIDs[id] = false
		}
	}

	processed := 0
	var preparedUpdates []*preparedUpdate
	var dbActions []func(tx *gorm.DB) error

	for _, req := range requests {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Scan cycle interrupted by context cancellation")
			return ctx.Err()
		default:
		}

		seenIDs[req.ID] = true

		// Skip if actively downloading and not already in the database
		if len(req.Media.DownloadStatus) > 0 {
			_, exists := existingMap[req.ID]
			if !exists {
				continue
			}
		}

		prep, err := s.prepareRequestUpdate(ctx, req, existingMap)
		if err != nil {
			slog.ErrorContext(ctx, "Error processing request",
				slog.Int("requestId", req.ID),
				slog.Int("tmdbId", req.Media.TmdbID),
				slog.Any("error", err),
			)
			continue
		}
		preparedUpdates = append(preparedUpdates, prep)
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
					localID := id
					dbActions = append(dbActions, func(tx *gorm.DB) error {
						return tx.Where("seerr_request_id = ?", localID).Delete(&database.TriageEntry{}).Error
					})
					staleCount++
				} else {
					slog.ErrorContext(ctx, "Error fetching missing request from Seerr during reconciliation, skipping", slog.Int("requestId", id), slog.Any("error", err))
				}
				continue
			}

			// If the media status is completed, transition it to COMPLETED in database
			isCompleted := false
			if seerrReq.MediaType == "movie" {
				mediaStatusVal := seerrReq.Media.Status
				if seerrReq.Is4K {
					mediaStatusVal = seerrReq.Media.Status4k
				}
				isCompleted = (mediaStatusVal == 4 || mediaStatusVal == 5)
			} else if seerrReq.MediaType == "tv" {
				completedSeasons := make(map[int]bool)
				for _, ms := range seerrReq.Media.Seasons {
					statusVal := ms.Status
					if seerrReq.Is4K {
						statusVal = ms.Status4k
					}
					if statusVal == 5 {
						completedSeasons[ms.SeasonNumber] = true
					}
				}
				hasUncompleted := false
				for _, season := range seerrReq.Seasons {
					if !completedSeasons[season.SeasonNumber] {
						hasUncompleted = true
						break
					}
				}
				mediaStatusVal := seerrReq.Media.Status
				if seerrReq.Is4K {
					mediaStatusVal = seerrReq.Media.Status4k
				}
				isCompleted = (mediaStatusVal == 5 || (mediaStatusVal == 4 && !hasUncompleted))
			}

			if isCompleted {
				slog.InfoContext(ctx, "Reconciling missing request: media available in Seerr, marking as completed locally", slog.Int("requestId", id))
				fulfilledAt := parseTime(seerrReq.UpdatedAt)
				updates := map[string]interface{}{
					"status":       database.StatusCompleted,
					"fulfilled_at": &fulfilledAt,
				}
				localID := id
				dbActions = append(dbActions, func(tx *gorm.DB) error {
					return tx.Model(&database.TriageEntry{}).Where("seerr_request_id = ?", localID).Updates(updates).Error
				})
				completedCount++
			} else {
				// If it was declined or changed status to something not approved, delete it
				slog.InfoContext(ctx, "Reconciling missing request: request no longer approved in Seerr, deleting locally", slog.Int("requestId", id))
				localID := id
				dbActions = append(dbActions, func(tx *gorm.DB) error {
					return tx.Where("seerr_request_id = ?", localID).Delete(&database.TriageEntry{}).Error
				})
				staleCount++
			}
		}
	}

	// Prepare GORM actions for prepared updates
	for _, prep := range preparedUpdates {
		p := prep
		if p.isNew {
			dbActions = append(dbActions, func(tx *gorm.DB) error {
				return tx.Create(&p.entry).Error
			})
		} else {
			dbActions = append(dbActions, func(tx *gorm.DB) error {
				return tx.Model(&p.entry).Updates(p.updates).Error
			})
		}
	}

	// Prepare GORM action for last scan metadata
	lastScanTime := time.Now()
	dbActions = append(dbActions, func(tx *gorm.DB) error {
		return tx.Save(&database.SystemMetadata{
			Key:   "last_scan_at",
			Value: lastScanTime.Format(time.RFC3339),
		}).Error
	})

	// Execute transaction block to run all writes in a single commit (1 fsync)
	if len(dbActions) > 0 {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, action := range dbActions {
				if err := action(tx); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			slog.ErrorContext(ctx, "Error executing database update transaction", slog.Any("error", err))
			return err
		}
	}

	s.lastScan = lastScanTime

	// Send notifications *after* transaction commits successfully
	for _, prep := range preparedUpdates {
		if prep.shouldNotify {
			posterURL := ""
			if prep.posterPath != "" {
				posterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w300%s", prep.posterPath)
			}

			requestedBy := prep.seerrReq.RequestedBy.DisplayName
			if requestedBy == "" {
				requestedBy = "Unknown"
			}

			requestAge := time.Since(parseTime(prep.seerrReq.CreatedAt))

			if err := s.notifier.NotifyUnfulfilled(prep.title, prep.seerrReq.MediaType, posterURL, prep.entry.ServiceURL, prep.releaseInfo, requestedBy, requestAge); err != nil {
				slog.ErrorContext(ctx, "Notification error", slog.String("title", prep.title), slog.Any("error", err))
			} else {
				now := time.Now()
				if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).
					Where("seerr_request_id = ?", prep.seerrReq.ID).
					Update("notified_at", &now).Error; err != nil {
					slog.ErrorContext(ctx, "Error updating notified_at timestamp", slog.Int("requestId", prep.seerrReq.ID), slog.Any("error", err))
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

	slog.InfoContext(ctx, "Scan cycle complete",
		slog.Int("processed", processed),
		slog.Duration("duration", time.Since(startTime).Round(time.Millisecond)),
	)
	return nil
}

func (s *Scanner) prepareRequestUpdate(ctx context.Context, req seerr.SeerrRequest, existingMap map[int]database.TriageEntry) (*preparedUpdate, error) {
	tmdbID := req.Media.TmdbID
	mediaType := req.MediaType

	var title string
	var posterPath string
	var releaseInfo ReleaseInfo

	switch mediaType {
	case "movie":
		detail, err := s.seerr.GetMovieDetail(ctx, tmdbID)
		if err != nil {
			return nil, fmt.Errorf("fetching movie %d: %w", tmdbID, err)
		}
		title = detail.Title
		posterPath = detail.PosterPath
		releaseInfo = EvaluateMovieRelease(detail, s.cfg)

	case "tv":
		detail, err := s.seerr.GetTVDetail(ctx, tmdbID)
		if err != nil {
			return nil, fmt.Errorf("fetching TV %d: %w", tmdbID, err)
		}
		title = detail.Name
		posterPath = detail.PosterPath

		// Build map of completed seasons
		completedSeasons := make(map[int]bool)
		for _, ms := range req.Media.Seasons {
			statusVal := ms.Status
			if req.Is4K {
				statusVal = ms.Status4k
			}
			if statusVal == 5 { // 5 = Available
				completedSeasons[ms.SeasonNumber] = true
			}
		}

		// Extract requested season numbers that are not yet completed
		var requestedSeasons []int
		for _, season := range req.Seasons {
			if !completedSeasons[season.SeasonNumber] {
				requestedSeasons = append(requestedSeasons, season.SeasonNumber)
			}
		}
		releaseInfo = EvaluateTVRelease(detail, requestedSeasons)

	default:
		return nil, fmt.Errorf("unknown media type: %s", mediaType)
	}

	existing, exists := existingMap[req.ID]

	prepared := &preparedUpdate{
		seerrReq:    req,
		title:       title,
		posterPath:  posterPath,
		releaseInfo: releaseInfo,
	}

	isCompleted := false
	if mediaType == "movie" {
		mediaStatusVal := req.Media.Status
		if req.Is4K {
			mediaStatusVal = req.Media.Status4k
		}
		isCompleted = (mediaStatusVal == 4 || mediaStatusVal == 5)
	} else if mediaType == "tv" {
		// Build map of completed seasons again for safety
		completedSeasons := make(map[int]bool)
		for _, ms := range req.Media.Seasons {
			statusVal := ms.Status
			if req.Is4K {
				statusVal = ms.Status4k
			}
			if statusVal == 5 {
				completedSeasons[ms.SeasonNumber] = true
			}
		}
		hasUncompleted := false
		for _, season := range req.Seasons {
			if !completedSeasons[season.SeasonNumber] {
				hasUncompleted = true
				break
			}
		}
		mediaStatusVal := req.Media.Status
		if req.Is4K {
			mediaStatusVal = req.Media.Status4k
		}
		isCompleted = (mediaStatusVal == 5 || (mediaStatusVal == 4 && !hasUncompleted))
	}

	if !exists {
		entry := database.TriageEntry{
			SeerrRequestID: req.ID,
			MediaID:        req.Media.ID,
			TmdbID:         tmdbID,
			Is4K:           req.Is4K,
			PosterPath:     posterPath,
			MediaType:      mediaType,
			Title:          title,
			ReleaseDate:    releaseInfo.Date,
			ReleaseSource:  &releaseInfo.Source,
			SeerrCreatedAt: parseTime(req.CreatedAt),
			ServiceURL:     req.Media.ServiceURL,
		}
		if mediaType == "tv" {
			entry.RequestedSeasons = formatSeasons(req.Seasons)
		}

		if isCompleted {
			entry.Status = database.StatusCompleted
			fulfilledAt := parseTime(req.UpdatedAt)
			entry.FulfilledAt = &fulfilledAt
		} else if releaseInfo.Date != nil && !releaseInfo.IsSureReleased() || releaseInfo.IsUnreleased() {
			entry.Status = database.StatusWaitingRelease
		} else {
			entry.Status = database.StatusPending
		}

		prepared.entry = entry
		prepared.isNew = true
	} else {
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
			"is4_k":            req.Is4K,
		}
		if mediaType == "tv" {
			updates["requested_seasons"] = formatSeasons(req.Seasons)
		} else {
			updates["requested_seasons"] = ""
		}

		// Auto-transition to COMPLETED if fulfilled
		if isCompleted {
			updates["status"] = database.StatusCompleted
			if existing.FulfilledAt == nil {
				fulfilledAt := parseTime(req.UpdatedAt)
				updates["fulfilled_at"] = &fulfilledAt
			}
		} else {
			// If it was completed, but now it's no longer completed (re-requested/deleted)
			if existing.Status == database.StatusCompleted {
				if releaseInfo.Date != nil && !releaseInfo.IsSureReleased() || releaseInfo.IsUnreleased() {
					updates["status"] = database.StatusWaitingRelease
				} else {
					updates["status"] = database.StatusPending
				}
				updates["fulfilled_at"] = nil
				updates["notified_at"] = nil
			} else if existing.Status == database.StatusPending && (releaseInfo.Date != nil && !releaseInfo.IsSureReleased() || releaseInfo.IsUnreleased()) {
				updates["status"] = database.StatusWaitingRelease
			} else if existing.Status == database.StatusWaitingRelease && releaseInfo.IsSureReleased() && !releaseInfo.IsUnreleased() {
				updates["status"] = database.StatusPending
			}
		}

		prepared.entry = existing
		prepared.isNew = false
		prepared.updates = updates
	}

	// Determine shouldNotify using temporary updated structure in memory
	tempEntry := prepared.entry
	if !prepared.isNew {
		if status, ok := prepared.updates["status"].(string); ok {
			tempEntry.Status = status
		}
		if val, exists := prepared.updates["fulfilled_at"]; exists {
			if fulfilledAt, ok := val.(*time.Time); ok {
				tempEntry.FulfilledAt = fulfilledAt
			} else {
				tempEntry.FulfilledAt = nil
			}
		}
		if val, exists := prepared.updates["notified_at"]; exists {
			if notifiedAt, ok := val.(*time.Time); ok {
				tempEntry.NotifiedAt = notifiedAt
			} else {
				tempEntry.NotifiedAt = nil
			}
		}
	}

	prepared.shouldNotify = s.notifier != nil && s.shouldNotify(tempEntry, req)

	return prepared, nil
}

func (s *Scanner) processRequest(ctx context.Context, req seerr.SeerrRequest) error {
	existingMap := make(map[int]database.TriageEntry)
	var existing database.TriageEntry
	if err := s.db.WithContext(ctx).Where("seerr_request_id = ?", req.ID).First(&existing).Error; err == nil {
		existingMap[req.ID] = existing
	}

	prep, err := s.prepareRequestUpdate(ctx, req, existingMap)
	if err != nil {
		return err
	}

	// Apply database write immediately (for single-item execution compatibility in tests)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if prep.isNew {
			return tx.Create(&prep.entry).Error
		}
		return tx.Model(&prep.entry).Updates(prep.updates).Error
	})
	if err != nil {
		return err
	}

	// Trigger notification if needed
	if prep.shouldNotify {
		posterURL := ""
		if prep.posterPath != "" {
			posterURL = fmt.Sprintf("https://image.tmdb.org/t/p/w300%s", prep.posterPath)
		}

		requestedBy := prep.seerrReq.RequestedBy.DisplayName
		if requestedBy == "" {
			requestedBy = "Unknown"
		}

		requestAge := time.Since(parseTime(prep.seerrReq.CreatedAt))

		if err := s.notifier.NotifyUnfulfilled(prep.title, prep.seerrReq.MediaType, posterURL, prep.entry.ServiceURL, prep.releaseInfo, requestedBy, requestAge); err != nil {
			slog.ErrorContext(ctx, "Notification error", slog.String("title", prep.title), slog.Any("error", err))
		} else {
			now := time.Now()
			if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).
				Where("seerr_request_id = ?", prep.seerrReq.ID).
				Update("notified_at", &now).Error; err != nil {
				slog.ErrorContext(ctx, "Error updating notified_at timestamp", slog.Int("requestId", prep.seerrReq.ID), slog.Any("error", err))
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

	// Don't notify if actively downloading
	if len(req.Media.DownloadStatus) > 0 {
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

	// Check age window: request must be old enough (delay) but not too old (max age)
	requestTime := parseTime(req.CreatedAt)
	age := time.Since(requestTime)

	if age < s.cfg.AlertDelay {
		return false
	}
	if age > s.cfg.AlertMaxAge {
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

func formatSeasons(seasons []struct {
	SeasonNumber int `json:"seasonNumber"`
}) string {
	if len(seasons) == 0 {
		return ""
	}

	// Copy and sort
	nums := make([]int, len(seasons))
	for i, s := range seasons {
		nums[i] = s.SeasonNumber
	}
	sort.Ints(nums)

	var parts []string
	n := len(nums)
	for i := 0; i < n; {
		start := nums[i]
		end := start
		j := i + 1
		for j < n && nums[j] == end+1 {
			end = nums[j]
			j++
		}
		runLen := j - i
		if runLen >= 2 {
			parts = append(parts, fmt.Sprintf("S%d-%d", start, end))
		} else {
			parts = append(parts, fmt.Sprintf("S%d", start))
		}
		i = j
	}
	return strings.Join(parts, ", ")
}
