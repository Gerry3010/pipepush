package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type ProjectHandler struct{ db *db.DB }

func NewProjectHandler(database *db.DB) *ProjectHandler {
	return &ProjectHandler{db: database}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	projects, err := h.db.ListProjects(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list projects")
		return
	}
	if projects == nil {
		projects = []*models.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EncryptedName == "" {
		writeError(w, http.StatusBadRequest, "encryptedName is required")
		return
	}
	p, err := h.db.CreateProject(r.Context(), uid, req.EncryptedName, req.EncryptedDescription)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.db.DeleteProject(r.Context(), id, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
