package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration parsed from environment variables.
type Config struct {
	// Database
	DBDriver string // "postgres" or "sqlite"
	DBDSN    string

	// Seerr
	SeerrURL       string
	SeerrPublicURL string
	SeerrAPIKey    string

	// Discord
	DiscordWebhookURL string

	// App
	ReleaseCountry      string
	ScanInterval        time.Duration
	AlertThreshold      time.Duration
	AlertWindow         time.Duration
	Port                string

	// Logging
	LogLevel  string // "debug", "info", "warn", "error"
	LogFormat string // "text", "json"
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		// Database
		DBDriver: envOrDefault("DB_DRIVER", "sqlite"),
		DBDSN:    envOrDefault("DB_DSN", "limbo.db"),

		// Seerr
		SeerrURL:       envOrDefault("SEERR_URL", "http://localhost:5055"),
		SeerrPublicURL: envOrDefault("SEERR_PUBLIC_URL", "http://localhost:5055"),
		SeerrAPIKey:    os.Getenv("SEERR_API_KEY"),

		// Discord
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),

		// App
		ReleaseCountry: envOrDefault("RELEASE_COUNTRY", "US"),
		ScanInterval:   envOrDuration("SCAN_INTERVAL_MINUTES", 10),
		AlertThreshold: envOrDuration("ALERT_THRESHOLD_MINUTES", 10),
		AlertWindow:    envOrDuration("ALERT_WINDOW_MINUTES", 10),
		Port:           envOrDefault("LIMBO_PORT", "3000"),

		// Logging
		LogLevel:  envOrDefault("LOG_LEVEL", "info"),
		LogFormat: envOrDefault("LOG_FORMAT", "text"),
	}
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

	driver := strings.ToLower(c.DBDriver)
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("unsupported DB_DRIVER %q: must be 'sqlite' or 'postgres'", c.DBDriver)
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
