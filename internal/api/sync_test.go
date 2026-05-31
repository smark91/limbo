package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"limbo/internal/config"
	"limbo/internal/scanner"
	"limbo/internal/seerr"
)

func TestSyncHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)

		// Mock Seerr API Server so scanner sync behaves
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			res := map[string]interface{}{
				"pageInfo": map[string]interface{}{"pages": 1, "page": 1, "results": 0},
				"results":  []interface{}{},
			}
			json.NewEncoder(w).Encode(res)
		}))
		defer server.Close()

		cfg := &config.Config{
			SeerrURL:    server.URL,
			SeerrAPIKey: "test-key",
		}
		sc := scanner.New(cfg, db, seerr.NewClient(cfg))

		req, _ := http.NewRequest("POST", "/api/sync", nil)
		rr := httptest.NewRecorder()

		handler := handleSync(sc)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["success"] != true {
			t.Errorf("expected success=true, got %v", resp["success"])
		}
		if resp["message"] != "Sync triggered" {
			t.Errorf("expected message 'Sync triggered', got %v", resp["message"])
		}
	})

	t.Run("failure", func(t *testing.T) {
		db := setupTestDB(t)

		// Point Seerr to invalid URL to force connection error
		cfg := &config.Config{
			SeerrURL:    "http://localhost:59999",
			SeerrAPIKey: "test-key",
		}
		sc := scanner.New(cfg, db, seerr.NewClient(cfg))

		req, _ := http.NewRequest("POST", "/api/sync", nil)
		rr := httptest.NewRecorder()

		handler := handleSync(sc)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502, got %d", rr.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["success"] != false {
			t.Errorf("expected success=false, got %v", resp["success"])
		}
		if resp["error"] == nil || resp["error"] == "" {
			t.Errorf("expected error message, got empty/nil")
		}
	})
}
