package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Gerry3010/pipepush/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ErrorResponse{Error: msg})
}

func isDuplicateError(err error) bool {
	// pgx returns error messages containing "duplicate key" for UNIQUE violations
	return strings.Contains(err.Error(), "duplicate key")
}
