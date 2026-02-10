package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// fetchAiredCount returns the number of episodes aired so far using the AniList
// GraphQL API (free, no auth). A single request is enough: for airing anime,
// nextAiringEpisode.episode is the NEXT episode, so latest aired = that - 1.
// AniList accepts MAL IDs via the idMal field.
func (h *MediaHandler) fetchAiredCount(ctx context.Context, malID string) int {
	body := fmt.Sprintf(
		`{"query":"query($id:Int){Media(idMal:$id,type:ANIME){episodes nextAiringEpisode{episode}}}","variables":{"id":%s}}`,
		malID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(body))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return 0
	}
	defer resp.Body.Close()
	var result struct {
		Data struct {
			Media struct {
				Episodes          *int `json:"episodes"`
				NextAiringEpisode *struct {
					Episode int `json:"episode"`
				} `json:"nextAiringEpisode"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}
	m := result.Data.Media
	if m.NextAiringEpisode != nil {
		return m.NextAiringEpisode.Episode - 1
	}
	if m.Episodes != nil {
		return *m.Episodes
	}
	return 0
}

// fetchAnimeData tries AniList first, falls back to Jikan.
func (h *MediaHandler) fetchAnimeData(ctx context.Context, malID string) *tmdbTVData {
	if data := h.fetchAniListData(ctx, malID); data != nil {
		return data
	}
	return h.fetchJikanData(ctx, malID)
}

func (h *MediaHandler) fetchAniListData(ctx context.Context, malID string) *tmdbTVData {
	body := fmt.Sprintf(`{"query":"query($id:Int){Media(idMal:$id,type:ANIME){episodes}}","variables":{"id":%s}}`, malID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var raw struct {
		Data struct {
			Media *struct {
				Episodes *int `json:"episodes"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || raw.Data.Media == nil {
		return nil
	}
	episodeCount := 0
	if raw.Data.Media.Episodes != nil && *raw.Data.Media.Episodes > 0 {
		episodeCount = *raw.Data.Media.Episodes
	}
	return &tmdbTVData{
		TotalEpisodes: raw.Data.Media.Episodes,
		Seasons:       []map[string]any{{"season_number": 1, "episode_count": episodeCount}},
	}
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
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil
	}
	// episodes is null for ongoing long-running anime (e.g. One Piece, Detective Conan)
	episodeCount := 0
	if raw.Data.Episodes != nil && *raw.Data.Episodes > 0 {
		episodeCount = *raw.Data.Episodes
	}
	seasons := []map[string]any{
		{"season_number": 1, "episode_count": episodeCount},
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

// parseMalID extracts the numeric ID from a "mal:12345" external ID.
// Returns ("", false) if the value is not a MAL id.
func parseMalID(externalID string) (string, bool) {
	id := strings.TrimPrefix(externalID, "mal:")
	return id, id != externalID
}

// parseTMDBID extracts the numeric ID from a "tmdb:12345" external ID.
// Returns ("", false) if the value is not a TMDB id.
func parseTMDBID(externalID string) (string, bool) {
	id := strings.TrimPrefix(externalID, "tmdb:")
	return id, id != externalID
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
			if tmdbID, ok := parseTMDBID(*body.ExternalID); ok {
				externalData = h.fetchTMDBData(r.Context(), tmdbID)
			}
		} else if body.MediaType == models.MediaTypeAnime {
			if malID, ok := parseMalID(*body.ExternalID); ok {
				externalData = h.fetchAnimeData(r.Context(), malID)
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
		tmdbID, ok := parseTMDBID(*item.ExternalID)
		if !ok {
			jsonErr(w, http.StatusBadRequest, "not a TMDB item")
			return
		}
		tvData = h.fetchTMDBData(r.Context(), tmdbID)
	} else if item.MediaType == models.MediaTypeAnime {
		malID, ok := parseMalID(*item.ExternalID)
		if !ok {
			jsonErr(w, http.StatusBadRequest, "not a MAL item")
			return
		}
		tvData = h.fetchAnimeData(r.Context(), malID)
	} else {
		jsonErr(w, http.StatusBadRequest, "only tv shows and anime can be refreshed")
		return
	}
	if tvData == nil {
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
	// For ongoing anime (TotalEpisodes == nil), fetch the exact aired count and
	// store it in total_progress only. episode_count in the season metadata stays
	// at 0 so the UI keeps treating this as ongoing (∞ display, + always enabled).
	if item.MediaType == models.MediaTypeAnime && tvData.TotalEpisodes == nil {
		if malID, ok := parseMalID(*item.ExternalID); ok {
			if n := h.fetchAiredCount(r.Context(), malID); n > 0 {
				tvData.TotalEpisodes = &n
			}
		}
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

// PATCH /media/{id}
func (h *MediaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Status          *models.MediaStatus `json:"status"           validate:"omitempty,oneof=want_to in_progress completed dropped on_hold"`
		Rating          *float64            `json:"rating"           validate:"omitempty,min=1,max=10"`
		ReviewText      *string             `json:"review_text"      validate:"omitempty,max=5000"`
		StartedAt       *string             `json:"started_at"`
		CompletedAt     *string             `json:"completed_at"`
		CurrentProgress *int                `json:"current_progress" validate:"omitempty,min=0"`
		TotalProgress   *int                `json:"total_progress"   validate:"omitempty,min=0"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		jsonErr(w, http.StatusUnprocessableEntity, err.Error())
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

// DELETE /media?type=film|tv_show|book|anime — delete all items of a type for the current user
func (h *MediaHandler) ClearByType(w http.ResponseWriter, r *http.Request) {
	mt := models.MediaType(r.URL.Query().Get("type"))
	switch mt {
	case models.MediaTypeFilm, models.MediaTypeTVShow, models.MediaTypeBook, models.MediaTypeAnime:
	default:
		jsonErr(w, http.StatusBadRequest, "type must be film, tv_show, book, or anime")
		return
	}
	deleted, err := h.media.DeleteAllByType(r.Context(), userIDFrom(r), mt)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to clear list")
		return
	}
	jsonOK(w, map[string]int64{"deleted": deleted})
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

// PUT /media/{id}/episodes/progress
func (h *MediaHandler) SetSeasonProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Season int `json:"season" validate:"required,min=1"`
		Count  int `json:"count"  validate:"min=0"`
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

	if err := h.episodes.SetSeasonProgress(r.Context(), id, body.Season, body.Count); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.autoUpdateEpisodeStatus(r.Context(), item, userIDFrom(r))

	w.WriteHeader(http.StatusNoContent)
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
		if _, err := h.media.Update(ctx, item.ID, userID, upd); err != nil {
			log.Printf("autoUpdateEpisodeStatus: update item %d: %v", item.ID, err)
		}
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
		if _, err := h.media.Update(ctx, item.ID, userID, upd); err != nil {
			log.Printf("autoUpdateChapterStatus: update item %d: %v", item.ID, err)
		}
	}
}
