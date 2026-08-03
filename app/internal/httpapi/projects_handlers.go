package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"niu/internal/auth"
	"niu/internal/projects"
)

type addProjectRequest struct {
	Name       string  `json:"name"`
	Budget     *string `json:"budget"`
	TargetDate *string `json:"target_date"`
}

// handleListProjects is GET /api/v1/projects — read-only, never mutates
// (EC-10/NFR-04).
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	list, err := s.projects.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": toProjectDTOs(list)})
}

// handleCreateProject is POST /api/v1/projects.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	var req addProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Cos de la petició no vàlid.")
		return
	}

	project, err := s.projects.Add(r.Context(), user.ID, req.Name, req.Budget, req.TargetDate)
	if err != nil {
		var dup projects.ErrDuplicate
		var val projects.ErrValidation
		switch {
		case errors.As(err, &dup):
			msg := "«" + trimForMessage(req.Name) + "» ja existeix."
			writeError(w, http.StatusConflict, "duplicate_project", msg)
		case errors.As(err, &val):
			writeError(w, http.StatusBadRequest, "validation_failed", val.Message)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toProjectDTO(project))
}

type patchProjectRequest struct {
	State string `json:"state"`
}

// handlePatchProjectState is PATCH /api/v1/projects/{id} — moves a
// project to an absolute state (never a toggle), in any direction (AC-09).
func (s *Server) handlePatchProjectState(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Identificador de projecte no vàlid.")
		return
	}

	var req patchProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Cos de la petició no vàlid.")
		return
	}

	project, err := s.projects.ChangeState(r.Context(), user.ID, id, projects.State(req.State))
	if err != nil {
		var notFound projects.ErrNotFound
		var val projects.ErrValidation
		switch {
		case errors.As(err, &notFound):
			writeError(w, http.StatusNotFound, "not_found", "El projecte ja no existeix.")
		case errors.As(err, &val):
			writeError(w, http.StatusBadRequest, "validation_failed", val.Message)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		}
		return
	}

	writeJSON(w, http.StatusOK, toProjectDTO(project))
}

// handleDeleteProject is DELETE /api/v1/projects/{id} — idempotent
// (EC-13).
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Identificador de projecte no vàlid.")
		return
	}

	if err := s.projects.Delete(r.Context(), user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
