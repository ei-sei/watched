package handler

import (
	"net/http"
	"strconv"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ListHandler struct {
	lists    *repository.ListRepo
	media    *repository.MediaRepo
	validate *validator.Validate
}

func NewListHandler(lists *repository.ListRepo, media *repository.MediaRepo) *ListHandler {
	return &ListHandler{lists: lists, media: media, validate: validator.New()}
}

// GET /lists
func (h *ListHandler) List(w http.ResponseWriter, r *http.Request) {
	lists, err := h.lists.List(r.Context(), userIDFrom(r))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if lists == nil {
		lists = []models.UserList{}
	}
	jsonOK(w, lists)
}

// POST /lists
func (h *ListHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"        validate:"required,max=200"`
		Description *string `json:"description"`
		IsPublic    bool    `json:"is_public"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	list, err := h.lists.Create(r.Context(), userIDFrom(r), body.Name, body.Description, body.IsPublic)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonCreated(w, list)
}

// GET /lists/{id}
func (h *ListHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	// Allow public lists to be read by anyone; private lists only by owner
	if !list.IsPublic && list.UserID != userIDFrom(r) {
		jsonErr(w, http.StatusForbidden, "forbidden")
		return
	}
	jsonOK(w, list)
}

// PATCH /lists/{id}
func (h *ListHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	list, err := h.lists.Update(r.Context(), id, userIDFrom(r), body.Name, body.Description, body.IsPublic)
	if err != nil || list == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	jsonOK(w, list)
}

// DELETE /lists/{id}
func (h *ListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.lists.Delete(r.Context(), id, userIDFrom(r)); err != nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /lists/{id}/items
func (h *ListHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		MediaItemID int `json:"media_item_id" validate:"required"`
		Position    int `json:"position"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Verify list belongs to user
	list, err := h.lists.GetByID(r.Context(), id)
	if err != nil || list == nil || list.UserID != userIDFrom(r) {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	// Verify media item belongs to user
	item, err := h.media.GetByID(r.Context(), body.MediaItemID, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "media item not found")
		return
	}

	li, err := h.lists.AddItem(r.Context(), id, body.MediaItemID, body.Position)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if li == nil {
		jsonErr(w, http.StatusConflict, "item already in list")
		return
	}
	jsonCreated(w, li)
}

// DELETE /lists/{id}/items/{mediaID}
func (h *ListHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	mediaID, err := strconv.Atoi(chi.URLParam(r, "mediaID"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid media id")
		return
	}

	list, err := h.lists.GetByID(r.Context(), id)
	if err != nil || list == nil || list.UserID != userIDFrom(r) {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.lists.RemoveItem(r.Context(), id, mediaID); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /lists/{id}/items/order
func (h *ListHandler) ReorderItems(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Order []int `json:"order" validate:"required"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	list, err := h.lists.GetByID(r.Context(), id)
	if err != nil || list == nil || list.UserID != userIDFrom(r) {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.lists.ReorderItems(r.Context(), id, body.Order); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
