package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/seerr"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&database.TriageEntry{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestHealthcheck(t *testing.T) {
	db := setupTestDB(t)

	// Start mock Seerr status endpoint server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-api-key",
	}
	seerrClient := seerr.NewClient(cfg)

	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := handleHealth(db, seerrClient)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
	}

	var gotMap map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &gotMap); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if gotMap["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", gotMap["status"])
	}
	if gotMap["database"] != "up" {
		t.Errorf("expected database 'up', got %q", gotMap["database"])
	}
	if gotMap["seerr"] != "up" {
		t.Errorf("expected seerr 'up', got %q", gotMap["seerr"])
	}
}

func TestTriageEndpoints(t *testing.T) {
	db := setupTestDB(t)

	// 1. Create a triage entry via POST /api/triage
	t.Run("Create Triage Entry", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"seerrRequestId": 123,
			"status":         "NOT_AVAILABLE",
			"notes":          "Will check later",
		})
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var entry database.TriageEntry
		json.Unmarshal(rr.Body.Bytes(), &entry)
		if entry.SeerrRequestID != 123 {
			t.Errorf("expected SeerrRequestID 123, got %d", entry.SeerrRequestID)
		}
		if entry.Status != "NOT_AVAILABLE" {
			t.Errorf("expected Status 'NOT_AVAILABLE', got %s", entry.Status)
		}
		if entry.Notes == nil || *entry.Notes != "Will check later" {
			t.Errorf("expected Notes 'Will check later', got %v", entry.Notes)
		}
	})

	// 2. Fetch the created entry via GET /api/triage/{seerrRequestId}
	t.Run("Get Triage Entry", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/triage/123", nil)
		rr := httptest.NewRecorder()

		// Set up chi path parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("seerrRequestId", "123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler := handleGetTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var entry database.TriageEntry
		json.Unmarshal(rr.Body.Bytes(), &entry)
		if entry.SeerrRequestID != 123 {
			t.Errorf("expected SeerrRequestID 123, got %d", entry.SeerrRequestID)
		}
	})

	// 3. Test Invalid Status validation
	t.Run("Create Triage Entry with Invalid Status", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"seerrRequestId": 456,
			"status":         "INVALID_STATUS_NAME",
		})
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}
