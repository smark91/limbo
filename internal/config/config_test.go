package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected default DBDriver 'sqlite', got %q", cfg.DBDriver)
	}
	if cfg.SqlitePath != "/data/limbo.db" {
		t.Errorf("expected default SqlitePath '/data/limbo.db', got %q", cfg.SqlitePath)
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
	os.Setenv("POSTGRES_URL", "postgres://user:pass@host/db")
	os.Setenv("SEERR_URL", "https://seerr.my-domain.com")
	os.Setenv("SEERR_PUBLIC_URL", "https://seerr-public.my-domain.com")
	os.Setenv("SEERR_API_KEY", "test-key-123")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.DBDriver != "postgres" {
		t.Errorf("expected DBDriver 'postgres', got %q", cfg.DBDriver)
	}
	if cfg.PostgresURL != "postgres://user:pass@host/db" {
		t.Errorf("expected PostgresURL, got %q", cfg.PostgresURL)
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
			name: "Valid sqlite configuration",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("SQLITE_PATH", "/data/limbo.db")
				os.Setenv("LOG_LEVEL", "info")
				os.Setenv("LOG_FORMAT", "text")
			},
			wantErr: false,
		},
		{
			name: "Valid postgres configuration",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "postgres")
				os.Setenv("POSTGRES_URL", "postgres://localhost:5432/db")
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
			name: "Missing POSTGRES_URL when driver is postgres",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "postgres")
				os.Setenv("POSTGRES_URL", "")
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
		{
			name: "Invalid Discord Webhook URL",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("DISCORD_WEBHOOK_URL", "not-a-valid-url")
			},
			wantErr: true,
		},
		{
			name: "Invalid Discord Webhook Host",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("DISCORD_WEBHOOK_URL", "https://google.com/api/webhooks/123/abc")
			},
			wantErr: true,
		},
		{
			name: "Invalid Discord Webhook Path",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/invalid/path/123/abc")
			},
			wantErr: true,
		},
		{
			name: "Bypass Discord Webhook Host Check",
			setup: func() {
				os.Setenv("SEERR_API_KEY", "some-key")
				os.Setenv("SEERR_URL", "http://localhost:5055")
				os.Setenv("SEERR_PUBLIC_URL", "http://localhost:5055")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("DISCORD_WEBHOOK_URL", "https://123discord.com/api/webhooks/123/abc")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			tt.setup()
			cfg, err := Load()
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("Load() unexpected error: %v", err)
			}
			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSecretExclusiveAndFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "limbo-secrets-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretFile := filepath.Join(tmpDir, "my_secret")
	secretValue := "secret-token-123"
	if err := os.WriteFile(secretFile, []byte(secretValue+"\n"), 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	// 1. Test loading via direct env
	os.Clearenv()
	os.Setenv("SEERR_API_KEY", "direct-val")
	val, err := loadSecret("SEERR_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "direct-val" {
		t.Errorf("expected direct-val, got %q", val)
	}

	// 2. Test loading via _FILE env
	os.Clearenv()
	os.Setenv("SEERR_API_KEY_FILE", secretFile)
	val, err = loadSecret("SEERR_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != secretValue {
		t.Errorf("expected %q, got %q", secretValue, val)
	}

	// 3. Test exclusive validation (both set)
	os.Clearenv()
	os.Setenv("SEERR_API_KEY", "direct-val")
	os.Setenv("SEERR_API_KEY_FILE", secretFile)
	_, err = loadSecret("SEERR_API_KEY")
	if err == nil {
		t.Error("expected error when both SEERR_API_KEY and SEERR_API_KEY_FILE are set, got nil")
	}

	// 4. Test missing file error when explicitly set
	os.Clearenv()
	os.Setenv("SEERR_API_KEY_FILE", "/nonexistent/secret/path")
	_, err = loadSecret("SEERR_API_KEY")
	if err == nil {
		t.Error("expected error when SEERR_API_KEY_FILE points to a nonexistent file, got nil")
	}
}
