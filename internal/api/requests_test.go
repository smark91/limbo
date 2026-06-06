package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"
)

func TestRequestsHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database
	now := time.Now()
	entries := []database.TriageEntry{
		{
			SeerrRequestID: 1,
			Title:          "Movie A",
			MediaType:      "movie",
			Status:         database.StatusPending,
			PosterPath:     "/posterA.jpg",
			ServiceURL:     "http://radarr/1",
			SeerrCreatedAt: now.Add(-2 * time.Hour),
		},
		{
			SeerrRequestID: 2,
			Title:          "TV B",
			MediaType:      "tv",
			Status:         database.StatusUnavailable,
			PosterPath:     "/posterB.jpg",
			SeerrCreatedAt: now.Add(-1 * time.Hour),
		},
		{
			SeerrRequestID: 3,
			Title:          "Movie C",
			MediaType:      "movie",
			Status:         database.StatusCompleted,
			SeerrCreatedAt: now.Add(-3 * time.Hour),
		},
	}
	for _, entry := range entries {
		db.Create(&entry)
	}

	cfg := &config.Config{
		SeerrPublicURL: "http://seerr-public",
	}

	t.Run("Get all requests default sort", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp) != 3 {
			t.Fatalf("expected 3 items, got %d", len(resp))
		}

		// Default sort is seerr_created_at DESC: TV B (1h ago), Movie A (2h ago), Movie C (3h ago)
		if resp[0].Title != "TV B" || resp[1].Title != "Movie A" || resp[2].Title != "Movie C" {
			t.Errorf("incorrect default sorting: %+v", resp)
		}

		// Verify enrichment
		if resp[1].PosterURL != "https://image.tmdb.org/t/p/w300/posterA.jpg" {
			t.Errorf("incorrect posterUrl: %q", resp[1].PosterURL)
		}
		if resp[1].SeerrURL != "http://seerr-public/movie/0" {
			t.Errorf("incorrect seerrUrl: %q", resp[1].SeerrURL)
		}
		if resp[1].ServiceURL != "http://radarr/1" {
			t.Errorf("incorrect serviceUrl: %q", resp[1].ServiceURL)
		}
	})

	t.Run("Filter by status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?status=pending", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp))
		}
		if resp[0].Title != "Movie A" {
			t.Errorf("expected Movie A, got %s", resp[0].Title)
		}
	})

	t.Run("Filter by status all", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?status=all", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp) != 3 {
			t.Fatalf("expected 3 items, got %d", len(resp))
		}
	})

	t.Run("Filter by type", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?type=tv", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp))
		}
		if resp[0].Title != "TV B" {
			t.Errorf("expected TV B, got %s", resp[0].Title)
		}
	})

	t.Run("Search by title", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?search=movie", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp) != 2 {
			t.Fatalf("expected 2 items, got %d", len(resp))
		}
	})

	t.Run("Sort by title", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?sort=title", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp[0].Title != "Movie A" || resp[1].Title != "Movie C" || resp[2].Title != "TV B" {
			t.Errorf("sorting by title failed: %+v", resp)
		}
	})

	t.Run("Sort by release", func(t *testing.T) {
		// Set release dates
		db.Model(&database.TriageEntry{}).Where("seerr_request_id = ?", 1).Update("release_date", now.Add(24*time.Hour))
		db.Model(&database.TriageEntry{}).Where("seerr_request_id = ?", 2).Update("release_date", now.Add(-24*time.Hour))

		req, _ := http.NewRequest("GET", "/api/requests?sort=release", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// TV B (released in past/earliest), Movie A (released in future), Movie C (nil release date)
		if resp[0].Title != "TV B" || resp[1].Title != "Movie A" || resp[2].Title != "Movie C" {
			t.Errorf("sorting by release failed: %+v", resp)
		}
	})

	t.Run("Sort by release desc", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?sort=release_desc", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Movie A (released in future), TV B (released in past), Movie C (nil release date)
		if resp[0].Title != "Movie A" || resp[1].Title != "TV B" || resp[2].Title != "Movie C" {
			t.Errorf("sorting by release desc failed: %+v", resp)
		}
	})

	t.Run("Sort by oldest", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/requests?sort=oldest", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		var resp []enrichedRequest
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// oldest: Movie C (3h ago), Movie A (2h ago), TV B (1h ago)
		if resp[0].Title != "Movie C" || resp[1].Title != "Movie A" || resp[2].Title != "TV B" {
			t.Errorf("sorting by oldest failed: %+v", resp)
		}
	})

	t.Run("Database error handling", func(t *testing.T) {
		// Delete table to cause query error
		db.Migrator().DropTable(&database.TriageEntry{})

		req, _ := http.NewRequest("GET", "/api/requests", nil)
		rr := httptest.NewRecorder()

		handler := handleRequests(cfg, db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 status on db query failure, got %d", rr.Code)
		}
	})
}
