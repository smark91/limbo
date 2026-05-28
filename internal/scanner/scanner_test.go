package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/seerr"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&database.TriageEntry{}, &database.SystemMetadata{}, &database.PushSubscription{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestRevertCompletedRequest(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with a COMPLETED request
	now := time.Now()
	initialEntry := database.TriageEntry{
		SeerrRequestID: 999,
		MediaID:        888,
		TmdbID:         777,
		Title:          "Deleted Movie Re-requested",
		MediaType:      "movie",
		Status:         database.StatusCompleted,
		FulfilledAt:    &now,
		NotifiedAt:     &now,
		SeerrCreatedAt: now.Add(-24 * time.Hour),
	}
	if err := db.Create(&initialEntry).Error; err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	// Mock Seerr API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/777" {
			movieDetail := map[string]interface{}{
				"id":          777,
				"title":       "Deleted Movie Re-requested",
				"posterPath":  "/test.jpg",
				"status":      "Released",
				"releaseDate": "2020-01-01",
				"releases": map[string]interface{}{
					"releases": []interface{}{
						map[string]interface{}{
							"iso_3166_1": "US",
							"release_dates": []interface{}{
								map[string]interface{}{
									"type":         4,
									"release_date": "2020-01-01T00:00:00.000Z",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:          server.URL,
		SeerrAPIKey:       "test-key",
		ScanInterval:      10 * time.Minute,
		AlertDelay:        10 * time.Minute,
		AlertMaxAge:       10 * time.Minute,
		DiscordWebhookURL: "http://mock-webhook",
	}
	seerrClient := seerr.NewClient(cfg)
	s := New(cfg, db, seerrClient)

	// Mock SeerrRequest with media status 2 (pending download)
	req := seerr.SeerrRequest{
		ID:        999,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:     888,
			TmdbID: 777,
			Status: 2, // Pending/Processing (NOT completed!)
		},
		CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	// Execute processRequest
	ctx := context.Background()
	err := s.processRequest(ctx, req)
	if err != nil {
		t.Fatalf("processRequest failed: %v", err)
	}

	// Assert database entry is reverted to PENDING and timestamps cleared
	var updatedEntry database.TriageEntry
	if err := db.First(&updatedEntry, "seerr_request_id = ?", 999).Error; err != nil {
		t.Fatalf("failed to query updated entry: %v", err)
	}

	if updatedEntry.Status != database.StatusPending {
		t.Errorf("expected status %s, got %s", database.StatusPending, updatedEntry.Status)
	}
	if updatedEntry.FulfilledAt != nil {
		t.Errorf("expected FulfilledAt to be nil, got %v", updatedEntry.FulfilledAt)
	}
	if updatedEntry.NotifiedAt != nil {
		t.Errorf("expected NotifiedAt to be nil, got %v", updatedEntry.NotifiedAt)
	}
}

func TestReconcileMissingRequest(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with two active requests
	now := time.Now()
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 991,
			MediaID:        881,
			TmdbID:         771,
			Title:          "To Be Completed Movie",
			MediaType:      "movie",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-24 * time.Hour),
		},
		{
			SeerrRequestID: 992,
			MediaID:        882,
			TmdbID:         772,
			Title:          "To Be Deleted Movie",
			MediaType:      "movie",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-24 * time.Hour),
		},
	}
	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to seed entry: %v", err)
		}
	}

	// Mock Seerr API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// 1. Fetch approved requests returns empty (missing both 991 and 992)
		if r.URL.Path == "/api/v1/request" {
			res := map[string]interface{}{
				"pageInfo": map[string]interface{}{
					"pages":   1,
					"page":    1,
					"results": 0,
				},
				"results": []interface{}{},
			}
			json.NewEncoder(w).Encode(res)
			return
		}

		// 2. Fetch specific request 991 (media is available/completed)
		if r.URL.Path == "/api/v1/request/991" {
			reqDetail := map[string]interface{}{
				"id":        991,
				"status":    2, // Approved
				"type":      "movie",
				"createdAt": now.Add(-24 * time.Hour).Format(time.RFC3339),
				"updatedAt": now.Format(time.RFC3339),
				"media": map[string]interface{}{
					"id":        881,
					"tmdbId":    771,
					"status":    5, // Available
					"mediaType": "movie",
				},
			}
			json.NewEncoder(w).Encode(reqDetail)
			return
		}

		// 3. Fetch specific request 992 (deleted in Seerr -> returns 404)
		if r.URL.Path == "/api/v1/request/992" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Request not found"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:     server.URL,
		SeerrAPIKey:  "test-key",
		ScanInterval: 10 * time.Minute,
	}
	seerrClient := seerr.NewClient(cfg)
	s := New(cfg, db, seerrClient)

	// Trigger a scan cycle to reconcile
	ctx := context.Background()
	s.scan(ctx)

	// Assert 991 is now marked as COMPLETED
	var completedEntry database.TriageEntry
	if err := db.First(&completedEntry, "seerr_request_id = ?", 991).Error; err != nil {
		t.Fatalf("failed to query completed entry 991: %v", err)
	}
	if completedEntry.Status != database.StatusCompleted {
		t.Errorf("expected status for 991 to be COMPLETED, got %s", completedEntry.Status)
	}
	if completedEntry.FulfilledAt == nil {
		t.Errorf("expected FulfilledAt for 991 to be set, got nil")
	}

	// Assert 992 is deleted from Limbo's database
	var deletedEntry database.TriageEntry
	err := db.First(&deletedEntry, "seerr_request_id = ?", 992).Error
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected 992 to be deleted, but found entry with status %s", deletedEntry.Status)
	}
}


func TestScannerProcessTVRequest(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()

	// Mock Seerr API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/tv/100" {
			tvDetail := map[string]interface{}{
				"id":           100,
				"name":         "Futurama",
				"posterPath":   "/futurama.jpg",
				"firstAirDate": "1999-03-28",
				"seasons": []map[string]interface{}{
					{
						"seasonNumber": 1,
						"airDate":      "1999-03-28",
					},
				},
			}
			json.NewEncoder(w).Encode(tvDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        50,
		Status:    2,
		MediaType: "tv",
		Media: seerr.Media{
			ID:     60,
			TmdbID: 100,
			Status: 2,
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}
	req.Seasons = append(req.Seasons, struct {
		SeasonNumber int `json:"seasonNumber"`
	}{SeasonNumber: 1})

	ctx := context.Background()
	err := s.processRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to process TV request: %v", err)
	}

	var entry database.TriageEntry
	if err := db.First(&entry, "seerr_request_id = ?", 50).Error; err != nil {
		t.Fatalf("failed to query entry: %v", err)
	}

	if entry.Title != "Futurama" || entry.MediaType != "tv" || entry.PosterPath != "/futurama.jpg" {
		t.Errorf("incorrect TV triage entry saved: %+v", entry)
	}
}

func TestScannerSkipActiveDownload(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        1,
		MediaType: "movie",
		Media: seerr.Media{
			ID:             2,
			TmdbID:         3,
			DownloadStatus: []interface{}{"downloading"}, // Active download
		},
	}

	// This should run the loop, see the downloadStatus, and skip processing
	ctx := context.Background()
	s.processedRequestsMock(ctx, []seerr.SeerrRequest{req})

	var count int64
	db.Model(&database.TriageEntry{}).Count(&count)
	if count != 0 {
		t.Errorf("expected request to be skipped, but got DB entry count=%d", count)
	}
}

// processedRequestsMock is a helper to run the loop with predefined requests list
func (s *Scanner) processedRequestsMock(ctx context.Context, requests []seerr.SeerrRequest) {
	for _, req := range requests {
		if len(req.Media.DownloadStatus) > 0 {
			var exists int64
			if err := s.db.WithContext(ctx).Model(&database.TriageEntry{}).Where("seerr_request_id = ?", req.ID).Count(&exists).Error; err == nil && exists == 0 {
				continue
			}
		}
		_ = s.processRequest(ctx, req)
	}
}

func TestScannerProcessRequestWaitingRelease(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()
	futureDate := now.Add(240 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/200" {
			movieDetail := map[string]interface{}{
				"id":          200,
				"title":       "Future Movie",
				"releaseDate": futureDate.Format("2006-01-02"),
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        100,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:     200,
			TmdbID: 200,
			Status: 2,
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	err := s.processRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	var entry database.TriageEntry
	db.First(&entry, "seerr_request_id = ?", 100)
	if entry.Status != database.StatusWaitingRelease {
		t.Errorf("expected status 'WAITING_RELEASE', got %q", entry.Status)
	}
}

func TestScannerErrors(t *testing.T) {
	db := setupTestDB(t)
	// Cancelled context should prevent request processing
	cfg := &config.Config{}
	s := New(cfg, db, seerr.NewClient(cfg))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run scan with cancelled context
	s.scan(ctx)

	// Should not panic, and not have processed anything
	var count int64
	db.Model(&database.TriageEntry{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 entries with cancelled context, got %d", count)
	}
}

func TestNewScannerWithLastScan(t *testing.T) {
	db := setupTestDB(t)

	// Save last scan time in DB
	lastScanTime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	db.Create(&database.SystemMetadata{
		Key:   "last_scan_at",
		Value: lastScanTime.Format(time.RFC3339),
	})

	cfg := &config.Config{}
	s := New(cfg, db, seerr.NewClient(cfg))

	if !s.LastScanTime().Equal(lastScanTime) {
		t.Errorf("expected last scan time %v, got %v", lastScanTime, s.LastScanTime())
	}
}

func TestScannerAutoResolve(t *testing.T) {
	db := setupTestDB(t)
	entry := database.TriageEntry{
		SeerrRequestID: 99,
		Status:         database.StatusPending,
	}
	db.Create(&entry)

	cfg := &config.Config{}
	s := New(cfg, db, seerr.NewClient(cfg))
	s.autoResolve(context.Background(), seerr.SeerrRequest{ID: 99})

	var check database.TriageEntry
	err := db.First(&check, "seerr_request_id = ?", 99).Error
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected entry 99 to be deleted, got: %v", err)
	}

	t.Run("DB Failure", func(t *testing.T) {
		dbFail := setupTestDB(t)
		dbFail.Create(&database.TriageEntry{SeerrRequestID: 99, Status: database.StatusPending})
		dbFail.Callback().Delete().Before("gorm:delete").Register("fail_delete", func(d *gorm.DB) {
			d.AddError(errors.New("mocked delete error"))
		})
		sFail := New(cfg, dbFail, seerr.NewClient(cfg))
		sFail.autoResolve(context.Background(), seerr.SeerrRequest{ID: 99})
	})
}

func TestParseTimePrivate(t *testing.T) {
	t.Run("Standard RFC3339Nano", func(t *testing.T) {
		got := parseTime("2026-04-15T14:59:24.056Z")
		if got.Year() != 2026 || got.Month() != 4 || got.Day() != 15 {
			t.Errorf("unexpected parse result: %v", got)
		}
	})

	t.Run("Simpler ISO format fallback", func(t *testing.T) {
		got := parseTime("2026-04-15T14:59:24.000Z")
		if got.Year() != 2026 || got.Month() != 4 || got.Day() != 15 {
			t.Errorf("unexpected parse result: %v", got)
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		got := parseTime("invalid-time-string")
		if !got.IsZero() {
			t.Errorf("expected zero time for invalid format, got %v", got)
		}
	})
}

func TestScannerWaitingReleaseToPending(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()

	// Seed database with a WAITING_RELEASE request
	initialEntry := database.TriageEntry{
		SeerrRequestID: 300,
		MediaID:        400,
		TmdbID:         500,
		Title:          "Released Movie",
		MediaType:      "movie",
		Status:         database.StatusWaitingRelease,
		SeerrCreatedAt: now.Add(-24 * time.Hour),
	}
	if err := db.Create(&initialEntry).Error; err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	// Mock Seerr API Server returning a release date in the past
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/500" {
			movieDetail := map[string]interface{}{
				"id":          500,
				"title":       "Released Movie",
				"posterPath":  "/released.jpg",
				"status":      "Released",
				"releaseDate": "2020-01-01",
				"releases": map[string]interface{}{
					"releases": []interface{}{
						map[string]interface{}{
							"iso_3166_1": "US",
							"release_dates": []interface{}{
								map[string]interface{}{
									"type":         4,
									"release_date": "2020-01-01T00:00:00.000Z",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        300,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:     400,
			TmdbID: 500,
			Status: 2, // Approved/Pending
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	err := s.processRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to process request: %v", err)
	}

	var entry database.TriageEntry
	if err := db.First(&entry, "seerr_request_id = ?", 300).Error; err != nil {
		t.Fatalf("failed to query entry: %v", err)
	}

	if entry.Status != database.StatusPending {
		t.Errorf("expected status 'PENDING' when release date passes, got %q", entry.Status)
	}
}

func TestScannerWaitingReleaseNoDateStaysWaiting(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()

	// Seed database with a WAITING_RELEASE request
	initialEntry := database.TriageEntry{
		SeerrRequestID: 301,
		MediaID:        401,
		TmdbID:         501,
		Title:          "Unknown Release Date Movie",
		MediaType:      "movie",
		Status:         database.StatusWaitingRelease,
		SeerrCreatedAt: now.Add(-24 * time.Hour),
	}
	if err := db.Create(&initialEntry).Error; err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	// Mock Seerr API Server returning no release date
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/501" {
			movieDetail := map[string]interface{}{
				"id":          501,
				"title":       "Unknown Release Date Movie",
				"posterPath":  "/unknown.jpg",
				"status":      "In Production",
				"releaseDate": "", // No release date
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        301,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:     401,
			TmdbID: 501,
			Status: 2,
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	err := s.processRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to process request: %v", err)
	}

	var entry database.TriageEntry
	if err := db.First(&entry, "seerr_request_id = ?", 301).Error; err != nil {
		t.Fatalf("failed to query entry: %v", err)
	}

	if entry.Status != database.StatusWaitingRelease {
		t.Errorf("expected status to remain 'WAITING_RELEASE' when release date is unknown, got %q", entry.Status)
	}
}

func TestScannerPastTheatricalStaysWaiting(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()

	// 1. Test existing WAITING_RELEASE entry
	// Seed database with a WAITING_RELEASE request
	initialEntry := database.TriageEntry{
		SeerrRequestID: 302,
		MediaID:        402,
		TmdbID:         502,
		Title:          "Past Theatrical Movie",
		MediaType:      "movie",
		Status:         database.StatusWaitingRelease,
		SeerrCreatedAt: now.Add(-24 * time.Hour),
	}
	if err := db.Create(&initialEntry).Error; err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	// Mock Seerr API Server returning only a past theatrical release date (no digital/physical)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/502" {
			movieDetail := map[string]interface{}{
				"id":          502,
				"title":       "Past Theatrical Movie",
				"posterPath":  "/theatrical.jpg",
				"status":      "Released",
				"releaseDate": "2020-01-01", // Fallback -> Theatrical
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        302,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:     402,
			TmdbID: 502,
			Status: 2,
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	err := s.processRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to process request: %v", err)
	}

	var entry database.TriageEntry
	if err := db.First(&entry, "seerr_request_id = ?", 302).Error; err != nil {
		t.Fatalf("failed to query entry: %v", err)
	}

	if entry.Status != database.StatusWaitingRelease {
		t.Errorf("expected status to remain 'WAITING_RELEASE' when release is theatrical fallback, got %q", entry.Status)
	}
}

func TestScannerActiveDownloadExistingEntry(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()

	// Seed database with a WAITING_RELEASE request
	initialEntry := database.TriageEntry{
		SeerrRequestID: 303,
		MediaID:        403,
		TmdbID:         503,
		Title:          "Downloading Movie",
		MediaType:      "movie",
		Status:         database.StatusWaitingRelease,
		SeerrCreatedAt: now.Add(-24 * time.Hour),
	}
	if err := db.Create(&initialEntry).Error; err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	// Mock Seerr API Server returning a release date in the past
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/movie/503" {
			movieDetail := map[string]interface{}{
				"id":          503,
				"title":       "Downloading Movie",
				"posterPath":  "/released.jpg",
				"status":      "Released",
				"releaseDate": "2020-01-01",
				"releases": map[string]interface{}{
					"releases": []interface{}{
						map[string]interface{}{
							"iso_3166_1": "US",
							"release_dates": []interface{}{
								map[string]interface{}{
									"type":         4,
									"release_date": "2020-01-01T00:00:00.000Z",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(movieDetail)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}
	s := New(cfg, db, seerr.NewClient(cfg))

	req := seerr.SeerrRequest{
		ID:        303,
		Status:    2,
		MediaType: "movie",
		Media: seerr.Media{
			ID:             403,
			TmdbID:         503,
			Status:         2, // Approved/Pending
			DownloadStatus: []interface{}{"downloading"},
		},
		CreatedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}

	// Run scan
	s.processedRequestsMock(context.Background(), []seerr.SeerrRequest{req})

	var entry database.TriageEntry
	if err := db.First(&entry, "seerr_request_id = ?", 303).Error; err != nil {
		t.Fatalf("failed to query entry: %v", err)
	}

	if entry.Status != database.StatusPending {
		t.Errorf("expected status 'PENDING' when release date passes and it starts downloading, got %q", entry.Status)
	}
}




