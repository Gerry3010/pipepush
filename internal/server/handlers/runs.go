package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

type RunHandler struct{ db *db.DB }

func NewRunHandler(database *db.DB) *RunHandler {
	return &RunHandler{db: database}
}

func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	pipelineID := chi.URLParam(r, "pipelineID")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	runs, err := h.db.ListRuns(r.Context(), pipelineID, uid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list runs")
		return
	}
	if runs == nil {
		runs = []*models.Run{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *RunHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	run, err := h.db.GetRun(r.Context(), id, uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}
