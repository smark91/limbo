package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Notifier sends Discord webhook embeds for unfulfilled requests.
type Notifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewNotifier creates a Discord notifier.
func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
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

// NotifyUnfulfilled sends a Discord embed for an unfulfilled request.
func (n *Notifier) NotifyUnfulfilled(title, mediaType, posterURL, serviceURL string, releaseInfo ReleaseInfo, requestedBy string, requestAge time.Duration) error {
	if n.webhookURL == "" {
		slog.Info("No Discord webhook URL configured, skipping notification")
		return nil
	}

	// Color: amber for pending
	color := 0xF59E0B

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

	resp, err := n.httpClient.Post(n.webhookURL, "application/json", bytes.NewReader(body))
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
