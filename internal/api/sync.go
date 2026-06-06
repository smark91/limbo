package api

import (
	"log/slog"
	"net/http"

	"github.com/smark91/limbo/internal/scanner"
)

// handleSync triggers an immediate background scan.
func handleSync(s *scanner.Scanner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slog.InfoContext(ctx, "Manual sync triggered via API")

		if err := s.TriggerScan(ctx); err != nil {
			slog.ErrorContext(ctx, "Manual sync failed", slog.Any("error", err))
			writeJSON(w, r, http.StatusBadGateway, map[string]interface{}{
				"success": false,
				"error":   "Failed to sync with Seerr: " + err.Error(),
			})
			return
		}

		writeJSON(w, r, http.StatusOK, map[string]interface{}{
			"success":  true,
			"message":  "Sync triggered",
			"lastScan": s.LastScanTime(),
		})
	}
}
