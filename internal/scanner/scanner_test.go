package scanner

import (
	"context"
	"encoding/json"
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

	err = db.AutoMigrate(&database.TriageEntry{}, &database.SystemMetadata{})
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
		AlertThreshold:    10 * time.Minute,
		AlertWindow:       10 * time.Minute,
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

