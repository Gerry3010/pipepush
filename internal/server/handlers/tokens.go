package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type TokenHandler struct{ db *db.DB }

func NewTokenHandler(database *db.DB) *TokenHandler {
	return &TokenHandler{db: database}
}

// HashToken returns the hex-encoded SHA-256 of a plaintext token.
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var req models.CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EncryptedName == "" || req.ProjectID == "" {
		writeError(w, http.StatusBadRequest, "encryptedName and projectId are required")
		return
	}

	// Verify project ownership
	proj, err := h.db.GetProject(r.Context(), req.ProjectID, uid)
	if err != nil || proj == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Generate a 32-byte random token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}
	plaintext := "pp_" + base64.RawURLEncoding.EncodeToString(raw)
	tokenHash := HashToken(plaintext)

	t, err := h.db.CreateNotificationToken(r.Context(), uid, req.ProjectID, req.PipelineID, req.EncryptedName, tokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("could not create token: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, models.CreateTokenResponse{
		Token:          *t,
		PlaintextToken: plaintext,
	})
}

func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	projectID := chi.URLParam(r, "projectID")
	tokens, err := h.db.ListTokens(r.Context(), projectID, uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list tokens")
		return
	}
	if tokens == nil {
		tokens = []*models.NotificationToken{}
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (h *TokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.db.RevokeToken(r.Context(), id, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete permanently removes a token (as opposed to Revoke, which deactivates it).
func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.db.DeleteToken(r.Context(), id, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
