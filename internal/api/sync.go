package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"limbo/internal/scanner"
)

// handleSync triggers an immediate background scan.
func handleSync(s *scanner.Scanner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slog.InfoContext(ctx, "Manual sync triggered via API")

		s.TriggerScan(ctx)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"message":  "Sync triggered",
			"lastScan": s.LastScanTime(),
		})
	}
}
