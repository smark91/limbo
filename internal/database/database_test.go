package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"limbo/internal/config"
	"gorm.io/gorm/logger"
)

func TestDatabaseInit(t *testing.T) {
	// Test 1: Invalid DB driver returns error
	t.Run("Invalid DB Driver", func(t *testing.T) {
		cfg := &config.Config{
			DBDriver:   "invalid-driver",
			SqlitePath: "test.db",
		}
		_, err := Init(cfg)
		if err == nil {
			t.Errorf("expected error for invalid DB driver, got nil")
		}
	})

	// Test 1b: Connection failure
	t.Run("SQLite Connection Failure", func(t *testing.T) {
		cfg := &config.Config{
			DBDriver:   "sqlite",
			SqlitePath: "/nonexistent/directory/uncreateable.db",
		}
		_, err := Init(cfg)
		if err == nil {
			t.Errorf("expected connection failure error, got nil")
		}
	})

	// Test 2: Valid SQLite in-memory initialization
	t.Run("SQLite In-Memory Init", func(t *testing.T) {
		cfg := &config.Config{
			DBDriver:   "sqlite",
			SqlitePath: ":memory:",
		}
		db, err := Init(cfg)
		if err != nil {
			t.Fatalf("expected no error initializing memory DB, got: %v", err)
		}
		if db == nil {
			t.Fatalf("expected gorm.DB instance, got nil")
		}

		// Verify tables were created
		if !db.Migrator().HasTable(&TriageEntry{}) {
			t.Error("expected database to have triage_entries table")
		}
		if !db.Migrator().HasTable(&SystemMetadata{}) {
			t.Error("expected database to have system_metadata table")
		}
		if !db.Migrator().HasTable(&PushSubscription{}) {
			t.Error("expected database to have push_subscriptions table")
		}
	})

	// Test 3: Status Migration
	t.Run("Status Migration", func(t *testing.T) {
		tempDbFile := "test_migration.db"
		// Clean up any stale test file
		os.Remove(tempDbFile)
		defer os.Remove(tempDbFile)

		cfg := &config.Config{
			DBDriver:   "sqlite",
			SqlitePath: tempDbFile,
		}
		db, err := Init(cfg)
		if err != nil {
			t.Fatalf("failed init: %v", err)
		}
		// Create old "NOT_AVAILABLE" entry
		oldEntry := TriageEntry{
			SeerrRequestID: 55,
			Status:         "NOT_AVAILABLE",
		}
		if err := db.Create(&oldEntry).Error; err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		// Close previous connection to release file lock
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("failed to get sql.DB: %v", err)
		}
		sqlDB.Close()

		// Re-run init to trigger migration logic
		db2, err := Init(cfg)
		if err != nil {
			t.Fatalf("failed re-init: %v", err)
		}
		defer func() {
			sqlDB2, err := db2.DB()
			if err == nil {
				sqlDB2.Close()
			}
		}()

		var checked TriageEntry
		if err := db2.First(&checked, "seerr_request_id = ?", 55).Error; err != nil {
			t.Fatalf("failed query: %v", err)
		}
		if checked.Status != "UNAVAILABLE" {
			t.Errorf("expected status 'UNAVAILABLE', got %s", checked.Status)
		}
	})
}

func TestSlogLogger(t *testing.T) {
	l := &SlogLogger{
		LogLevel: logger.Info,
	}

	ctx := context.Background()

	// Test LogMode
	l2 := l.LogMode(logger.Warn).(*SlogLogger)
	if l2.LogLevel != logger.Warn {
		t.Errorf("expected LogLevel to be Warn, got %v", l2.LogLevel)
	}

	// Call logger methods to ensure they do not panic and achieve coverage
	l.Info(ctx, "Test Info: %s", "hello")
	l.Warn(ctx, "Test Warn: %s", "hello")
	l.Error(ctx, "Test Error: %s", "hello")

	// Call logger methods with lower severity than set (should be no-op)
	silentLogger := &SlogLogger{LogLevel: logger.Silent}
	silentLogger.Info(ctx, "Silent Info")
	silentLogger.Warn(ctx, "Silent Warn")
	silentLogger.Error(ctx, "Silent Error")

	// Test Trace
	l.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM test", 1
	}, nil)

	// Test Trace with error
	l.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM test", 0
	}, errors.New("query error"))

	// Test Trace with slow query (> 200ms)
	l.Trace(ctx, time.Now().Add(-300*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM slow_test", 5
	}, nil)

	// Test Trace under silent level
	silentLogger.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM test", 1
	}, nil)
}
