package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"limbo/internal/api"
	"limbo/internal/config"
	"limbo/internal/database"
	"limbo/internal/scanner"
	"limbo/internal/seerr"
)

//go:embed frontend/*
var frontendFiles embed.FS

func main() {
	// Load configuration first
	cfg := config.Load()

	// Parse and configure slog logging
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))

	slog.Info("🌀 Limbo starting...", slog.String("version", api.Version))

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("Configuration validation failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Configuration validated successfully")

	// Initialize database
	db, err := database.Init(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("[DB] Initialized successfully")

	// Initialize Seerr client
	seerrClient := seerr.NewClient(cfg)

	// Initialize scanner
	scan := scanner.New(cfg, db, seerrClient)

	// Setup frontend filesystem (strip the "frontend" prefix)
	frontendFS, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		slog.Error("Failed to setup frontend filesystem", slog.Any("error", err))
		os.Exit(1)
	}

	// Setup router
	handlerHTTP := api.NewRouter(api.Deps{
		Config:  cfg,
		DB:      db,
		Scanner: scan,
		Seerr:   seerrClient,
	}, http.FS(frontendFS))

	// HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handlerHTTP,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background scanner
	go scan.Run(ctx)

	// Start HTTP server
	go func() {
		slog.Info(fmt.Sprintf("🌐 HTTP server listening on :%s", cfg.Port), slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("⛔ Received signal, shutting down...", slog.String("signal", sig.String()))

	// Cancel context to stop scanner
	cancel()

	// Graceful HTTP shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", slog.Any("error", err))
	}

	slog.Info("👋 Limbo stopped")
}
