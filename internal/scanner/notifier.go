package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/database"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"
)

// Notifier handles both Discord webhook alerts and Web Push notifications.
type Notifier struct {
	cfg        *config.Config
	db         *gorm.DB
	httpClient *http.Client
}

// NewNotifier creates a Notifier.
func NewNotifier(cfg *config.Config, db *gorm.DB) *Notifier {
	return &Notifier{
		cfg:        cfg,
		db:         db,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// DiscordEmbed represents a Discord webhook embed.
type DiscordEmbed struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Color       int             `json:"color"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Fields      []EmbedField    `json:"fields"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
}

type EmbedThumbnail struct {
	URL string `json:"url"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}

type webhookPayload struct {
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

// NotifyUnfulfilled dispatches the alert to both Discord and all active PWA Web Push subscribers.
func (n *Notifier) NotifyUnfulfilled(title, mediaType, posterURL, serviceURL string, releaseInfo ReleaseInfo, requestedBy string, requestAge time.Duration) error {
	var discordErr, pushErr error

	// 1. Send Discord alert if configured
	if n.cfg.DiscordWebhookURL != "" {
		discordErr = n.sendDiscord(title, mediaType, posterURL, serviceURL, releaseInfo, requestedBy, requestAge)
	} else {
		slog.Debug("Discord Webhook URL not configured, skipping Discord alert")
	}

	// 2. Send Web Push notifications to registered clients
	pushErr = n.sendWebPush(title, mediaType, posterURL, serviceURL)

	// Return error if either notification dispatch failed
	if discordErr != nil {
		return discordErr
	}
	return pushErr
}

func (n *Notifier) sendDiscord(title, mediaType, posterURL, serviceURL string, releaseInfo ReleaseInfo, requestedBy string, requestAge time.Duration) error {
	color := 0xF59E0B // Amber for pending
	typeEmoji := "🎬"
	if mediaType == "tv" {
		typeEmoji = "📺"
	}

	releaseStr := "Unknown"
	if releaseInfo.Date != nil {
		releaseStr = fmt.Sprintf("%s (%s)", releaseInfo.Date.Format("02 Jan 2006"), releaseInfo.Source)
	}

	embed := DiscordEmbed{
		Title:       fmt.Sprintf("%s %s", typeEmoji, title),
		Description: fmt.Sprintf("This request has been approved for **%s** but hasn't been fulfilled yet.", formatDuration(requestAge)),
		Color:       color,
		Fields: []EmbedField{
			{Name: "Type", Value: mediaType, Inline: true},
			{Name: "Release", Value: releaseStr, Inline: true},
			{Name: "Requested By", Value: requestedBy, Inline: true},
		},
		Footer:    &EmbedFooter{Text: "Limbo • Unfulfilled Request Monitor"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if posterURL != "" {
		embed.Thumbnail = &EmbedThumbnail{URL: posterURL}
	}

	if serviceURL != "" {
		serviceName := "Radarr"
		if mediaType == "tv" {
			serviceName = "Sonarr"
		}
		embed.Fields = append(embed.Fields, EmbedField{
			Name:  serviceName,
			Value: fmt.Sprintf("[Open in %s](%s)", serviceName, serviceURL),
		})
	}

	payload := webhookPayload{
		Username: "Limbo",
		Embeds:   []DiscordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling webhook payload: %w", err)
	}

	resp, err := n.httpClient.Post(n.cfg.DiscordWebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned %d", resp.StatusCode)
	}

	slog.Info("Sent Discord alert for request", slog.String("title", title))
	return nil
}

func (n *Notifier) sendWebPush(title, mediaType, posterURL, serviceURL string) error {
	if n.cfg.VapidPublicKey == "" || n.cfg.VapidPrivateKey == "" {
		slog.Debug("VAPID keys not configured, skipping Web Push")
		return nil
	}

	var subs []database.PushSubscription
	if err := n.db.Find(&subs).Error; err != nil {
		return fmt.Errorf("failed to fetch push subscriptions: %w", err)
	}

	if len(subs) == 0 {
		slog.Debug("No PWA push subscriptions registered, skipping Web Push")
		return nil
	}

	destURL := n.cfg.SeerrPublicURL
	if serviceURL != "" {
		destURL = serviceURL
	}

	displayType := "Movie"
	if mediaType == "tv" {
		displayType = "TV"
	}
	titleStr := fmt.Sprintf("%s (%s)", title, displayType)

	payload, err := json.Marshal(map[string]string{
		"title":    titleStr,
		"body":     "This request didn't get fulfilled.",
		"url":      destURL,
		"seerrUrl": n.cfg.SeerrPublicURL,
		"image":    posterURL,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}

	sentCount := 0
	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, s, &webpush.Options{
			Subscriber:      n.cfg.VapidSubject,
			VAPIDPublicKey:  n.cfg.VapidPublicKey,
			VAPIDPrivateKey: n.cfg.VapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			slog.Error("Error sending web push notification", slog.String("endpoint", sub.Endpoint), slog.Any("error", err))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			slog.Info("Removing invalid/expired push subscription", slog.String("endpoint", sub.Endpoint))
			n.db.Delete(&sub)
		} else {
			sentCount++
		}
	}

	if sentCount > 0 {
		slog.Info("Dispatched Web Push alerts", slog.Int("count", sentCount), slog.String("title", title))
	}
	return nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 1 {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if hours < 24 {
		return fmt.Sprintf("%d hours", hours)
	}
	days := hours / 24
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// IsDiscordConfigured returns true if a Discord Webhook URL is set.
func (n *Notifier) IsDiscordConfigured() bool {
	return n.cfg.DiscordWebhookURL != ""
}

// IsVAPIDConfigured returns true if VAPID keys are configured.
func (n *Notifier) IsVAPIDConfigured() bool {
	return n.cfg.VapidPublicKey != "" && n.cfg.VapidPrivateKey != ""
}
