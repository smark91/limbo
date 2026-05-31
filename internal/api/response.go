package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON writes a JSON response, sets content-type headers, and logs any encoding errors.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode json response", slog.Any("error", err))
	}
}
