package handler

import (
	"net/http"
	"strconv"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	users    *repository.UserRepo
	media    *repository.MediaRepo
	cfg      *config.Config
	validate *validator.Validate
}

func NewUserHandler(users *repository.UserRepo, media *repository.MediaRepo, cfg *config.Config) *UserHandler {
	return &UserHandler{users: users, media: media, cfg: cfg, validate: validator.New()}
}

// GET /users/me
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFrom(r.Context())
	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	jsonOK(w, user)
}

// PATCH /users/me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	claims := auth.ClaimsFrom(r.Context())

	if body.IsPublic != nil {
		user, err := h.users.UpdatePublic(r.Context(), claims.UserID, *body.IsPublic)
		if err != nil || user == nil {
			jsonErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		// If only toggling public, return early
		if body.DisplayName == nil && body.AvatarURL == nil {
			jsonOK(w, user)
			return
		}
	}

	user, err := h.users.UpdateProfile(r.Context(), claims.UserID, body.DisplayName, body.AvatarURL)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, user)
}

// PUT /users/me/password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password"     validate:"required,min=8"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	ctx := r.Context()
	claims := auth.ClaimsFrom(ctx)

	user, err := h.users.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, body.CurrentPassword) {
		jsonErr(w, http.StatusUnauthorized, "incorrect current password")
		return
	}

	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.users.UpdatePassword(ctx, claims.UserID, hash); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /u/:username — public profile
func (h *UserHandler) PublicProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil || user == nil || !user.IsPublic {
		jsonErr(w, http.StatusNotFound, "profile not found")
		return
	}

	result, err := h.media.List(r.Context(), user.ID, repository.MediaFilter{NoLimit: true})
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	jsonOK(w, map[string]any{
		"username":     user.Username,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
		"media":        result.Items,
	})
}

// --- Admin routes ---

// GET /admin/users
func (h *UserHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, users)
}

// PATCH /admin/users/{id}/flags
func (h *UserHandler) AdminUpdateFlags(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		IsPremium *bool `json:"is_premium"`
		IsAdmin   *bool `json:"is_admin"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	user, err := h.users.UpdateFlags(r.Context(), id, body.IsPremium, body.IsAdmin)
	if err != nil || user == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	jsonOK(w, user)
}

// POST /admin/invites
func (h *UserHandler) AdminCreateInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code" validate:"required,min=8,max=32,alphanum"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if err := h.users.CreateInvite(r.Context(), body.Code); err != nil {
		jsonErr(w, http.StatusConflict, "code already exists")
		return
	}
	jsonCreated(w, map[string]string{"code": body.Code})
}

