package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	client   *http.Client
	tmdbKey  string
}

func NewMediaHandler(media *repository.MediaRepo, episodes *repository.EpisodeRepo, chapters *repository.ChapterRepo, tmdbKey string) *MediaHandler {
	return &MediaHandler{
		media: media, episodes: episodes, chapters: chapters,
		validate: validator.New(),
		client:   &http.Client{Timeout: 8 * time.Second},
		tmdbKey:  tmdbKey,
	}
}

type tmdbTVData struct {
	TotalEpisodes *int
	Seasons       []map[string]any
}

func (h *MediaHandler) fetchJikanData(ctx context.Context, malID string) *tmdbTVData {
	u := fmt.Sprintf("https://api.jikan.moe/v4/anime/%s", malID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var raw struct {
		Data struct {
			Episodes *int `json:"episodes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || raw.Data.Episodes == nil || *raw.Data.Episodes == 0 {
		return nil
	}
	seasons := []map[string]any{
		{"season_number": 1, "episode_count": *raw.Data.Episodes},
	}
	return &tmdbTVData{TotalEpisodes: raw.Data.Episodes, Seasons: seasons}
}

func (h *MediaHandler) fetchTMDBData(ctx context.Context, tmdbID string) *tmdbTVData {
	if h.tmdbKey == "" {
		return nil
	}
	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s?api_key=%s", tmdbID, h.tmdbKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var raw struct {
		NumberOfEpisodes *int `json:"number_of_episodes"`
		Seasons          []struct {
			SeasonNumber int `json:"season_number"`
			EpisodeCount int `json:"episode_count"`
		} `json:"seasons"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil
	}
	seasons := make([]map[string]any, 0)
	for _, s := range raw.Seasons {
		if s.SeasonNumber == 0 {
			continue // skip specials
		}
		seasons = append(seasons, map[string]any{
			"season_number": s.SeasonNumber,
			"episode_count": s.EpisodeCount,
		})
	}
	return &tmdbTVData{TotalEpisodes: raw.NumberOfEpisodes, Seasons: seasons}
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
		MediaType       models.MediaType   `json:"media_type"  validate:"required,oneof=film tv_show book anime"`
		ExternalID      *string            `json:"external_id"`
		Title           string             `json:"title"       validate:"required,max=500"`
		Year            *int               `json:"year"`
		PosterURL       *string            `json:"poster_url"`
		Metadata        map[string]any     `json:"metadata"`
		Status          models.MediaStatus `json:"status"      validate:"omitempty,oneof=want_to in_progress completed dropped on_hold"`
		TotalProgress   *int               `json:"total_progress"`
		CurrentProgress *int               `json:"current_progress"`
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

	// Auto-fill total episodes and season breakdown from TMDB (TV) or Jikan (anime)
	if body.ExternalID != nil {
		var externalData *tmdbTVData
		if body.MediaType == models.MediaTypeTVShow {
			if tmdbID := strings.TrimPrefix(*body.ExternalID, "tmdb:"); tmdbID != *body.ExternalID {
				externalData = h.fetchTMDBData(r.Context(), tmdbID)
			}
		} else if body.MediaType == models.MediaTypeAnime {
			if malID := strings.TrimPrefix(*body.ExternalID, "mal:"); malID != *body.ExternalID {
				externalData = h.fetchJikanData(r.Context(), malID)
			}
		}
		if externalData != nil {
			if body.TotalProgress == nil {
				body.TotalProgress = externalData.TotalEpisodes
			}
			if len(externalData.Seasons) > 0 {
				if body.Metadata == nil {
					body.Metadata = map[string]any{}
				}
				body.Metadata["seasons"] = externalData.Seasons
			}
		}
	}

	item, err := h.media.Create(r.Context(), repository.CreateMediaInput{
		UserID:          userIDFrom(r),
		MediaType:       body.MediaType,
		ExternalID:      body.ExternalID,
		Title:           body.Title,
		Year:            body.Year,
		PosterURL:       body.PosterURL,
		Metadata:        body.Metadata,
		Status:          body.Status,
		TotalProgress:   body.TotalProgress,
		CurrentProgress: body.CurrentProgress,
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

// POST /media/{id}/refresh
func (h *MediaHandler) RefreshFromTMDB(w http.ResponseWriter, r *http.Request) {
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
	if item.ExternalID == nil {
		jsonErr(w, http.StatusBadRequest, "item has no external id")
		return
	}
	var tvData *tmdbTVData
	if item.MediaType == models.MediaTypeTVShow {
		tmdbID := strings.TrimPrefix(*item.ExternalID, "tmdb:")
		if tmdbID == *item.ExternalID {
			jsonErr(w, http.StatusBadRequest, "not a TMDB item")
			return
		}
		tvData = h.fetchTMDBData(r.Context(), tmdbID)
	} else if item.MediaType == models.MediaTypeAnime {
		malID := strings.TrimPrefix(*item.ExternalID, "mal:")
		if malID == *item.ExternalID {
			jsonErr(w, http.StatusBadRequest, "not a MAL item")
			return
		}
		tvData = h.fetchJikanData(r.Context(), malID)
	} else {
		jsonErr(w, http.StatusBadRequest, "only tv shows and anime can be refreshed")
		return
	}
	if tvData == nil || tvData.TotalEpisodes == nil {
		jsonErr(w, http.StatusServiceUnavailable, "could not fetch episode data")
		return
	}
	// Merge seasons into existing metadata
	meta := item.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	if len(tvData.Seasons) > 0 {
		meta["seasons"] = tvData.Seasons
	}
	updated, err := h.media.Update(r.Context(), id, userIDFrom(r), repository.UpdateMediaInput{
		TotalProgress: tvData.TotalEpisodes,
		Metadata:      meta,
	})
	if err != nil || updated == nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, updated)
}

// POST /media/{id}/seasons — add a new season to an anime entry from a MAL ID
func (h *MediaHandler) AddAnimeSeason(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		MalID int `json:"mal_id" validate:"required,min=1"`
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
	if item.MediaType != models.MediaTypeAnime {
		jsonErr(w, http.StatusBadRequest, "only anime entries support season linking")
		return
	}
	malID := fmt.Sprintf("%d", body.MalID)
	data := h.fetchJikanData(r.Context(), malID)
	if data == nil || data.TotalEpisodes == nil {
		jsonErr(w, http.StatusServiceUnavailable, "could not fetch episode data from MAL")
		return
	}
	meta := item.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	// Determine next season number
	existing := []map[string]any{}
	if raw, ok := meta["seasons"]; ok {
		if slice, ok := raw.([]any); ok {
			for _, s := range slice {
				if m, ok := s.(map[string]any); ok {
					existing = append(existing, m)
				}
			}
		} else if ms, ok := raw.([]map[string]any); ok {
			existing = ms
		}
	}
	nextSeason := len(existing) + 1
	existing = append(existing, map[string]any{
		"season_number": nextSeason,
		"episode_count": *data.TotalEpisodes,
		"mal_id":        body.MalID,
	})
	meta["seasons"] = existing
	// Recalculate total episodes
	total := 0
	for _, s := range existing {
		if ec, ok := s["episode_count"].(int); ok {
			total += ec
		} else if ec, ok := s["episode_count"].(float64); ok {
			total += int(ec)
		}
	}
	updated, err := h.media.Update(r.Context(), id, userIDFrom(r), repository.UpdateMediaInput{
		TotalProgress: &total,
		Metadata:      meta,
	})
	if err != nil || updated == nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, updated)
}

// DELETE /media/{id}/seasons/{seasonNum} — remove a season from an anime entry
func (h *MediaHandler) RemoveAnimeSeason(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	seasonNum, err := strconv.Atoi(chi.URLParam(r, "seasonNum"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid season number")
		return
	}
	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	meta := item.Metadata
	if meta == nil {
		jsonErr(w, http.StatusBadRequest, "no seasons to remove")
		return
	}
	raw, ok := meta["seasons"]
	if !ok {
		jsonErr(w, http.StatusBadRequest, "no seasons to remove")
		return
	}
	var seasons []map[string]any
	if slice, ok := raw.([]any); ok {
		for _, s := range slice {
			if m, ok := s.(map[string]any); ok {
				seasons = append(seasons, m)
			}
		}
	} else if ms, ok := raw.([]map[string]any); ok {
		seasons = ms
	}
	// Filter out the target season and renumber
	filtered := []map[string]any{}
	for _, s := range seasons {
		var sn int
		switch v := s["season_number"].(type) {
		case float64:
			sn = int(v)
		case int:
			sn = v
		}
		if sn != seasonNum {
			filtered = append(filtered, s)
		}
	}
	// Renumber remaining seasons
	for i := range filtered {
		filtered[i]["season_number"] = i + 1
	}
	meta["seasons"] = filtered
	total := 0
	for _, s := range filtered {
		if ec, ok := s["episode_count"].(int); ok {
			total += ec
		} else if ec, ok := s["episode_count"].(float64); ok {
			total += int(ec)
		}
	}
	updated, err := h.media.Update(r.Context(), id, userIDFrom(r), repository.UpdateMediaInput{
		TotalProgress: &total,
		Metadata:      meta,
	})
	if err != nil || updated == nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, updated)
}

// PATCH /media/{id}
func (h *MediaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Status          *models.MediaStatus `json:"status"`
		Rating          *float64            `json:"rating"`
		ReviewText      *string             `json:"review_text"`
		StartedAt       *string             `json:"started_at"`
		CompletedAt     *string             `json:"completed_at"`
		CurrentProgress *int                `json:"current_progress"`
		TotalProgress   *int                `json:"total_progress"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	item, err := h.media.Update(r.Context(), id, userIDFrom(r), repository.UpdateMediaInput{
		Status:          body.Status,
		Rating:          body.Rating,
		ReviewText:      body.ReviewText,
		StartedAt:       body.StartedAt,
		CompletedAt:     body.CompletedAt,
		CurrentProgress: body.CurrentProgress,
		TotalProgress:   body.TotalProgress,
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

	// Auto-update item status based on progress
	h.autoUpdateEpisodeStatus(r.Context(), item, userIDFrom(r))

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

	// Auto-update item status based on progress
	h.autoUpdateChapterStatus(r.Context(), item, body.Status, userIDFrom(r))

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

// POST /media/{id}/chapters/import
func (h *MediaHandler) ImportChapters(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Count int `json:"count"`
	}
	_ = decode(r, &body)

	item, err := h.media.GetByID(r.Context(), id, userIDFrom(r))
	if err != nil || item == nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}

	var inputs []repository.UpsertChapterInput
	source := "manual"

	// Try Open Library TOC if the book was added from Open Library
	if item.ExternalID != nil && strings.HasPrefix(*item.ExternalID, "ol:") {
		olKey := strings.TrimPrefix(*item.ExternalID, "ol:")
		if toc, err := fetchOLChapters(r.Context(), h.client, olKey); err == nil && len(toc) > 0 {
			inputs = toc
			source = "openlibrary"
		}
	}

	// Fallback: generate numbered chapters from count
	if len(inputs) == 0 {
		if body.Count < 1 || body.Count > 5000 {
			jsonErr(w, http.StatusBadRequest, "count must be between 1 and 5000")
			return
		}
		for i := 1; i <= body.Count; i++ {
			inputs = append(inputs, repository.UpsertChapterInput{
				ChapterNumber: i,
				Status:        models.ChapterUnread,
			})
		}
	}

	// Trim empty chapters that exceed the new count
	if err := h.chapters.DeleteEmptyAbove(r.Context(), id, len(inputs)); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	n, err := h.chapters.BulkUpsert(r.Context(), id, inputs)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, map[string]any{"source": source, "imported": n})
}

func fetchOLChapters(ctx context.Context, client *http.Client, olKey string) ([]repository.UpsertChapterInput, error) {
	// olKey is like /works/OL123W
	u := fmt.Sprintf("https://openlibrary.org%s.json", olKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		TableOfContents []struct {
			Title   string `json:"title"`
			Level   int    `json:"level"`
			PageNum string `json:"pagenum"`
		} `json:"table_of_contents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var inputs []repository.UpsertChapterInput
	chNum := 0
	for _, toc := range raw.TableOfContents {
		title := strings.TrimSpace(toc.Title)
		if title == "" {
			continue
		}
		chNum++
		t := title
		inputs = append(inputs, repository.UpsertChapterInput{
			ChapterNumber: chNum,
			ChapterTitle:  &t,
			Status:        models.ChapterUnread,
		})
	}
	return inputs, nil
}

// autoUpdateEpisodeStatus applies auto status rules after an episode is logged.
func (h *MediaHandler) autoUpdateEpisodeStatus(ctx context.Context, item *models.MediaItem, userID int) {
	today := time.Now().Format("2006-01-02")
	upd := repository.UpdateMediaInput{}
	changed := false

	// Rule 1: move to in_progress when starting from want_to / on_hold / dropped
	if item.Status == models.StatusWantTo || item.Status == models.StatusOnHold || item.Status == models.StatusDropped {
		s := models.StatusInProgress
		upd.Status = &s
		if item.StartedAt == nil {
			upd.StartedAt = &today
		}
		changed = true
	}

	// Rule 2: move to completed when all episodes are watched
	if item.TotalProgress != nil && *item.TotalProgress > 0 {
		watched, err := h.episodes.CountWatched(ctx, item.ID)
		if err == nil && watched >= *item.TotalProgress {
			s := models.StatusCompleted
			upd.Status = &s
			upd.CompletedAt = &today
			changed = true
		}
	}

	if changed {
		h.media.Update(ctx, item.ID, userID, upd) //nolint:errcheck
	}
}

// autoUpdateChapterStatus applies auto status rules after a chapter is logged.
func (h *MediaHandler) autoUpdateChapterStatus(ctx context.Context, item *models.MediaItem, chapterStatus models.ChapterStatus, userID int) {
	today := time.Now().Format("2006-01-02")
	upd := repository.UpdateMediaInput{}
	changed := false

	// Rule 1: move to in_progress on any chapter activity from want_to / on_hold / dropped
	if item.Status == models.StatusWantTo || item.Status == models.StatusOnHold || item.Status == models.StatusDropped {
		s := models.StatusInProgress
		upd.Status = &s
		if item.StartedAt == nil {
			upd.StartedAt = &today
		}
		changed = true
	}

	// Rule 2: move to completed when all chapters are done
	if chapterStatus == models.ChapterCompleted && item.TotalProgress != nil && *item.TotalProgress > 0 {
		counts, err := h.chapters.CountByStatus(ctx, item.ID)
		if err == nil {
			if counts[models.ChapterCompleted] >= *item.TotalProgress {
				s := models.StatusCompleted
				upd.Status = &s
				upd.CompletedAt = &today
				changed = true
			}
		}
	}

	if changed {
		h.media.Update(ctx, item.ID, userID, upd) //nolint:errcheck
	}
}
