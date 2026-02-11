package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type RewatchHandler struct {
	rewatches *repository.RewatchRepo
	media     *repository.MediaRepo
}

func NewRewatchHandler(rewatches *repository.RewatchRepo, media *repository.MediaRepo) *RewatchHandler {
	return &RewatchHandler{rewatches: rewatches, media: media}
}

// GET /media/{id}/rewatches
func (h *RewatchHandler) List(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := auth.ClaimsFrom(r.Context()).UserID

	// Verify ownership
	item, err := h.media.GetByID(r.Context(), id, userID)
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	list, err := h.rewatches.List(r.Context(), userID, id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, list)
}

// POST /media/{id}/rewatches
func (h *RewatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := auth.ClaimsFrom(r.Context()).UserID

	// Verify ownership
	item, err := h.media.GetByID(r.Context(), id, userID)
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		StartedAt  *string `json:"started_at"`
		FinishedAt *string `json:"finished_at"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rw, err := h.rewatches.Create(r.Context(), userID, id, body.StartedAt, body.FinishedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			jsonErr(w, http.StatusConflict, "already logged for this date")
			return
		}
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonCreated(w, rw)
}

// DELETE /rewatches/{id}
func (h *RewatchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := auth.ClaimsFrom(r.Context()).UserID

	if err := h.rewatches.Delete(r.Context(), userID, id); err != nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
