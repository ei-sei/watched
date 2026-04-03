package handler

import (
	"net/http"
	"time"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	users    *repository.UserRepo
	cfg      *config.Config
	validate *validator.Validate
}

func NewAuthHandler(users *repository.UserRepo, cfg *config.Config) *AuthHandler {
	return &AuthHandler{users: users, cfg: cfg, validate: validator.New()}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username   string `json:"username"    validate:"required,min=3,max=50,alphanum"`
		Password   string `json:"password"    validate:"required,min=8"`
		InviteCode string `json:"invite_code" validate:"required"`
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

	if err := h.users.UseInvite(ctx, body.InviteCode); err != nil {
		jsonErr(w, http.StatusForbidden, "invalid or already used invite code")
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	user, err := h.users.Create(ctx, body.Username, hash)
	if err != nil {
		jsonErr(w, http.StatusConflict, "username or email already taken")
		return
	}

	jsonCreated(w, user)
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
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

	user, err := h.users.GetByUsername(ctx, body.Username)
	if err != nil || user == nil {
		jsonErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Lockout check
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		jsonErr(w, http.StatusTooManyRequests, "account locked, try later")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, body.Password) {
		attempts := user.FailedAttempts + 1
		var lockUntil *time.Time
		if attempts >= 5 {
			t := time.Now().Add(15 * time.Minute)
			lockUntil = &t
		}
		_ = h.users.UpdateLoginFail(ctx, user.ID, attempts, lockUntil)
		jsonErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Reset failed attempts on success
	_ = h.users.UpdateLoginFail(ctx, user.ID, 0, nil)

	access, err := auth.NewAccessToken(h.cfg.JWTSecret, user.ID, user.IsAdmin)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	refresh, err := auth.NewRefreshToken(h.cfg.JWTSecret, user.ID, user.IsAdmin)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/auth/refresh",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.RefreshTokenDuration.Seconds()),
	})

	jsonOK(w, map[string]string{"access_token": access})
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		jsonErr(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	claims, err := auth.ParseToken(h.cfg.JWTSecret, cookie.Value)
	if err != nil || claims.Kind != "refresh" {
		jsonErr(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	access, err := auth.NewAccessToken(h.cfg.JWTSecret, claims.UserID, claims.IsAdmin)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	jsonOK(w, map[string]string{"access_token": access})
}

// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Path:     "/auth/refresh",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
