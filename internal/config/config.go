package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration parsed from environment variables.
type Config struct {
	// Database
	DBDriver    string // "postgres" or "sqlite"
	PostgresURL string
	SqlitePath  string

	// Seerr
	SeerrURL       string
	SeerrPublicURL string
	SeerrAPIKey    string

	// Discord
	DiscordWebhookURL string

	// App
	ReleaseCountry      string
	ScanInterval        time.Duration
	AlertDelay          time.Duration
	AlertMaxAge         time.Duration
	Port                string

	// Logging
	LogLevel  string // "debug", "info", "warn", "error"
	LogFormat string // "text", "json"

	// VAPID keys for Web Push Notifications
	VapidPublicKey  string
	VapidPrivateKey string
	VapidSubject    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	postgresURL, err := loadSecret("POSTGRES_URL")
	if err != nil {
		return nil, err
	}

	seerrAPIKey, err := loadSecret("SEERR_API_KEY")
	if err != nil {
		return nil, err
	}

	discordWebhookURL, err := loadSecret("DISCORD_WEBHOOK_URL")
	if err != nil {
		return nil, err
	}

	return &Config{
		// Database
		DBDriver:    envOrDefault("DB_DRIVER", "sqlite"),
		PostgresURL: postgresURL,
		SqlitePath:  envOrDefault("SQLITE_PATH", "/data/limbo.db"),

		// Seerr
		SeerrURL:       envOrDefault("SEERR_URL", "http://localhost:5055"),
		SeerrPublicURL: envOrDefault("SEERR_PUBLIC_URL", "http://localhost:5055"),
		SeerrAPIKey:    seerrAPIKey,

		// Discord
		DiscordWebhookURL: discordWebhookURL,

		// App
		ReleaseCountry: envOrDefault("RELEASE_COUNTRY", "US"),
		ScanInterval:   envOrDuration("SCAN_INTERVAL_MINUTES", 10),
		AlertDelay:     envOrDuration("ALERT_DELAY_MINUTES", 10),
		AlertMaxAge:    envOrDuration("ALERT_MAX_AGE_MINUTES", 1440),
		Port:           envOrDefault("LIMBO_PORT", "3000"),

		// Logging
		LogLevel:  envOrDefault("LOG_LEVEL", "info"),
		LogFormat: envOrDefault("LOG_FORMAT", "text"),

		// VAPID
		VapidPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VapidPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VapidSubject:    envOrDefault("VAPID_SUBJECT", "mailto:admin@limbo.local"),
	}, nil
}

// Validate checks the configuration for required settings and correct formats.
func (c *Config) Validate() error {
	if c.SeerrAPIKey == "" {
		return errors.New("SEERR_API_KEY is required but not set")
	}

	if _, err := url.ParseRequestURI(c.SeerrURL); err != nil {
		return fmt.Errorf("invalid SEERR_URL: %w", err)
	}

	if _, err := url.ParseRequestURI(c.SeerrPublicURL); err != nil {
		return fmt.Errorf("invalid SEERR_PUBLIC_URL: %w", err)
	}

	if c.DiscordWebhookURL != "" {
		parsed, err := url.ParseRequestURI(c.DiscordWebhookURL)
		if err != nil {
			return fmt.Errorf("invalid DISCORD_WEBHOOK_URL: %w", err)
		}
		host := strings.ToLower(parsed.Host)
		if host != "discord.com" && host != "discordapp.com" &&
			!strings.HasSuffix(host, ".discord.com") && !strings.HasSuffix(host, ".discordapp.com") {
			return fmt.Errorf("invalid DISCORD_WEBHOOK_URL: host must be discord.com, discordapp.com, or a subdomain of them")
		}
		if !strings.HasPrefix(parsed.Path, "/api/webhooks/") {
			return fmt.Errorf("invalid DISCORD_WEBHOOK_URL: path must start with /api/webhooks/")
		}
	}

	driver := strings.ToLower(c.DBDriver)
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("unsupported DB_DRIVER %q: must be 'sqlite' or 'postgres'", c.DBDriver)
	}

	if driver == "postgres" && c.PostgresURL == "" {
		return errors.New("POSTGRES_URL is required when DB_DRIVER is 'postgres'")
	}

	level := strings.ToLower(c.LogLevel)
	if level != "debug" && level != "info" && level != "warn" && level != "error" {
		return fmt.Errorf("invalid LOG_LEVEL %q: must be one of 'debug', 'info', 'warn', 'error'", c.LogLevel)
	}

	format := strings.ToLower(c.LogFormat)
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid LOG_FORMAT %q: must be 'text' or 'json'", c.LogFormat)
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDuration(key string, defaultMinutes int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if mins, err := strconv.Atoi(v); err == nil {
			return time.Duration(mins) * time.Minute
		}
	}
	return time.Duration(defaultMinutes) * time.Minute
}

func loadSecret(key string) (string, error) {
	val := os.Getenv(key)
	fileVar := key + "_FILE"
	filePath := os.Getenv(fileVar)

	// Rule 1: both cannot be set
	if val != "" && filePath != "" {
		return "", fmt.Errorf("both %s and %s environment variables are set; only one is allowed", key, fileVar)
	}

	if val != "" {
		return val, nil
	}

	isDefault := false
	if filePath == "" {
		filePath = "/run/secrets/" + strings.ToLower(key)
		isDefault = true
	}

	filePath = filepath.Clean(filePath)
	// #nosec G304
	content, err := os.ReadFile(filePath)
	if err != nil {
		if isDefault && os.IsNotExist(err) {
			// It is okay if the default secrets file does not exist
			return "", nil
		}
		return "", fmt.Errorf("failed to read secret file from %s: %w", filePath, err)
	}

	return strings.TrimSpace(string(content)), nil
}
