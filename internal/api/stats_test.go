package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/scanner"
	"limbo/internal/seerr"

	"gorm.io/gorm"
)

func TestStatsHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed database with multiple statuses
	db.Create(&database.TriageEntry{SeerrRequestID: 1, Status: database.StatusPending})
	db.Create(&database.TriageEntry{SeerrRequestID: 2, Status: database.StatusPending})
	db.Create(&database.TriageEntry{SeerrRequestID: 3, Status: database.StatusUnavailable})
	db.Create(&database.TriageEntry{SeerrRequestID: 4, Status: database.StatusWaitingRelease})
	db.Create(&database.TriageEntry{SeerrRequestID: 5, Status: database.StatusCompleted})

	cfg := &config.Config{}
	sc := scanner.New(cfg, db, seerr.NewClient(cfg))

	t.Run("Get Stats Successfully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats", nil)
		rr := httptest.NewRecorder()

		handler := handleStats(db, sc)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp statsResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp.Pending != 2 {
			t.Errorf("expected 2 pending, got %d", resp.Pending)
		}
		if resp.Unavailable != 1 {
			t.Errorf("expected 1 unavailable, got %d", resp.Unavailable)
		}
		if resp.WaitingRelease != 1 {
			t.Errorf("expected 1 waiting release, got %d", resp.WaitingRelease)
		}
		if resp.Completed != 1 {
			t.Errorf("expected 1 completed, got %d", resp.Completed)
		}
		if resp.Total != 5 {
			t.Errorf("expected 5 total, got %d", resp.Total)
		}
		if resp.Version != "dev" {
			t.Errorf("expected version dev, got %s", resp.Version)
		}
	})

	for failIndex := 1; failIndex <= 5; failIndex++ {
		t.Run(fmt.Sprintf("Database error on query %d", failIndex), func(t *testing.T) {
			db := setupTestDB(t)
			db.Create(&database.TriageEntry{SeerrRequestID: 1, Status: database.StatusPending})

			queryCount := 0
			db.Callback().Query().Before("gorm:query").Register("fail_nth_query", func(d *gorm.DB) {
				queryCount++
				if queryCount == failIndex {
					d.AddError(errors.New("mocked error"))
				}
			})

			req, _ := http.NewRequest("GET", "/api/stats", nil)
			rr := httptest.NewRecorder()

			handler := handleStats(db, sc)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("expected 500 status on database failure at query %d, got %d", failIndex, rr.Code)
			}
		})
	}
}
