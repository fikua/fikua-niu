package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"niu/internal/auth"
	"niu/internal/ideas"
)

type addIdeaRequest struct {
	URL string `json:"url"`
}

// handleListIdeas is GET /api/v1/ideas — read-only, never mutates, never
// triggers a re-scrape (EC-13/NFR-03/NFR-09).
func (s *Server) handleListIdeas(w http.ResponseWriter, r *http.Request) {
	list, err := s.ideas.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ideas": toIdeaDTOs(list)})
}

// handleCreateIdea is POST /api/v1/ideas. Responds 201 immediately with
// the idea in preview_status='pending' (ADR-03) — the scrape resolves in
// the background; this handler never waits for it.
func (s *Server) handleCreateIdea(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	var req addIdeaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Cos de la petició no vàlid.")
		return
	}

	idea, err := s.ideas.Add(r.Context(), user.ID, req.URL)
	if err != nil {
		var val ideas.ErrValidation
		switch {
		case errors.As(err, &val):
			writeError(w, http.StatusBadRequest, "validation_failed", val.Message)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toIdeaDTO(idea))
}

// handleDeleteIdea is DELETE /api/v1/ideas/{id} — idempotent (EC-15),
// always responds 204.
func (s *Server) handleDeleteIdea(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Identificador d'idea no vàlid.")
		return
	}

	if err := s.ideas.Delete(r.Context(), user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
