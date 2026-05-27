package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"limbo/internal/config"
	"limbo/internal/database"

	"gorm.io/gorm"
)

func TestGetNotificationsConfigHandler(t *testing.T) {
	cfg := &config.Config{
		VapidPublicKey: "test-vapid-public-key",
	}

	req, err := http.NewRequest("GET", "/api/notifications/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := handleGetNotificationsConfig(cfg)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["publicKey"] != "test-vapid-public-key" {
		t.Errorf("expected publicKey 'test-vapid-public-key', got %q", resp["publicKey"])
	}
}

func TestSubscribeHandler(t *testing.T) {
	db := setupTestDB(t)

	// 1. Create a new subscription
	t.Run("Create Subscription", func(t *testing.T) {
		payload := SubscriptionPayload{
			Endpoint: "https://fcm.googleapis.com/fcm/send/test-token-123",
		}
		payload.Keys.P256dh = "test-p256dh"
		payload.Keys.Auth = "test-auth"

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/notifications/subscribe", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler := handleSubscribe(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Verify record exists in DB
		var sub database.PushSubscription
		err := db.Where("endpoint = ?", payload.Endpoint).First(&sub).Error
		if err != nil {
			t.Fatalf("failed to find subscription in database: %v", err)
		}
		if sub.P256dh != "test-p256dh" || sub.Auth != "test-auth" {
			t.Errorf("incorrect subscription keys stored in database")
		}
	})

	// 2. Update existing subscription
	t.Run("Update Subscription", func(t *testing.T) {
		payload := SubscriptionPayload{
			Endpoint: "https://fcm.googleapis.com/fcm/send/test-token-123",
		}
		payload.Keys.P256dh = "updated-p256dh"
		payload.Keys.Auth = "updated-auth"

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/notifications/subscribe", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler := handleSubscribe(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d", rr.Code)
		}

		// Verify record was updated
		var sub database.PushSubscription
		db.Where("endpoint = ?", payload.Endpoint).First(&sub)
		if sub.P256dh != "updated-p256dh" || sub.Auth != "updated-auth" {
			t.Errorf("expected subscription keys to be updated in database")
		}
	})

	// 3. Rejects invalid requests
	t.Run("Reject Invalid Payload", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/notifications/subscribe", bytes.NewBuffer([]byte(`{invalid json`)))
		rr := httptest.NewRecorder()

		handler := handleSubscribe(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("Reject Missing Fields", func(t *testing.T) {
		payload := SubscriptionPayload{
			Endpoint: "https://fcm.googleapis.com/fcm/send/test-token-123",
		}
		// Missing keys

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/notifications/subscribe", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler := handleSubscribe(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("Database error handling", func(t *testing.T) {
		db := setupTestDB(t)
		db.Callback().Query().Before("gorm:query").Register("fail_query", func(d *gorm.DB) {
			d.AddError(errors.New("mocked database error"))
		})

		payload := SubscriptionPayload{
			Endpoint: "https://fcm.googleapis.com/fcm/send/test-token-123",
		}
		payload.Keys.P256dh = "test-p256dh"
		payload.Keys.Auth = "test-auth"

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/notifications/subscribe", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler := handleSubscribe(db)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 Internal Server Error, got %d", rr.Code)
		}
	})
}
