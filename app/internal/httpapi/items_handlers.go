package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"niu/internal/auth"
	"niu/internal/items"
)

// boxLabel returns the Catalan display name for a location, used in
// error messages (proposal.md §8.4.3).
func boxLabel(loc items.Location) string {
	if loc == items.LocationPantry {
		return "Rebost"
	}
	return "A comprar"
}

type addItemRequest struct {
	Name string `json:"name"`
}

// handleListItems is GET /api/v1/items — read-only, never mutates
// (EC-08/NFR-04).
func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	list, err := s.items.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": toItemDTOs(list)})
}

// handleCreateItem is POST /api/v1/items.
func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Cos de la petició no vàlid.")
		return
	}

	item, err := s.items.Add(r.Context(), user.ID, req.Name)
	if err != nil {
		var dup items.ErrDuplicate
		var val items.ErrValidation
		switch {
		case errors.As(err, &dup):
			msg := "«" + trimForMessage(req.Name) + "» ja hi és a " + boxLabel(dup.ExistingLocation) + "."
			writeError(w, http.StatusConflict, "duplicate_item", msg)
		case errors.As(err, &val):
			writeError(w, http.StatusBadRequest, "validation_failed", val.Message)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toItemDTO(item))
}

func trimForMessage(raw string) string {
	// Best-effort trim for the message only; validation already trims
	// before comparing. Avoids leaking leading/trailing whitespace back
	// into the user-facing duplicate message.
	start, end := 0, len(raw)
	for start < end && isSpaceByte(raw[start]) {
		start++
	}
	for end > start && isSpaceByte(raw[end-1]) {
		end--
	}
	return raw[start:end]
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

type patchItemRequest struct {
	Location string `json:"location"`
}

// handleUpdateItem is PATCH /api/v1/items/{id} — moves an item to an
// absolute location (never a toggle).
func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Identificador d'ítem no vàlid.")
		return
	}

	var req patchItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Cos de la petició no vàlid.")
		return
	}

	loc := items.Location(req.Location)
	if loc != items.LocationShopping && loc != items.LocationPantry {
		writeError(w, http.StatusBadRequest, "validation_failed", "Ubicació no vàlida.")
		return
	}

	item, err := s.items.Move(r.Context(), user.ID, id, loc)
	if err != nil {
		var notFound items.ErrNotFound
		if errors.As(err, &notFound) {
			writeError(w, http.StatusNotFound, "not_found", "L'ítem ja no existeix.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	writeJSON(w, http.StatusOK, toItemDTO(item))
}

// handleDeleteItem is DELETE /api/v1/items/{id} — idempotent (EC-11).
func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "Identificador d'ítem no vàlid.")
		return
	}

	if err := s.items.Delete(r.Context(), user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleMe is GET /api/v1/me — delegates to auth.Authenticator via the
// context injected by WithCurrentUser (ADR-03).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	full, err := s.items.CurrentUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           full.ID,
		"name":         full.Name,
		"display_name": full.DisplayName,
		"avatar_emoji": full.AvatarEmoji,
	})
}

// handleHealthz is GET /healthz — REL-03/NFR-08: reflects the real state
// of the SQLite dependency, not just process liveness.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.health.Healthy(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "internal_error", "Base de dades no disponible.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
