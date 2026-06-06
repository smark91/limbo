package scanner

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{30 * time.Minute, "30 minutes"},
		{10 * time.Hour, "10 hours"},
		{24 * time.Hour, "1 day"},
		{48 * time.Hour, "2 days"},
		{100 * time.Hour, "4 days"},
	}

	for _, tc := range tests {
		got := formatDuration(tc.d)
		if got != tc.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", tc.d, got, tc.expected)
		}
	}
}

func TestNotifier(t *testing.T) {
	// Generate valid VAPID keys
	vapidPriv, vapidPub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("failed to generate VAPID keys: %v", err)
	}

	// Generate valid elliptic curve key pair for subscription p256dh
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate EC key: %v", err)
	}
	pubKeyBytes := priv.PublicKey().Bytes()
	subP256dh := base64.StdEncoding.EncodeToString(pubKeyBytes)

	// Generate random 16-byte auth secret for subscription auth
	authBytes := make([]byte, 16)
	if _, err := rand.Read(authBytes); err != nil {
		t.Fatalf("failed to read random bytes: %v", err)
	}
	subAuth := base64.StdEncoding.EncodeToString(authBytes)

	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&database.PushSubscription{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// 1. Mock Discord Server
	var receivedDiscordPayload map[string]interface{}
	discordCalls := 0
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discordCalls++
		json.NewDecoder(r.Body).Decode(&receivedDiscordPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer discordServer.Close()

	// 2. Mock Web Push service endpoint
	pushCalls := 0
	lastPushStatus := http.StatusOK
	pushServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pushCalls++
		w.WriteHeader(lastPushStatus)
	}))
	defer pushServer.Close()

	// Setup Config
	cfg := &config.Config{
		DiscordWebhookURL: discordServer.URL,
		SeerrPublicURL:    "http://seerr-public",
		VapidPublicKey:    vapidPub,
		VapidPrivateKey:   vapidPriv,
		VapidSubject:      "mailto:admin@test.com",
	}

	notifier := NewNotifier(cfg, db)

	t.Run("IsDiscordConfigured", func(t *testing.T) {
		if !notifier.IsDiscordConfigured() {
			t.Errorf("expected IsDiscordConfigured to be true")
		}

		emptyNotifier := NewNotifier(&config.Config{}, db)
		if emptyNotifier.IsDiscordConfigured() {
			t.Errorf("expected IsDiscordConfigured to be false for empty URL")
		}
	})

	t.Run("IsVAPIDConfigured", func(t *testing.T) {
		if !notifier.IsVAPIDConfigured() {
			t.Errorf("expected IsVAPIDConfigured to be true")
		}

		emptyNotifier := NewNotifier(&config.Config{}, db)
		if emptyNotifier.IsVAPIDConfigured() {
			t.Errorf("expected IsVAPIDConfigured to be false for empty keys")
		}
	})

	t.Run("NotifyUnfulfilled - Success Discord and WebPush", func(t *testing.T) {
		// Clean up subscriptions
		db.Exec("DELETE FROM push_subscriptions")

		// Create a subscription pointing to our mock push server
		sub := database.PushSubscription{
			Endpoint: pushServer.URL,
			P256dh:   subP256dh,
			Auth:     subAuth,
		}
		if err := db.Create(&sub).Error; err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		discordCalls = 0
		pushCalls = 0
		lastPushStatus = http.StatusOK
		receivedDiscordPayload = nil

		now := time.Now()
		releaseInfo := ReleaseInfo{
			Date:   &now,
			Source: "Digital",
		}

		err := notifier.NotifyUnfulfilled("Interstellar", "movie", "http://poster/path.jpg", "http://radarr/1", releaseInfo, "RequesterUser", 2*time.Hour)
		if err != nil {
			t.Fatalf("NotifyUnfulfilled failed: %v", err)
		}

		if discordCalls != 1 {
			t.Errorf("expected 1 discord call, got %d", discordCalls)
		}
		if pushCalls != 1 {
			t.Errorf("expected 1 push call, got %d", pushCalls)
		}

		// Verify Discord payload
		embeds := receivedDiscordPayload["embeds"].([]interface{})
		embed := embeds[0].(map[string]interface{})
		if embed["title"] != "🎬 Interstellar" {
			t.Errorf("unexpected title: %v", embed["title"])
		}

		// Check if subscription was preserved (since status was 200)
		var count int64
		db.Model(&database.PushSubscription{}).Count(&count)
		if count != 1 {
			t.Errorf("expected subscription to be preserved, got count=%d", count)
		}
	})

	t.Run("NotifyUnfulfilled - WebPush Gone/NotFound Cleans DB", func(t *testing.T) {
		// Set mock push server to return 410 Gone
		lastPushStatus = http.StatusGone
		pushCalls = 0

		// Re-create subscription
		db.Exec("DELETE FROM push_subscriptions")
		sub := database.PushSubscription{
			Endpoint: pushServer.URL,
			P256dh:   subP256dh,
			Auth:     subAuth,
		}
		if err := db.Create(&sub).Error; err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		now := time.Now()
		releaseInfo := ReleaseInfo{
			Date:   &now,
			Source: "Digital",
		}

		err := notifier.NotifyUnfulfilled("Interstellar", "movie", "", "", releaseInfo, "RequesterUser", 2*time.Hour)
		if err != nil {
			t.Fatalf("NotifyUnfulfilled failed: %v", err)
		}

		if pushCalls != 1 {
			t.Errorf("expected 1 push call, got %d", pushCalls)
		}

		// Check if subscription was deleted from DB
		var count int64
		db.Model(&database.PushSubscription{}).Count(&count)
		if count != 0 {
			t.Errorf("expected subscription to be deleted from DB, got count=%d", count)
		}
	})

	t.Run("NotifyUnfulfilled - Discord Failure handles gracefully", func(t *testing.T) {
		discordServerErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer discordServerErr.Close()

		badCfg := &config.Config{
			DiscordWebhookURL: discordServerErr.URL,
		}
		badNotifier := NewNotifier(badCfg, db)

		err := badNotifier.NotifyUnfulfilled("Title", "tv", "", "", ReleaseInfo{}, "User", 10*time.Minute)
		if err == nil {
			t.Errorf("expected error from bad Discord webhook, got nil")
		}
		if !strings.Contains(err.Error(), "discord webhook returned 400") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
