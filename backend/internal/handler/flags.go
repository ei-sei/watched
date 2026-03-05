package handler

import (
	"net/http"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
)

type FlagsHandler struct {
	flags *repository.FlagsRepo
}

func NewFlagsHandler(flags *repository.FlagsRepo) *FlagsHandler {
	return &FlagsHandler{flags: flags}
}

// GET /admin/flags — superadmin only
func (h *FlagsHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFrom(r.Context())
	if claims == nil || claims.Username != "admin" {
		jsonErr(w, http.StatusForbidden, "forbidden")
		return
	}
	flags, err := h.flags.GetAll(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, flags)
}

// PATCH /admin/flags/{key} — superadmin only
func (h *FlagsHandler) Set(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFrom(r.Context())
	if claims == nil || claims.Username != "admin" {
		jsonErr(w, http.StatusForbidden, "forbidden")
		return
	}

	key := chi.URLParam(r, "key")
	if !repository.IsValidFlagKey(key) {
		jsonErr(w, http.StatusBadRequest, "unknown feature key")
		return
	}

	var body struct {
		IsPremium bool `json:"is_premium"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.flags.Set(r.Context(), key, body.IsPremium); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RequireFeature is a middleware that gates a route behind a feature flag.
// Admins always pass through. Non-admins are checked against the flag:
// if is_premium=true they need is_premium on their token; if is_premium=false anyone passes.
func RequireFeature(flagsRepo *repository.FlagsRepo, key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := auth.ClaimsFrom(r.Context())
			if c == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if c.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}
			isPremium, err := flagsRepo.IsPremium(r.Context(), key)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if isPremium && !c.IsPremium {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
