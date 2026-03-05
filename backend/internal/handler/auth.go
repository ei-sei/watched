package handler

import (
	"log"
	"net/http"
	"strings"
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

	body.Username = strings.ToLower(body.Username)
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

	body.Username = strings.ToLower(body.Username)
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
		if err := h.users.UpdateLoginFail(ctx, user.ID, attempts, lockUntil); err != nil {
			log.Printf("UpdateLoginFail user %d: %v", user.ID, err)
		}
		jsonErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Reset failed attempts on success
	if err := h.users.UpdateLoginFail(ctx, user.ID, 0, nil); err != nil {
		log.Printf("UpdateLoginFail reset user %d: %v", user.ID, err)
	}

	access, err := auth.NewAccessToken(h.cfg.JWTSecret, user.Username, user.ID, user.IsAdmin, user.IsPremium)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	refresh, err := auth.NewRefreshToken(h.cfg.JWTSecret, user.Username, user.ID, user.IsAdmin, user.IsPremium)
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

	jsonOK(w, map[string]string{"access_token": access, "refresh_token": refresh})
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Cookie is primary; X-Refresh-Token header is fallback for iOS PWA
	// where the HttpOnly cookie is cleared when the app is closed.
	tokenStr := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		tokenStr = cookie.Value
	} else if h := r.Header.Get("X-Refresh-Token"); h != "" {
		tokenStr = h
	} else {
		jsonErr(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	claims, err := auth.ParseToken(h.cfg.JWTSecret, tokenStr)
	if err != nil || claims.Kind != "refresh" {
		jsonErr(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Re-read IsAdmin from DB so flag changes take effect on the next refresh
	// without requiring the user to log out.
	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		jsonErr(w, http.StatusUnauthorized, "user not found")
		return
	}

	access, err := auth.NewAccessToken(h.cfg.JWTSecret, user.Username, user.ID, user.IsAdmin, user.IsPremium)
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
