package handler

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/models"
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

	target, err := h.users.GetByID(r.Context(), id)
	if err != nil || target == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if target.Username == "admin" {
		jsonErr(w, http.StatusForbidden, "cannot modify the admin account")
		return
	}

	user, err := h.users.UpdateFlags(r.Context(), id, body.IsPremium, body.IsAdmin)
	if err != nil || user == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	jsonOK(w, user)
}

// DELETE /admin/users/{id}
func (h *UserHandler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	target, err := h.users.GetByID(r.Context(), id)
	if err != nil || target == nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if target.Username == "admin" {
		jsonErr(w, http.StatusForbidden, "cannot delete the admin account")
		return
	}
	if err := h.users.Delete(r.Context(), id); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// readMemInfo parses /proc/meminfo and returns MemTotal, MemAvailable in kB.
// Returns zeros silently if the file is unavailable (non-Linux hosts).
func readMemInfo() (totalKB, availableKB int64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		var key string
		var val int64
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key = strings.TrimSuffix(fields[0], ":")
		val, err = strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "MemTotal":
			totalKB = val
		case "MemAvailable":
			availableKB = val
		}
		if totalKB > 0 && availableKB > 0 {
			break
		}
	}
	return totalKB, availableKB
}

// readDiskUsage returns total and used bytes for the filesystem at the given path.
// Returns zeros silently on error.
func readDiskUsage(path string) (totalBytes, usedBytes int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total := int64(stat.Blocks) * stat.Bsize
	free  := int64(stat.Bavail) * stat.Bsize
	return total, total - free
}

// GET /admin/stats
func (h *UserHandler) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.users.GetAdminStats(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	stats.MemTotalKB, stats.MemAvailableKB = readMemInfo()
	stats.MemUsedKB = stats.MemTotalKB - stats.MemAvailableKB
	stats.DiskTotalBytes, stats.DiskUsedBytes = readDiskUsage("/")
	stats.DBSizeLimitBytes = h.cfg.DBSizeLimitBytes
	jsonOK(w, stats)
}

// GET /admin/invites
func (h *UserHandler) AdminListInvites(w http.ResponseWriter, r *http.Request) {
	codes, err := h.users.ListInvites(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if codes == nil {
		codes = []models.InviteCode{}
	}
	jsonOK(w, codes)
}

// DELETE /admin/invites/{code}
func (h *UserHandler) AdminDeleteInvite(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.users.DeleteInvite(r.Context(), code); err != nil {
		jsonErr(w, http.StatusNotFound, "invite code not found or already used")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

