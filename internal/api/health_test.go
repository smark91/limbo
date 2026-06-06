package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/seerr"
)

func TestHealthcheckErrors(t *testing.T) {
	t.Run("Seerr API down", func(t *testing.T) {
		db := setupTestDB(t)

		// Mock Seerr status endpoint returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := &config.Config{
			SeerrURL:    server.URL,
			SeerrAPIKey: "test-api-key",
		}
		seerrClient := seerr.NewClient(cfg)

		req, _ := http.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()

		handler := handleHealth(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", rr.Code)
		}

		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["status"] != "error" || resp["seerr"] != "down" || resp["database"] != "up" {
			t.Errorf("incorrect health response: %+v", resp)
		}
	})

	t.Run("Database down", func(t *testing.T) {
		db := setupTestDB(t)

		// Close underlying connection to simulate database down
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("failed to get sql.DB: %v", err)
		}
		sqlDB.Close()

		// Mock Seerr status endpoint returns ok
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		cfg := &config.Config{
			SeerrURL:    server.URL,
			SeerrAPIKey: "test-api-key",
		}
		seerrClient := seerr.NewClient(cfg)

		req, _ := http.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()

		handler := handleHealth(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", rr.Code)
		}

		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["status"] != "error" || resp["seerr"] != "up" || resp["database"] != "down" {
			t.Errorf("incorrect health response: %+v", resp)
		}
	})
	t.Run("Database DB() error", func(t *testing.T) {
		db := setupTestDB(t)
		db.ConnPool = nil // trigger db.DB() error
		if db.Statement != nil {
			db.Statement.ConnPool = nil
		}

		// Mock Seerr status endpoint returns ok
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		cfg := &config.Config{
			SeerrURL:    server.URL,
			SeerrAPIKey: "test-api-key",
		}
		seerrClient := seerr.NewClient(cfg)

		req, _ := http.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()

		handler := handleHealth(db, seerrClient)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", rr.Code)
		}

		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["status"] != "error" || resp["database"] != "down" {
			t.Errorf("incorrect health response: %+v", resp)
		}
	})
}

