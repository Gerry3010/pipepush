package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/push"
	"github.com/Gerry3010/pipepush/internal/routing"
)

// validStatuses are the accepted pipeline statuses.
var validStatuses = map[string]bool{
	"success":   true,
	"failure":   true,
	"cancelled": true,
	"running":   true,
	"skipped":   true,
}

// maxLogBytes bounds how much log output we keep per run so encrypted payloads
// (and the DB) don't grow without limit. We keep the tail — the end of CI output
// is usually the most relevant (errors, summary).
const maxLogBytes = 16 * 1024

func capLogs(s string) string {
	if len(s) <= maxLogBytes {
		return s
	}
	return "…(truncated)\n" + s[len(s)-maxLogBytes:]
}

type WebhookHandler struct {
	db         *db.DB
	dispatcher *push.Dispatcher
	hub        *SSEHub
}

func NewWebhookHandler(database *db.DB, dispatcher *push.Dispatcher, hub *SSEHub) *WebhookHandler {
	return &WebhookHandler{db: database, dispatcher: dispatcher, hub: hub}
}

// Handle receives a webhook from a CI/CD pipeline, encrypts the payload with the
// owning user's public key (E2E), stores the run, and dispatches notifications.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req models.WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status must be one of: success, failure, cancelled, running, skipped")
		return
	}

	// Look up token by hash — server never stores the plaintext.
	tokenHash := HashToken(req.Token)
	token, publicKeyB64, err := h.db.GetTokenByHash(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if token == nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	pubKey, err := crypto.PublicKeyFromBase64(publicKeyB64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user key error")
		return
	}

	// Resolve target pipeline. A pipeline-scoped token routes to its bound
	// pipeline. A project-scoped token routes by the plaintext pipeline name:
	// match an existing pipeline via its routing key, else create one. Names
	// stay E2E-encrypted; only their hash (the routing key) is used to match.
	pipelineID := token.PipelineID
	if pipelineID == "" {
		if req.Pipeline == "" {
			writeError(w, http.StatusBadRequest, "this token is project-scoped; the request must include a \"pipeline\" name")
			return
		}
		rk := routing.Key(req.Pipeline)
		pipe, err := h.db.GetPipelineByRoutingKey(r.Context(), token.ProjectID, rk)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not resolve pipeline")
			return
		}
		if pipe == nil {
			encName, encErr := crypto.Encrypt(pubKey, []byte(req.Pipeline))
			if encErr != nil {
				writeError(w, http.StatusInternalServerError, "encryption failed")
				return
			}
			pipe, err = h.db.CreatePipeline(r.Context(), token.UserID, token.ProjectID, encName, rk)
			if err != nil {
				// A concurrent first-run for the same name may have created it
				// (unique on project_id+routing_key). Re-resolve before failing.
				pipe, _ = h.db.GetPipelineByRoutingKey(r.Context(), token.ProjectID, rk)
				if pipe == nil {
					writeError(w, http.StatusInternalServerError, "could not create pipeline")
					return
				}
			}
		}
		pipelineID = pipe.ID
	}

	// Build the plaintext payload and ECIES-encrypt it for the user.
	payload := models.RunPayload{
		Status:   req.Status,
		Pipeline: req.Pipeline,
		RunID:    req.RunID,
		Commit:   req.Commit,
		Branch:   req.Branch,
		Duration: req.Duration,
		Message:  req.Message,
		Logs:     capLogs(req.Logs),
	}
	plaintext, _ := json.Marshal(payload)

	encPayload, err := crypto.Encrypt(pubKey, plaintext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	run, err := h.db.CreateRun(r.Context(), token.UserID, token.ProjectID, pipelineID, token.ID, req.Status, encPayload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store run")
		return
	}

	// Mark token usage (best-effort).
	_ = h.db.TouchToken(r.Context(), token.ID)

	// Fan out: SSE for connected CLIs/web clients, Web Push for offline devices.
	h.hub.Broadcast(token.UserID, models.SSEEvent{
		Type:             "run_update",
		RunID:            run.ID,
		EncryptedPayload: encPayload,
		ReceivedAt:       run.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
	})

	go func() {
		ctx := context.Background()
		h.dispatcher.SendRunNotification(ctx, token.UserID, run)
	}()

	slog.Info("webhook processed", "user", token.UserID, "pipeline", pipelineID, "status", req.Status)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted", "runId": run.ID})
}
