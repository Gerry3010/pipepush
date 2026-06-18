package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type PushHandler struct {
	db          *db.DB
	vapidPublic string
}

func NewPushHandler(database *db.DB, vapidPublic string) *PushHandler {
	return &PushHandler{db: database, vapidPublic: vapidPublic}
}

// VAPIDPublicKey returns the server's VAPID public key for client subscription.
func (h *PushHandler) VAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"publicKey": h.vapidPublic})
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var req models.PushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" || req.P256DHKey == "" || req.AuthKey == "" {
		writeError(w, http.StatusBadRequest, "endpoint, p256dhKey and authKey are required")
		return
	}
	if err := h.db.UpsertVAPIDSubscription(r.Context(), uid, req.Endpoint, req.P256DHKey, req.AuthKey, req.DeviceName); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save subscription")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var req models.PushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		writeError(w, http.StatusBadRequest, "endpoint is required")
		return
	}
	if err := h.db.DeleteVAPIDSubscription(r.Context(), uid, req.Endpoint); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
