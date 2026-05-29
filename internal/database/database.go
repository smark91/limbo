package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"limbo/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init opens a database connection based on the config driver and runs auto-migration.
func Init(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresURL)
	case "sqlite":
		dialector = sqlite.Open(cfg.SqlitePath)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s (use 'postgres' or 'sqlite')", cfg.DBDriver)
	}

	gormLogger := &SlogLogger{
		LogLevel: logger.Warn, // Only log warnings/errors/slow-queries by default
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database to tune pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if cfg.DBDriver == "sqlite" {
		// SQLite WAL mode & timeout pragmas to handle concurrency safely
		if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
			return nil, fmt.Errorf("failed to set journal_mode to WAL: %w", err)
		}
		if err := db.Exec("PRAGMA busy_timeout=5000;").Error; err != nil {
			return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
		}

		// SQLite works best under high concurrency with 1 open writer connection
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		// PostgreSQL connection pooling config
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(10 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&TriageEntry{}, &SystemMetadata{}, &PushSubscription{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Migrate status 'NOT_AVAILABLE' to 'UNAVAILABLE' for backward compatibility
	if err := db.Model(&TriageEntry{}).Where("status = ?", "NOT_AVAILABLE").Update("status", "UNAVAILABLE").Error; err != nil {
		slog.Warn("Failed to migrate NOT_AVAILABLE statuses to UNAVAILABLE", slog.Any("error", err))
	}

	slog.Info("Connected and migrated database", slog.String("driver", cfg.DBDriver))
	return db, nil
}

// SlogLogger bridges GORM logging to Go's standard log/slog.
type SlogLogger struct {
	LogLevel logger.LogLevel
}

// LogMode sets the logging level.
func (l *SlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs messages at info level.
func (l *SlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn logs messages at warn level.
func (l *SlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error logs messages at error level.
func (l *SlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL statements and timing.
func (l *SlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.ErrorContext(ctx, "GORM query error",
			slog.Any("err", err),
			slog.Duration("elapsed", elapsed),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
		return
	}

	// Warn on slow queries (longer than 200ms)
	if elapsed > 200*time.Millisecond && l.LogLevel >= logger.Warn {
		slog.WarnContext(ctx, "GORM slow query",
			slog.Duration("elapsed", elapsed),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
		return
	}

	if l.LogLevel >= logger.Info {
		slog.DebugContext(ctx, "GORM query",
			slog.Duration("elapsed", elapsed),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	}
}
