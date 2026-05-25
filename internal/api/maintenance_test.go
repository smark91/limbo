package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/seerr"
)

func TestCleanOlderHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with mock entries
	now := time.Now()
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 101,
			MediaID:        201,
			TmdbID:         301,
			Title:          "Old Pending Movie",
			MediaType:      "movie",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-48 * time.Hour), // Older than 24h
		},
		{
			SeerrRequestID: 102,
			MediaID:        202,
			TmdbID:         302,
			Title:          "New Pending Movie",
			MediaType:      "movie",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-12 * time.Hour), // Newer than 24h
		},
		{
			SeerrRequestID: 103,
			MediaID:        203,
			TmdbID:         303,
			Title:          "Old Unavailable TV",
			MediaType:      "tv",
			Status:         database.StatusNotAvailable,
			SeerrCreatedAt: now.Add(-48 * time.Hour), // Older than 24h
		},
		{
			SeerrRequestID: 104,
			MediaID:        204,
			TmdbID:         304,
			Title:          "Old Completed Movie",
			MediaType:      "movie",
			Status:         database.StatusCompleted,
			SeerrCreatedAt: now.Add(-48 * time.Hour), // Older than 24h (but completed!)
		},
	}

	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to seed entry: %v", err)
		}
	}

	// Set up mock HTTP server for Seerr API
	deletedRequests := make(map[int]bool)
	deletedMedia := make(map[int]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			var requestID, mediaID int
			if _, err := fmt.Sscanf(r.URL.Path, "/api/v1/request/%d", &requestID); err == nil {
				deletedRequests[requestID] = true
				w.WriteHeader(http.StatusOK)
				return
			}
			if _, err := fmt.Sscanf(r.URL.Path, "/api/v1/media/%d", &mediaID); err == nil {
				deletedMedia[mediaID] = true
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-api-key",
	}
	seerrClient := seerr.NewClient(cfg)

	// Test 1: Clean older requests (Older than 24 hours, PENDING only)
	t.Run("Clean older requests - Pending only", func(t *testing.T) {
		// Clean older than 24 hours ago
		olderThanTime := now.Add(-24 * time.Hour)
		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": olderThanTime.Format(time.RFC3339),
			"statuses":  []string{database.StatusPending},
		})

		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["success"] != true {
			t.Errorf("expected success to be true, got %v", resp["success"])
		}

		// Since only 101 matches the "older than 24h AND PENDING" condition, deletedCount should be 1
		if count, ok := resp["deletedCount"].(float64); !ok || count != 1 {
			t.Errorf("expected deletedCount to be 1, got %v", resp["deletedCount"])
		}

		// Verify entry 101 is gone from DB
		var checkEntry101 database.TriageEntry
		if err := db.Where("seerr_request_id = ?", 101).First(&checkEntry101).Error; err == nil {
			t.Errorf("expected request 101 to be deleted from database, but it still exists")
		}

		// Verify entry 102 (new pending) is still in DB
		var checkEntry102 database.TriageEntry
		if err := db.Where("seerr_request_id = ?", 102).First(&checkEntry102).Error; err != nil {
			t.Errorf("expected request 102 to still exist, got error: %v", err)
		}

		// Verify entry 103 (old not_available) is still in DB (since we only specified PENDING)
		var checkEntry103 database.TriageEntry
		if err := db.Where("seerr_request_id = ?", 103).First(&checkEntry103).Error; err != nil {
			t.Errorf("expected request 103 to still exist, got error: %v", err)
		}

		// Verify Seerr calls
		if !deletedRequests[101] {
			t.Errorf("expected request 101 to be deleted from Seerr API")
		}
		if !deletedMedia[201] {
			t.Errorf("expected media 201 to be deleted from Seerr API")
		}
	})

	// Test 2: Clean older requests (Older than 24 hours, NOT_AVAILABLE)
	t.Run("Clean older requests - Not Available", func(t *testing.T) {
		olderThanTime := now.Add(-24 * time.Hour)
		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": olderThanTime.Format(time.RFC3339),
			"statuses":  []string{database.StatusNotAvailable},
		})

		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if count, ok := resp["deletedCount"].(float64); !ok || count != 1 {
			t.Errorf("expected deletedCount to be 1, got %v", resp["deletedCount"])
		}

		// Verify entry 103 (old unavailable) is gone from DB
		var checkEntry database.TriageEntry
		if err := db.Where("seerr_request_id = ?", 103).First(&checkEntry).Error; err == nil {
			t.Errorf("expected request 103 to be deleted from database, but it still exists")
		}

		// Verify Seerr call
		if !deletedRequests[103] {
			t.Errorf("expected request 103 to be deleted from Seerr API")
		}
	})

	// Test 3: Attempt to clean COMPLETED status
	t.Run("Reject Completed Status", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": now.Format(time.RFC3339),
			"statuses":  []string{database.StatusCompleted},
		})

		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected bad request 400 when attempting to delete completed requests, got %d", rr.Code)
		}
	})
}

func TestRefreshCacheHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with mock entries
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 201,
			TmdbID:         12345,
			Title:          "Old Title Movie",
			MediaType:      "movie",
			Status:         database.StatusPending,
		},
		{
			SeerrRequestID: 202,
			TmdbID:         67890,
			Title:          "Old Title TV",
			MediaType:      "tv",
			Status:         database.StatusWaitingRelease,
		},
	}

	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to seed entry: %v", err)
		}
	}

	// Set up mock HTTP server for Seerr movie/tv details API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.URL.Path == "/api/v1/movie/12345" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"id": 12345,
					"title": "Refreshed Movie Title",
					"posterPath": "/new_poster_movie.jpg",
					"releaseDate": "2026-06-01"
				}`))
				return
			}
			if r.URL.Path == "/api/v1/tv/67890" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"id": 67890,
					"name": "Refreshed TV Title",
					"posterPath": "/new_poster_tv.jpg",
					"firstAirDate": "2026-07-01",
					"seasons": [
						{"seasonNumber": 1, "airDate": "2026-07-01"}
					]
				}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:       server.URL,
		SeerrAPIKey:    "test-api-key",
		ReleaseCountry: "US",
	}
	seerrClient := seerr.NewClient(cfg)

	req, _ := http.NewRequest("POST", "/api/maintenance/refresh-cache", nil)
	rr := httptest.NewRecorder()

	handler := handleRefreshCache(db, seerrClient, cfg)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success to be true, got %v", resp["success"])
	}

	if count, ok := resp["refreshedCount"].(float64); !ok || count != 2 {
		t.Errorf("expected refreshedCount to be 2, got %v", resp["refreshedCount"])
	}

	// Verify entries were updated in database
	var refreshedMovie database.TriageEntry
	if err := db.Where("seerr_request_id = ?", 201).First(&refreshedMovie).Error; err != nil {
		t.Fatalf("failed to query refreshed movie: %v", err)
	}
	if refreshedMovie.Title != "Refreshed Movie Title" {
		t.Errorf("expected movie title 'Refreshed Movie Title', got %q", refreshedMovie.Title)
	}
	if refreshedMovie.PosterPath != "/new_poster_movie.jpg" {
		t.Errorf("expected movie poster '/new_poster_movie.jpg', got %q", refreshedMovie.PosterPath)
	}

	var refreshedTV database.TriageEntry
	if err := db.Where("seerr_request_id = ?", 202).First(&refreshedTV).Error; err != nil {
		t.Fatalf("failed to query refreshed TV: %v", err)
	}
	if refreshedTV.Title != "Refreshed TV Title" {
		t.Errorf("expected TV title 'Refreshed TV Title', got %q", refreshedTV.Title)
	}
	if refreshedTV.PosterPath != "/new_poster_tv.jpg" {
		t.Errorf("expected TV poster '/new_poster_tv.jpg', got %q", refreshedTV.PosterPath)
	}
}

func TestGetCacheInfoHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with mock entries
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 301,
			Title:          "Movie 1",
			MediaType:      "movie",
			Status:         database.StatusPending,
		},
		{
			SeerrRequestID: 302,
			Title:          "Movie 2",
			MediaType:      "movie",
			Status:         database.StatusWaitingRelease,
		},
		{
			SeerrRequestID: 303,
			Title:          "Movie 3",
			MediaType:      "movie",
			Status:         database.StatusCompleted, // Completed, so not active
		},
	}

	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to seed entry: %v", err)
		}
	}

	cfg := &config.Config{
		DBDriver: "sqlite",
	}

	req, _ := http.NewRequest("GET", "/api/maintenance/cache", nil)
	rr := httptest.NewRecorder()

	handler := handleGetCacheInfo(db, cfg)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp cacheInfoResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ActiveCount != 2 {
		t.Errorf("expected activeCount to be 2, got %d", resp.ActiveCount)
	}

	if resp.TotalCount != 3 {
		t.Errorf("expected totalCount to be 3, got %d", resp.TotalCount)
	}

	if resp.Size <= 0 {
		t.Errorf("expected database size to be greater than 0, got %d", resp.Size)
	}
}
