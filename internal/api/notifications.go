package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"

	"gorm.io/gorm"
)

// SubscriptionPayload matches the standard browser PushSubscription JSON structure.
type SubscriptionPayload struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func handleGetNotificationsConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := map[string]string{
			"publicKey": cfg.VapidPublicKey,
		}
		writeJSON(w, r, http.StatusOK, res)
	}
}

func handleSubscribe(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload SubscriptionPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if payload.Endpoint == "" || payload.Keys.P256dh == "" || payload.Keys.Auth == "" {
			http.Error(w, "Missing subscription fields", http.StatusBadRequest)
			return
		}

		sub := database.PushSubscription{
			Endpoint:  payload.Endpoint,
			P256dh:    payload.Keys.P256dh,
			Auth:      payload.Keys.Auth,
			CreatedAt: time.Now(),
		}

		// Save or update subscription
		err := db.Transaction(func(tx *gorm.DB) error {
			var existing database.PushSubscription
			err := tx.Where("endpoint = ?", sub.Endpoint).First(&existing).Error
			if err == nil {
				// Update keys if endpoint exists
				existing.P256dh = sub.P256dh
				existing.Auth = sub.Auth
				return tx.Save(&existing).Error
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Insert new
				return tx.Create(&sub).Error
			}
			return err
		})

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, r, http.StatusCreated, map[string]string{"status": "subscribed"})
	}
}
