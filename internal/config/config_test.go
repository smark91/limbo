package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	os.Clearenv()

	cfg := Load()

	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected default DBDriver 'sqlite', got %q", cfg.DBDriver)
	}
	if cfg.DBDSN != "limbo.db" {
		t.Errorf("expected default DBDSN 'limbo.db', got %q", cfg.DBDSN)
	}
	if cfg.SeerrURL != "http://localhost:5055" {
		t.Errorf("expected default SeerrURL 'http://localhost:5055', got %q", cfg.SeerrURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected default LogFormat 'text', got %q", cfg.LogFormat)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	os.Clearenv()
	os.Setenv("DB_DRIVER", "postgres")
	os.Setenv("DB_DSN", "postgres://user:pass@host/db")
	os.Setenv("SEERR_URL", "https://seerr.my-domain.com")
	os.Setenv("SEERR_PUBLIC_URL", "https://seerr-public.my-domain.com")
	os.Setenv("SEERR_API_KEY", "test-key-123")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")

	cfg := Load()

	if cfg.DBDriver != "postgres" {
		t.Errorf("expected DBDriver 'postgres', got %q", cfg.DBDriver)
	}
	if cfg.DBDSN != "postgres://user:pass@host/db" {
		t.Errorf("expected DBDSN, got %q", cfg.DBDSN)
	}
	if cfg.SeerrURL != "https://seerr.my-domain.com" {
		t.Errorf("expected SeerrURL, got %q", cfg.SeerrURL)
	}
	if cfg.SeerrPublicURL != "https://seerr-public.my-domain.com" {
		t.Errorf("expected SeerrPublicURL, got %q", cfg.SeerrPublicURL)
	}
	if cfg.SeerrAPIKey != "test-key-123" {
		t.Errorf("expected SeerrAPIKey, got %q", cfg.SeerrAPIKey)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected LogFormat 'json', got %q", cfg.LogFormat)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Valid configuration",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("LOG_LEVEL", "info")
				os.Setenv("LOG_FORMAT", "text")
			},
			wantErr: false,
		},
		{
			name: "Missing Seerr API Key",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "")
			},
			wantErr: true,
		},
		{
			name: "Invalid Seerr URL",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "invalid-url-format")
			},
			wantErr: true,
		},
		{
			name: "Invalid Seerr Public URL",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "invalid-public-url")
			},
			wantErr: true,
		},
		{
			name: "Invalid DB Driver",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "mysql")
			},
			wantErr: true,
		},
		{
			name: "Invalid Log Level",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("LOG_LEVEL", "trace")
			},
			wantErr: true,
		},
		{
			name: "Invalid Log Format",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("LOG_LEVEL", "info")
				os.Setenv("LOG_FORMAT", "yaml")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			tt.setup()
			cfg := Load()
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
