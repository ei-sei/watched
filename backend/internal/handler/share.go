package handler

import (
	"net/http"
	"strconv"

	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ShareHandler struct {
	lists *repository.ListRepo
}

func NewShareHandler(lists *repository.ListRepo) *ShareHandler {
	return &ShareHandler{lists: lists}
}

// GET /share/lists/{id} — no auth required, only returns public lists
func (h *ShareHandler) GetList(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	list, err := h.lists.GetByID(r.Context(), id)
	if err != nil || list == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if !list.IsPublic {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	jsonOK(w, list)
}
