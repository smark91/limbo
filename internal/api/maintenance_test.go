package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"
	"github.com/smark91/limbo/internal/scanner"
	"github.com/smark91/limbo/internal/seerr"

	"gorm.io/gorm"
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
			Status:         database.StatusUnavailable,
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
	t.Run("Clean older requests - Unavailable", func(t *testing.T) {
		olderThanTime := now.Add(-24 * time.Hour)
		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": olderThanTime.Format(time.RFC3339),
			"statuses":  []string{database.StatusUnavailable},
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

func TestTestNotificationHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with mock entries
	now := time.Now()
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 501,
			Title:          "Latest Movie Request",
			MediaType:      "movie",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-10 * time.Hour),
		},
		{
			SeerrRequestID: 502,
			Title:          "Latest TV Request",
			MediaType:      "tv",
			Status:         database.StatusPending,
			SeerrCreatedAt: now.Add(-5 * time.Hour),
		},
	}

	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to seed entry: %v", err)
		}
	}

	// Set up mock HTTP server for Seerr and Discord webhook
	var receivedWebhookPayload map[string]interface{}
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&receivedWebhookPayload)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer discordServer.Close()

	seerrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/request/501" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 501,
				"type": "movie",
				"requestedBy": {"displayName": "Movie Requester"}
			}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/v1/request/502" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 502,
				"type": "tv",
				"requestedBy": {"displayName": "TV Requester"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer seerrServer.Close()

	cfg := &config.Config{
		DBDriver:          "sqlite",
		DiscordWebhookURL: discordServer.URL,
		SeerrURL:          seerrServer.URL,
		SeerrAPIKey:       "test-api-key",
	}

	seerrClient := seerr.NewClient(cfg)
	scannerInstance := scanner.New(cfg, db, seerrClient)

	// Sub-test 1: System Test Notification
	t.Run("System Test Notification", func(t *testing.T) {
		receivedWebhookPayload = nil
		reqBody, _ := json.Marshal(map[string]string{"type": "system"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, scannerInstance, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		if receivedWebhookPayload == nil {
			t.Errorf("expected webhook notification to be sent to Discord")
		} else {
			embeds := receivedWebhookPayload["embeds"].([]interface{})
			embed := embeds[0].(map[string]interface{})
			if embed["title"] != "🎬 System Test Notification" {
				t.Errorf("expected embed title '🎬 System Test Notification', got %v", embed["title"])
			}
		}
	})

	// Sub-test 2: Latest Movie Test Notification
	t.Run("Latest Movie Notification", func(t *testing.T) {
		receivedWebhookPayload = nil
		reqBody, _ := json.Marshal(map[string]string{"type": "movie"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, scannerInstance, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		if receivedWebhookPayload == nil {
			t.Errorf("expected webhook notification to be sent to Discord")
		} else {
			embeds := receivedWebhookPayload["embeds"].([]interface{})
			embed := embeds[0].(map[string]interface{})
			if embed["title"] != "🎬 Latest Movie Request" {
				t.Errorf("expected embed title '🎬 Latest Movie Request', got %v", embed["title"])
			}
			fields := embed["fields"].([]interface{})
			found := false
			for _, f := range fields {
				fMap := f.(map[string]interface{})
				if fMap["name"] == "Requested By" && fMap["value"] == "Movie Requester" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected field 'Requested By' with value 'Movie Requester'")
			}
		}
	})

	// Sub-test 3: Webhook unconfigured rejection
	t.Run("Unconfigured Webhook", func(t *testing.T) {
		emptyCfg := &config.Config{
			DBDriver: "sqlite",
		}
		emptyScanner := scanner.New(emptyCfg, db, seerrClient)
		reqBody, _ := json.Marshal(map[string]string{"type": "system"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, emptyScanner, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 when webhook is unconfigured, got %d", rr.Code)
		}
	})

	// Sub-test 3.5: VAPID Configured Only (Discord unconfigured)
	t.Run("VAPID Configured Only", func(t *testing.T) {
		vapidCfg := &config.Config{
			DBDriver:        "sqlite",
			VapidPublicKey:  "test-public-key",
			VapidPrivateKey: "test-private-key",
		}
		vapidScanner := scanner.New(vapidCfg, db, seerrClient)
		reqBody, _ := json.Marshal(map[string]string{"type": "system"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, vapidScanner, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200 when only VAPID is configured, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	// Sub-test 4: Invalid JSON body
	t.Run("Invalid JSON body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer([]byte("{invalid")))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, scannerInstance, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// Sub-test 5: Invalid test type
	t.Run("Invalid test type", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{"type": "invalid_type"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, scannerInstance, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// Sub-test 6: Target movie/tv not found in DB
	t.Run("Movie/TV not found in DB", func(t *testing.T) {
		// Create a separate db without seeding movie/tv requests
		emptyDb := setupTestDB(t)
		emptyScanner := scanner.New(cfg, emptyDb, seerrClient)

		reqBody, _ := json.Marshal(map[string]string{"type": "movie"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(emptyDb, emptyScanner, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	// Sub-test 7: Seerr API error on movie/tv request fetch (fails gracefully to Seerr User)
	t.Run("Seerr API error on request DisplayName fetch", func(t *testing.T) {
		// Mock Seerr server returns error on 501
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer errServer.Close()

		errCfg := &config.Config{
			DBDriver:          "sqlite",
			DiscordWebhookURL: discordServer.URL,
			SeerrURL:          errServer.URL,
			SeerrAPIKey:       "test-api-key",
		}
		errSeerrClient := seerr.NewClient(errCfg)
		errScanner := scanner.New(errCfg, db, errSeerrClient)

		reqBody, _ := json.Marshal(map[string]string{"type": "movie"})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handleTestNotification(db, errScanner, errSeerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("Database error in test notification", func(t *testing.T) {
		db := setupTestDB(t)
		db.Callback().Query().Before("gorm:query").Register("fail_query", func(d *gorm.DB) {
			d.AddError(errors.New("mocked error"))
		})

		cfg := &config.Config{DiscordWebhookURL: "http://discord-webhook.com"}
		sc := scanner.New(cfg, db, seerr.NewClient(cfg))

		reqBody, _ := json.Marshal(map[string]interface{}{
			"type": "movie",
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/test-notification", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleTestNotification(db, sc, seerr.NewClient(cfg))
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 status on database failure, got %d", rr.Code)
		}
	})
}

func TestCleanOlderEdgeCases(t *testing.T) {
	db := setupTestDB(t)

	// Mock Seerr server returning error to test error resilience
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errServer.Close()
	seerrClient := seerr.NewClient(&config.Config{SeerrURL: errServer.URL})

	t.Run("Invalid JSON body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer([]byte("{invalid")))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Missing olderThan date", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"statuses": []string{"PENDING"},
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Invalid status", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": time.Now().Format(time.RFC3339),
			"statuses":  []string{"INVALID_STATUS"},
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Empty statuses defaults to PENDING", func(t *testing.T) {
		// Seed old pending entry
		db.Create(&database.TriageEntry{
			SeerrRequestID: 900,
			Status:         database.StatusPending,
			SeerrCreatedAt: time.Now().Add(-48 * time.Hour),
		})

		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["deletedCount"].(float64) != 1 {
			t.Errorf("expected 1 deleted count, got %v", resp["deletedCount"])
		}
	})

	t.Run("Database Failures", func(t *testing.T) {
		dbClosed := setupTestDB(t)
		sqlDB, _ := dbClosed.DB()
		sqlDB.Close()

		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": time.Now().Format(time.RFC3339),
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(dbClosed, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("Delete failure in clean-older", func(t *testing.T) {
		db := setupTestDB(t)
		db.Create(&database.TriageEntry{
			SeerrRequestID: 100,
			Status:         database.StatusPending,
			SeerrCreatedAt: time.Now().Add(-48 * time.Hour),
		})
		db.Callback().Delete().Before("gorm:delete").Register("fail_delete", func(d *gorm.DB) {
			d.AddError(errors.New("mocked delete error"))
		})

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		seerrClient := seerr.NewClient(&config.Config{SeerrURL: server.URL})

		reqBody, _ := json.Marshal(map[string]interface{}{
			"olderThan": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		})
		req, _ := http.NewRequest("POST", "/api/maintenance/clean-older", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler := handleCleanOlder(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["deletedCount"].(float64) != 0 {
			t.Errorf("expected 0 deleted count due to delete failure, got %v", resp["deletedCount"])
		}
	})
}

func TestRefreshCacheEdgeCases(t *testing.T) {
	t.Run("Database Failures", func(t *testing.T) {
		dbClosed := setupTestDB(t)
		sqlDB, _ := dbClosed.DB()
		sqlDB.Close()

		cfg := &config.Config{}
		seerrClient := seerr.NewClient(cfg)

		req, _ := http.NewRequest("POST", "/api/maintenance/refresh-cache", nil)
		rr := httptest.NewRecorder()
		handler := handleRefreshCache(dbClosed, seerrClient, cfg)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("Seerr API Failure handles gracefully", func(t *testing.T) {
		db := setupTestDB(t)
		db.Create(&database.TriageEntry{SeerrRequestID: 1, TmdbID: 10, MediaType: "movie", Status: "PENDING"})
		db.Create(&database.TriageEntry{SeerrRequestID: 2, TmdbID: 20, MediaType: "tv", Status: "PENDING"})

		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer errServer.Close()

		cfg := &config.Config{}
		seerrClient := seerr.NewClient(&config.Config{SeerrURL: errServer.URL})

		req, _ := http.NewRequest("POST", "/api/maintenance/refresh-cache", nil)
		rr := httptest.NewRecorder()
		handler := handleRefreshCache(db, seerrClient, cfg)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["refreshedCount"].(float64) != 0 {
			t.Errorf("expected 0 refreshed count, got %v", resp["refreshedCount"])
		}
	})
}

func TestGetCacheInfoEdgeCases(t *testing.T) {
	// Queries 1 and 2: GORM Model.Count calls — use the query callback
	for failIndex := 1; failIndex <= 2; failIndex++ {
		failIndex := failIndex
		t.Run(fmt.Sprintf("Database Failures query %d", failIndex), func(t *testing.T) {
			db := setupTestDB(t)
			queryCount := 0
			db.Callback().Query().Before("gorm:query").Register("fail_nth_query", func(d *gorm.DB) {
				queryCount++
				if queryCount == failIndex {
					d.AddError(errors.New("mocked error"))
				}
			})

			cfg := &config.Config{DBDriver: "sqlite"}
			req, _ := http.NewRequest("GET", "/api/maintenance/cache", nil)
			rr := httptest.NewRecorder()
			handler := handleGetCacheInfo(db, cfg)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("expected 500, got %d for query %d", rr.Code, failIndex)
			}
		})
	}

	// Queries 3 and 4: Raw().Scan() uses the 'row' callback — inject errors there
	for failIndex := 1; failIndex <= 2; failIndex++ {
		failIndex := failIndex
		t.Run(fmt.Sprintf("Database Failures PRAGMA %d", failIndex), func(t *testing.T) {
			db := setupTestDB(t)
			rowCount := 0
			db.Callback().Row().Before("gorm:row").Register("fail_nth_row", func(d *gorm.DB) {
				rowCount++
				if rowCount == failIndex {
					d.AddError(errors.New("mocked PRAGMA error"))
				}
			})

			cfg := &config.Config{DBDriver: "sqlite"}
			req, _ := http.NewRequest("GET", "/api/maintenance/cache", nil)
			rr := httptest.NewRecorder()
			handler := handleGetCacheInfo(db, cfg)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("expected 500, got %d for PRAGMA %d", rr.Code, failIndex)
			}
		})
	}

	t.Run("Postgres Driver size lookup", func(t *testing.T) {
		// Mock postgres driver to test size lookup logic
		db := setupTestDB(t)
		cfg := &config.Config{DBDriver: "postgres"}
		req, _ := http.NewRequest("GET", "/api/maintenance/cache", nil)
		rr := httptest.NewRecorder()
		handler := handleGetCacheInfo(db, cfg)
		handler.ServeHTTP(rr, req)

		// It will fail because sqlite doesn't support current_database() function
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 on sqlite executing pg queries, got %d", rr.Code)
		}
	})

	t.Run("Unsupported Driver size lookup", func(t *testing.T) {
		db := setupTestDB(t)
		cfg := &config.Config{DBDriver: "unsupported-driver"}
		req, _ := http.NewRequest("GET", "/api/maintenance/cache", nil)
		rr := httptest.NewRecorder()
		handler := handleGetCacheInfo(db, cfg)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}
