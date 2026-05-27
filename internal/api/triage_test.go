package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	err = db.AutoMigrate(&database.TriageEntry{}, &database.SystemMetadata{}, &database.PushSubscription{})
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
			"status":         "UNAVAILABLE",
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
		if entry.Status != "UNAVAILABLE" {
			t.Errorf("expected Status 'UNAVAILABLE', got %s", entry.Status)
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

	// 4. GET Triage - Invalid ID format
	t.Run("Get Triage Invalid ID Format", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/triage/abc", nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("seerrRequestId", "abc")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler := handleGetTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// 5. GET Triage - Not Found
	t.Run("Get Triage Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/triage/999", nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("seerrRequestId", "999")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler := handleGetTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	// 6. POST Triage - Invalid JSON
	t.Run("Create Triage Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer([]byte(`{invalid`)))
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// 7. POST Triage - Missing request ID
	t.Run("Create Triage Missing Request ID", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"status": "PENDING",
		})
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// 8. POST Triage - Manual transition to COMPLETED is forbidden
	t.Run("Create Triage Forbidden Completed Status", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"seerrRequestId": 12345,
			"status":         "COMPLETED",
		})
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// 9. POST Triage - Update completed entry forbidden
	t.Run("Update Completed Entry Forbidden", func(t *testing.T) {
		// Seed completed entry
		completedEntry := database.TriageEntry{
			SeerrRequestID: 777,
			Status:         database.StatusCompleted,
		}
		db.Create(&completedEntry)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"seerrRequestId": 777,
			"status":         "PENDING",
		})
		req, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()

		handler := handlePostTriage(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// 10. Database query failures
	t.Run("Database Failures", func(t *testing.T) {
		// Close db connection to trigger failures
		dbClosed := setupTestDB(t)
		sqlDB, _ := dbClosed.DB()
		sqlDB.Close()

		// Test GET failure
		reqGet, _ := http.NewRequest("GET", "/api/triage/123", nil)
		rrGet := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("seerrRequestId", "123")
		reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), chi.RouteCtxKey, rctx))

		handlerGet := handleGetTriage(dbClosed)
		handlerGet.ServeHTTP(rrGet, reqGet)
		if rrGet.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 on db closed for GET, got %d", rrGet.Code)
		}

		// Test POST failure
		reqBody, _ := json.Marshal(map[string]interface{}{
			"seerrRequestId": 123,
			"status":         "PENDING",
		})
		reqPost, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
		rrPost := httptest.NewRecorder()

		handlerPost := handlePostTriage(dbClosed)
		handlerPost.ServeHTTP(rrPost, reqPost)
		if rrPost.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 on db closed for POST, got %d", rrPost.Code)
		}

		// Test POST Create failure
		t.Run("Create Failure", func(t *testing.T) {
			dbErrorCreate := setupTestDB(t)
			dbErrorCreate.Callback().Create().Before("gorm:create").Register("fail_create", func(d *gorm.DB) {
				d.AddError(errors.New("mocked create error"))
			})
			reqBody, _ := json.Marshal(map[string]interface{}{
				"seerrRequestId": 123,
				"status":         "PENDING",
			})
			reqPost, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
			rrPost := httptest.NewRecorder()
			handlerPost := handlePostTriage(dbErrorCreate)
			handlerPost.ServeHTTP(rrPost, reqPost)
			if rrPost.Code != http.StatusInternalServerError {
				t.Errorf("expected status 500 on db create failure, got %d", rrPost.Code)
			}
		})

		// Test POST Update failure
		t.Run("Update Failure", func(t *testing.T) {
			dbErrorUpdate := setupTestDB(t)
			dbErrorUpdate.Create(&database.TriageEntry{SeerrRequestID: 555, Status: database.StatusPending})
			dbErrorUpdate.Callback().Update().Before("gorm:update").Register("fail_update", func(d *gorm.DB) {
				d.AddError(errors.New("mocked update error"))
			})
			reqBody, _ := json.Marshal(map[string]interface{}{
				"seerrRequestId": 555,
				"status":         "WAITING_RELEASE",
			})
			reqPost, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
			rrPost := httptest.NewRecorder()
			handlerPost := handlePostTriage(dbErrorUpdate)
			handlerPost.ServeHTTP(rrPost, reqPost)
			if rrPost.Code != http.StatusInternalServerError {
				t.Errorf("expected status 500 on db update failure, got %d", rrPost.Code)
			}
		})

		// Test POST Reload failure
		t.Run("Reload Failure", func(t *testing.T) {
			dbErrorReload := setupTestDB(t)
			dbErrorReload.Create(&database.TriageEntry{SeerrRequestID: 666, Status: database.StatusPending})
			queryCount := 0
			dbErrorReload.Callback().Query().Before("gorm:query").Register("fail_reload", func(d *gorm.DB) {
				queryCount++
				if queryCount == 2 { // first query is finding the entry, second query is reload
					d.AddError(errors.New("mocked reload error"))
				}
			})
			reqBody, _ := json.Marshal(map[string]interface{}{
				"seerrRequestId": 666,
				"status":         "WAITING_RELEASE",
			})
			reqPost, _ := http.NewRequest("POST", "/api/triage", bytes.NewBuffer(reqBody))
			rrPost := httptest.NewRecorder()
			handlerPost := handlePostTriage(dbErrorReload)
			handlerPost.ServeHTTP(rrPost, reqPost)
			if rrPost.Code != http.StatusInternalServerError {
				t.Errorf("expected status 500 on db reload failure, got %d", rrPost.Code)
			}
		})
	})
}

