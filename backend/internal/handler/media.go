package handler

import (
	"net/http"
	"strconv"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/models"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type MediaHandler struct {
	media    *repository.MediaRepo
	episodes *repository.EpisodeRepo
	chapters *repository.ChapterRepo
	validate *validator.Validate
}

func NewMediaHandler(media *repository.MediaRepo, episodes *repository.EpisodeRepo, chapters *repository.ChapterRepo) *MediaHandler {
	return &MediaHandler{media: media, episodes: episodes, chapters: chapters, validate: validator.New()}
}

func userIDFrom(r *http.Request) int {
	return auth.ClaimsFrom(r.Context()).UserID
}

// GET /media
func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := repository.MediaFilter{}

	if mt := q.Get("media_type"); mt != "" {
		v := models.MediaType(mt)
		f.MediaType = &v
	}
	if st := q.Get("status"); st != "" {
		v := models.MediaStatus(st)
		f.Status = &v
	}
	if search := q.Get("q"); search != "" {
		f.Search = &search
	}
	f.Sort = q.Get("sort")
	f.Order = q.Get("order")
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PerPage, _ = strconv.Atoi(q.Get("per_page"))

	result, err := h.media.List(r.Context(), userIDFrom(r), f)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, result)
}

// POST /media
func (h *MediaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MediaType  models.MediaType   `json:"media_type"  validate:"required,oneof=film tv_show book anime"`
		ExternalID *string            `json:"external_id"`
		Title      string             `json:"title"       validate:"required,max=500"`
		Year       *int               `json:"year"`
		PosterURL  *string            `json:"poster_url"`
		Metadata   map[string]any     `json:"metadata"`
		Status     models.MediaStatus `json:"status"      validate:"omitempty,oneof=want_to in_progress completed dropped on_hold"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if body.Status == "" {
		body.Status = models.StatusWantTo
	}

	item, err := h.media.Create(r.Context(), repository.CreateMediaInput{
		UserID:     userIDFrom(r),
		MediaType:  body.MediaType,
		ExternalID: body.ExternalID,
		Title:      body.Title,
		Year:       body.Year,
		PosterURL:  body.PosterURL,
		Metadata:   body.Metadata,
		Status:     body.Status,
	})
	if err != nil {
		jsonErr(w, http.StatusConflict, "item already in library")
		return
	}
	jsonCreated(w, item)
}

// GET /media/{id}
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	jsonOK(w, item)
}

// PATCH /media/{id}
func (h *MediaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Status      *models.MediaStatus `json:"status"`
		Rating      *float64            `json:"rating"`
		ReviewText  *string             `json:"review_text"`
		StartedAt   *string             `json:"started_at"`
		CompletedAt *string             `json:"completed_at"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	item, err := h.media.Update(r.Context(), id, userIDFrom(r), repository.UpdateMediaInput{
		Status:      body.Status,
		Rating:      body.Rating,
		ReviewText:  body.ReviewText,
		StartedAt:   body.StartedAt,
		CompletedAt: body.CompletedAt,
	})
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	jsonOK(w, item)
}

// DELETE /media/{id}
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.media.Delete(r.Context(), id, userIDFrom(r)); err != nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /media/{id}/episodes
func (h *MediaHandler) ListEpisodes(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	// Verify ownership
	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	logs, err := h.episodes.List(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if logs == nil {
		logs = []models.TvEpisodeLog{}
	}
	jsonOK(w, logs)
}

// PUT /media/{id}/episodes
func (h *MediaHandler) UpsertEpisode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Season    int      `json:"season"  validate:"required,min=1"`
		Episode   int      `json:"episode" validate:"required,min=1"`
		WatchedAt *string  `json:"watched_at"`
		Rating    *float64 `json:"rating"`
		Note      *string  `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Verify ownership
	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	log, err := h.episodes.Upsert(r.Context(), id, body.Season, body.Episode, body.WatchedAt, body.Rating, body.Note)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, log)
}

// DELETE /media/{id}/episodes/{epID}
func (h *MediaHandler) DeleteEpisode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	epID, err := strconv.Atoi(chi.URLParam(r, "epID"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid episode id")
		return
	}

	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.episodes.Delete(r.Context(), epID, id); err != nil {
		jsonErr(w, http.StatusNotFound, "episode not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /media/{id}/chapters
func (h *MediaHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	logs, err := h.chapters.List(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if logs == nil {
		logs = []models.BookChapterLog{}
	}
	jsonOK(w, logs)
}

// PUT /media/{id}/chapters
func (h *MediaHandler) UpsertChapter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		ChapterNumber int                  `json:"chapter_number" validate:"required,min=1"`
		ChapterTitle  *string              `json:"chapter_title"`
		StartPage     *int                 `json:"start_page"`
		EndPage       *int                 `json:"end_page"`
		Status        models.ChapterStatus `json:"status" validate:"required,oneof=unread in_progress completed"`
		Note          *string              `json:"note"`
		StartedAt     *string              `json:"started_at"`
		CompletedAt   *string              `json:"completed_at"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	log, err := h.chapters.Upsert(r.Context(), id, repository.UpsertChapterInput{
		ChapterNumber: body.ChapterNumber,
		ChapterTitle:  body.ChapterTitle,
		StartPage:     body.StartPage,
		EndPage:       body.EndPage,
		Status:        body.Status,
		Note:          body.Note,
		StartedAt:     body.StartedAt,
		CompletedAt:   body.CompletedAt,
	})
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, log)
}

// DELETE /media/{id}/chapters/{chID}
func (h *MediaHandler) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	chID, err := strconv.Atoi(chi.URLParam(r, "chID"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid chapter id")
		return
	}

	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.chapters.Delete(r.Context(), chID, id); err != nil {
		jsonErr(w, http.StatusNotFound, "chapter not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
