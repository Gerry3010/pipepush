package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type SettingsHandler struct{ db *db.DB }

func NewSettingsHandler(database *db.DB) *SettingsHandler {
	return &SettingsHandler{db: database}
}

// allowedRetention whitelists the retention windows (in hours) the UI offers.
// A nil value ("keep forever") is always allowed and handled separately.
var allowedRetention = map[int]bool{
	6:   true, // 6 hours
	24:  true, // 1 day
	72:  true, // 3 days
	168: true, // 1 week
	336: true, // 2 weeks
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	hours, err := h.db.GetRetention(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	writeJSON(w, http.StatusOK, models.SettingsResponse{RetentionHours: hours})
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var req models.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RetentionHours != nil && !allowedRetention[*req.RetentionHours] {
		writeError(w, http.StatusBadRequest, "invalid retention value")
		return
	}
	if err := h.db.SetRetention(r.Context(), uid, req.RetentionHours); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save settings")
		return
	}
	writeJSON(w, http.StatusOK, models.SettingsResponse{RetentionHours: req.RetentionHours})
}
