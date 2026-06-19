package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type PipelineHandler struct{ db *db.DB }

func NewPipelineHandler(database *db.DB) *PipelineHandler {
	return &PipelineHandler{db: database}
}

func (h *PipelineHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	projectID := chi.URLParam(r, "projectID")
	pipelines, err := h.db.ListPipelines(r.Context(), projectID, uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list pipelines")
		return
	}
	if pipelines == nil {
		pipelines = []*models.Pipeline{}
	}
	writeJSON(w, http.StatusOK, pipelines)
}

func (h *PipelineHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	projectID := chi.URLParam(r, "projectID")
	var req models.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EncryptedName == "" {
		writeError(w, http.StatusBadRequest, "encryptedName is required")
		return
	}
	// Verify project belongs to user
	proj, err := h.db.GetProject(r.Context(), projectID, uid)
	if err != nil || proj == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	p, err := h.db.CreatePipeline(r.Context(), uid, projectID, req.EncryptedName, req.RoutingKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create pipeline")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *PipelineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.db.DeletePipeline(r.Context(), id, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete pipeline")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
