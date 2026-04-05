package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/models"
	"github.com/ei-sei/brsti/internal/repository"
)

type ImportHandler struct {
	media    *repository.MediaRepo
	episodes *repository.EpisodeRepo
	cfg      *config.Config
	client   *http.Client
	enrichMu sync.Mutex // guards against concurrent enrichAllMeta runs
}

func NewImportHandler(media *repository.MediaRepo, episodes *repository.EpisodeRepo, cfg *config.Config) *ImportHandler {
	return &ImportHandler{media: media, episodes: episodes, cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}}
}

// ── Shared result type ─────────────────────────────────────────────────────────

type importResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ── MAL XML import ─────────────────────────────────────────────────────────────

var malStatusMap = map[string]models.MediaStatus{
	"Watching":      models.StatusInProgress,
	"watching":      models.StatusInProgress,
	"Completed":     models.StatusCompleted,
	"completed":     models.StatusCompleted,
	"On-Hold":       models.StatusOnHold,
	"on_hold":       models.StatusOnHold,
	"Dropped":       models.StatusDropped,
	"dropped":       models.StatusDropped,
	"Plan to Watch": models.StatusWantTo,
	"plan_to_watch": models.StatusWantTo,
}

type malXMLList struct {
	Anime []malXMLAnime `xml:"anime"`
}

type malXMLAnime struct {
	ID       int     `xml:"series_animedb_id"`
	Title    string  `xml:"series_title"`
	Image    string  `xml:"series_image"`
	Episodes int     `xml:"series_episodes"`
	Score    float64 `xml:"my_score"`
	Status   string  `xml:"my_status"`
	Start    string  `xml:"my_start_date"`
	Finish   string  `xml:"my_finish_date"`
	Watched  int     `xml:"my_watched_episodes"`
}

// POST /import/mal  (multipart, field: "file")
func (h *ImportHandler) ImportMAL(w http.ResponseWriter, r *http.Request) {
	data, err := readUpload(r)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	var list malXMLList
	if err := xml.Unmarshal(data, &list); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid MAL XML file")
		return
	}

	userID := auth.ClaimsFrom(r.Context()).UserID
	result, toEnrich := h.upsertAnimeList(r.Context(), userID, list.Anime)
	jsonOK(w, result)

	if len(toEnrich) > 0 {
		go func() {
			h.enrichTitles(context.Background(), toEnrich)
			h.enrichPosters(context.Background(), toEnrich)
		}()
	}
}

func (h *ImportHandler) upsertAnimeList(ctx context.Context, userID int, items []malXMLAnime) (importResult, []enrichItem) {
	result := importResult{Errors: []string{}}
	var toEnrich []enrichItem

	for _, a := range items {
		status, ok := malStatusMap[a.Status]
		if !ok {
			status = models.StatusWantTo
		}

		externalID := fmt.Sprintf("mal:%d", a.ID)
		var rating *float64
		if a.Score > 0 {
			r := a.Score
			rating = &r
		}

		existing, _ := h.media.GetByExternalID(ctx, userID, externalID)
		if existing != nil {
			result.Skipped++
			continue
		}

		var poster *string
		if a.Image != "" {
			poster = &a.Image
		}

		var currentProgress, totalProgress *int
		if a.Watched > 0 {
			currentProgress = &a.Watched
		}
		if a.Episodes > 0 {
			totalProgress = &a.Episodes
		}

		startedAt := malDate(a.Start)
		completedAt := malDate(a.Finish)

		// Build the same seasons structure that AddAnimeSeason produces so
		// TVSeasonProgress and AnimeSeasonManager behave identically.
		// episode_count is 0 for ongoing anime (MAL exports series_episodes=0 when unknown)
		metadata := map[string]any{
			"mal_id":   a.ID,
			"episodes": a.Episodes,
			"seasons": []map[string]any{
				{
					"season_number": 1,
					"episode_count": a.Episodes,
					"mal_id":        a.ID,
				},
			},
		}

		in := repository.CreateMediaInput{
			UserID:          userID,
			MediaType:       models.MediaTypeAnime,
			ExternalID:      &externalID,
			Title:           a.Title,
			PosterURL:       poster,
			Status:          status,
			Metadata:        metadata,
			CurrentProgress: currentProgress,
			TotalProgress:   totalProgress,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		}

		created, err := h.media.Create(ctx, in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", a.Title, err))
			continue
		}

		if rating != nil && created != nil {
			if _, err := h.media.Update(ctx, created.ID, userID, repository.UpdateMediaInput{Rating: rating}); err != nil {
				log.Printf("import anime: set rating for item %d: %v", created.ID, err)
			}
		}

		if created != nil {
			// Bulk-insert episode logs so the episode tracker matches progress.
			// Completed: all episodes watched_at = completed_at.
			// In-progress: episodes 1..watched, watched_at = started_at.
			switch status {
			case models.StatusCompleted:
				if a.Episodes > 0 {
					if err := h.episodes.BulkInsertWatched(ctx, created.ID, 1, a.Episodes, completedAt); err != nil {
						log.Printf("import anime: bulk insert episodes for item %d: %v", created.ID, err)
					}
				}
			case models.StatusInProgress:
				if a.Watched > 0 {
					if err := h.episodes.BulkInsertWatched(ctx, created.ID, 1, a.Watched, startedAt); err != nil {
						log.Printf("import anime: bulk insert episodes for item %d: %v", created.ID, err)
					}
				}
			}

			toEnrich = append(toEnrich, enrichItem{id: created.ID, malID: a.ID, userID: userID})
		}

		result.Imported++
	}

	return result, toEnrich
}

// malDate parses a MAL date string (YYYY-MM-DD). Returns nil if unset or zero.
func malDate(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" || s == "0000-00-00" {
		return nil
	}
	return &s
}

// ── Jikan poster enrichment ────────────────────────────────────────────────────

type enrichItem struct {
	id     int
	malID  int
	userID int
}

// anilistPoster fetches a cover image URL from AniList using the MAL ID.
// AniList has a higher rate limit (90 req/min) and is more reliable than Jikan for poster bulk-fetches.
func (h *ImportHandler) anilistPoster(ctx context.Context, malID int) *string {
	body := fmt.Sprintf(
		`{"query":"query($id:Int){Media(idMal:$id,type:ANIME){coverImage{large}}}","variables":{"id":%d}}`,
		malID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Data struct {
			Media struct {
				CoverImage struct {
					Large *string `json:"large"`
				} `json:"coverImage"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Data.Media.CoverImage.Large
}

// jikanPoster fetches a cover image URL from Jikan, retrying once on 429.
func (h *ImportHandler) jikanPoster(ctx context.Context, malID int) *string {
	url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d", malID)
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil
		}
		resp, err := h.client.Do(req)
		if err != nil {
			return nil
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil
		}
		defer resp.Body.Close()
		var raw struct {
			Data struct {
				Images struct {
					JPG struct {
						LargeImageURL *string `json:"large_image_url"`
						ImageURL      *string `json:"image_url"`
					} `json:"jpg"`
				} `json:"images"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return nil
		}
		if raw.Data.Images.JPG.LargeImageURL != nil {
			return raw.Data.Images.JPG.LargeImageURL
		}
		return raw.Data.Images.JPG.ImageURL
	}
	return nil
}

// GET /import/posters/missing-count
func (h *ImportHandler) MissingPosterCount(w http.ResponseWriter, r *http.Request) {
	userID := auth.ClaimsFrom(r.Context()).UserID
	missing, err := h.media.GetAnimeMissingPosters(r.Context(), userID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to query missing posters")
		return
	}
	jsonOK(w, map[string]int{"count": len(missing)})
}

// POST /import/posters/refetch
// Finds all of the user's anime with a mal_id but no poster, responds
// immediately with the count, then fetches posters in the background.
func (h *ImportHandler) RefetchPosters(w http.ResponseWriter, r *http.Request) {
	userID := auth.ClaimsFrom(r.Context()).UserID

	missing, err := h.media.GetAnimeMissingPosters(r.Context(), userID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to query missing posters")
		return
	}

	items := make([]enrichItem, len(missing))
	for i, m := range missing {
		items[i] = enrichItem{id: m.ID, malID: m.MalID}
	}

	jsonOK(w, map[string]int{"queued": len(items)})

	if len(items) > 0 {
		go h.enrichPosters(context.Background(), items)
	}
}

// applyPosters iterates over a set of item IDs, calls fetchFn to obtain a
// poster URL for each, saves it, and sleeps between requests. Centralises the
// repeated fetch→save→sleep pattern shared by the three poster enrichers.
func (h *ImportHandler) applyPosters(ctx context.Context, ids []int, fetchFn func(i int) *string, sleep time.Duration) {
	for i, id := range ids {
		if poster := fetchFn(i); poster != nil {
			if err := h.media.SetPoster(ctx, id, *poster); err != nil {
				log.Printf("applyPosters item %d: %v", id, err)
			}
		}
		time.Sleep(sleep)
	}
}

// enrichPosters runs in a background goroutine after the response is sent.
// Tries AniList first (higher rate limit), falls back to Jikan.
func (h *ImportHandler) enrichPosters(ctx context.Context, items []enrichItem) {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.id
	}
	h.applyPosters(ctx, ids, func(i int) *string {
		if p := h.anilistPoster(ctx, items[i].malID); p != nil {
			return p
		}
		return h.jikanPoster(ctx, items[i].malID)
	}, 700*time.Millisecond)
}

// enrichTitles batch-fetches English titles from AniList (50 per request) and
// updates each item's title + stores title_romaji in metadata.
func (h *ImportHandler) enrichTitles(ctx context.Context, items []enrichItem) {
	if len(items) == 0 {
		return
	}
	userID := items[0].userID

	// Build a map from malID → db item ID for fast lookup
	byMalID := make(map[int]int, len(items))
	for _, it := range items {
		byMalID[it.malID] = it.id
	}

	// Collect all MAL IDs
	malIDs := make([]int, 0, len(items))
	for _, it := range items {
		malIDs = append(malIDs, it.malID)
	}

	const batchSize = 50
	for i := 0; i < len(malIDs); i += batchSize {
		batch := malIDs[i:min(i+batchSize, len(malIDs))]

		payload, err := json.Marshal(map[string]any{
			"query":     `query($ids:[Int]){Page(perPage:50){media(idMal_in:$ids,type:ANIME){idMal title{romaji english}}}}`,
			"variables": map[string]any{"ids": batch},
		})
		if err != nil {
			log.Printf("enrichTitles: marshal batch payload: %v", err)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(string(payload)))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := h.client.Do(req)
		if err != nil {
			continue
		}

		var raw struct {
			Data struct {
				Page struct {
					Media []struct {
						IdMal *int `json:"idMal"`
						Title struct {
							Romaji  string `json:"romaji"`
							English string `json:"english"`
						} `json:"title"`
					} `json:"media"`
				} `json:"Page"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, m := range raw.Data.Page.Media {
			if m.IdMal == nil {
				continue
			}
			dbID, ok := byMalID[*m.IdMal]
			if !ok {
				continue
			}
			english := m.Title.English
			if english == "" {
				english = m.Title.Romaji
			}

			// Fetch current metadata to merge title_romaji in
			existing, err := h.media.GetByID(ctx, dbID, userID)
			if err != nil || existing == nil {
				continue
			}
			meta := existing.Metadata
			if meta == nil {
				meta = map[string]any{}
			}
			meta["title_romaji"] = m.Title.Romaji

			title := english
			if _, err := h.media.Update(ctx, dbID, userID, repository.UpdateMediaInput{
				Title:    &title,
				Metadata: meta,
			}); err != nil {
				log.Printf("enrichTitles: update item %d: %v", dbID, err)
			}
		}

		// Stay within AniList's 90 req/min rate limit
		time.Sleep(700 * time.Millisecond)
	}
}

// ── Film poster enrichment (TMDB) ─────────────────────────────────────────────

type filmEnrichItem struct {
	id    int
	title string
	year  *int
}

func (h *ImportHandler) enrichFilmPosters(ctx context.Context, items []filmEnrichItem) {
	if h.cfg.TMDBKey == "" {
		return
	}
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.id
	}
	h.applyPosters(ctx, ids, func(i int) *string {
		return h.tmdbPoster(ctx, items[i].title, items[i].year)
	}, 300*time.Millisecond)
}

func (h *ImportHandler) tmdbPoster(ctx context.Context, title string, year *int) *string {
	u := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s",
		h.cfg.TMDBKey, url.QueryEscape(title))
	if year != nil {
		u += fmt.Sprintf("&year=%d", *year)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var raw struct {
		Results []struct {
			PosterPath *string `json:"poster_path"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw.Results) == 0 || raw.Results[0].PosterPath == nil {
		return nil
	}
	s := "https://image.tmdb.org/t/p/w500" + *raw.Results[0].PosterPath
	return &s
}

// ── Book poster enrichment (Google Books) ─────────────────────────────────────

type bookEnrichItem struct {
	id    int
	title string
}

func (h *ImportHandler) enrichBookPosters(ctx context.Context, items []bookEnrichItem) {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.id
	}
	h.applyPosters(ctx, ids, func(i int) *string {
		return h.googleBooksPoster(ctx, items[i].title)
	}, 300*time.Millisecond)
}

func (h *ImportHandler) googleBooksPoster(ctx context.Context, title string) *string {
	apiURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=1", url.QueryEscape(title))
	if h.cfg.GoogleBooksKey != "" {
		apiURL += "&key=" + h.cfg.GoogleBooksKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var raw struct {
		Items []struct {
			VolumeInfo struct {
				ImageLinks *struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw.Items) == 0 || raw.Items[0].VolumeInfo.ImageLinks == nil {
		return nil
	}
	s := strings.Replace(raw.Items[0].VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
	return &s
}

// ── Metadata enrichment (all types) ───────────────────────────────────────────

// GET /import/metadata/missing-count
func (h *ImportHandler) MissingMetadataCount(w http.ResponseWriter, r *http.Request) {
	userID := auth.ClaimsFrom(r.Context()).UserID
	count, err := h.media.CountMissingMetadata(r.Context(), userID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to query missing metadata")
		return
	}
	jsonOK(w, map[string]int{"count": count})
}

// POST /import/metadata/refetch
// Queues a background enrichment pass for all items missing poster, year, or
// total_progress. Returns {queued: N} immediately.
func (h *ImportHandler) RefetchMetadata(w http.ResponseWriter, r *http.Request) {
	userID := auth.ClaimsFrom(r.Context()).UserID
	items, err := h.media.GetMissingMetadata(r.Context(), userID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to query missing metadata")
		return
	}
	jsonOK(w, map[string]int{"queued": len(items)})
	if len(items) > 0 {
		go func() {
			if !h.enrichMu.TryLock() {
				return // a run is already in progress
			}
			defer h.enrichMu.Unlock()
			h.enrichAllMeta(context.Background(), items)
		}()
	}
}

// enrichAllMeta runs in a background goroutine. Items are processed per type
// at their respective safe rate limits.
func (h *ImportHandler) enrichAllMeta(ctx context.Context, items []repository.MissingMetadataItem) {
	for _, item := range items {
		switch item.MediaType {
		case "anime":
			h.enrichAnimeMeta(ctx, item)
			time.Sleep(700 * time.Millisecond) // AniList: 90 req/min
		case "film":
			h.enrichFilmMeta(ctx, item)
			time.Sleep(300 * time.Millisecond)
		case "tv_show":
			h.enrichTVMeta(ctx, item)
			time.Sleep(300 * time.Millisecond)
		case "book":
			h.enrichBookMeta(ctx, item)
			time.Sleep(300 * time.Millisecond)
		}
	}
}

// enrichAnimeMeta fetches poster + episode count for an anime item via AniList/Jikan.
func (h *ImportHandler) enrichAnimeMeta(ctx context.Context, item repository.MissingMetadataItem) {
	malID := 0
	if item.Metadata != nil {
		switch v := item.Metadata["mal_id"].(type) {
		case float64:
			malID = int(v)
		case int:
			malID = v
		}
	}
	// Fallback: parse mal_id out of external_id ("mal:12345") when metadata blob
	// doesn't carry it (e.g. items added via search rather than MAL import).
	if malID == 0 && item.ExternalID != nil {
		if after, ok := strings.CutPrefix(*item.ExternalID, "mal:"); ok {
			fmt.Sscanf(after, "%d", &malID)
		}
	}
	if malID == 0 {
		return
	}

	in := repository.UpdateMediaInput{}

	// Poster
	if item.IsPosterMissing() {
		poster := h.anilistPoster(ctx, malID)
		if poster == nil {
			poster = h.jikanPoster(ctx, malID)
		}
		in.PosterURL = poster
	}

	// Episodes from AniList
	if item.TotalProgress == nil {
		episodes := h.anilistEpisodes(ctx, malID)
		if episodes != nil && *episodes > 0 {
			in.TotalProgress = episodes
		}
	}

	if in.PosterURL != nil || in.TotalProgress != nil {
		if _, err := h.media.Update(ctx, item.ID, item.UserID, in); err != nil {
			log.Printf("enrichAnimeMeta: update item %d: %v", item.ID, err)
		}
	}
}

// enrichFilmMeta fetches poster + year for a film via TMDB.
func (h *ImportHandler) enrichFilmMeta(ctx context.Context, item repository.MissingMetadataItem) {
	if h.cfg.TMDBKey == "" {
		return
	}
	var poster *string
	var year *int

	tmdbID := tmdbIntID(item.ExternalID)
	if tmdbID > 0 {
		poster, year = h.tmdbMovieByID(ctx, tmdbID)
	} else {
		poster = h.tmdbPoster(ctx, item.Title, item.Year)
	}

	in := repository.UpdateMediaInput{}
	if item.IsPosterMissing() {
		in.PosterURL = poster
	}
	if item.Year == nil && year != nil {
		in.Year = year
	}
	if in.PosterURL != nil || in.Year != nil {
		if _, err := h.media.Update(ctx, item.ID, item.UserID, in); err != nil {
			log.Printf("enrichFilmMeta: update item %d: %v", item.ID, err)
		}
	}
}

// enrichTVMeta fetches poster + year + episode count for a TV show via TMDB.
func (h *ImportHandler) enrichTVMeta(ctx context.Context, item repository.MissingMetadataItem) {
	if h.cfg.TMDBKey == "" {
		return
	}
	var poster *string
	var year *int
	var episodes *int

	tmdbID := tmdbIntID(item.ExternalID)
	if tmdbID > 0 {
		poster, year, episodes = h.tmdbTVByID(ctx, tmdbID)
	} else {
		poster, year, episodes = h.tmdbTVSearch(ctx, item.Title, item.Year)
	}

	in := repository.UpdateMediaInput{}
	if item.IsPosterMissing() {
		in.PosterURL = poster
	}
	if item.Year == nil && year != nil {
		in.Year = year
	}
	if item.TotalProgress == nil && episodes != nil && *episodes > 0 {
		in.TotalProgress = episodes
	}
	if in.PosterURL != nil || in.Year != nil || in.TotalProgress != nil {
		if _, err := h.media.Update(ctx, item.ID, item.UserID, in); err != nil {
			log.Printf("enrichTVMeta: update item %d: %v", item.ID, err)
		}
	}
}

// enrichBookMeta fetches poster + year + page count for a book via Google Books.
func (h *ImportHandler) enrichBookMeta(ctx context.Context, item repository.MissingMetadataItem) {
	var poster *string
	var year *int
	var pages *int

	gbID := googleBooksVolumeID(item.ExternalID)
	if gbID != "" {
		poster, year, pages = h.googleBooksByID(ctx, gbID)
	} else {
		poster, year, pages = h.googleBooksByTitle(ctx, item.Title)
	}

	in := repository.UpdateMediaInput{}
	if item.IsPosterMissing() {
		in.PosterURL = poster
	}
	if item.Year == nil && year != nil {
		in.Year = year
	}
	if item.TotalProgress == nil && pages != nil && *pages > 0 {
		in.TotalProgress = pages
	}
	if in.PosterURL != nil || in.Year != nil || in.TotalProgress != nil {
		if _, err := h.media.Update(ctx, item.ID, item.UserID, in); err != nil {
			log.Printf("enrichBookMeta: update item %d: %v", item.ID, err)
		}
	}
}

// ── Per-source fetch helpers ───────────────────────────────────────────────────

func (h *ImportHandler) anilistEpisodes(ctx context.Context, malID int) *int {
	body := fmt.Sprintf(
		`{"query":"query($id:Int){Media(idMal:$id,type:ANIME){episodes}}","variables":{"id":%d}}`,
		malID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Data struct {
			Media struct {
				Episodes *int `json:"episodes"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Data.Media.Episodes
}

func (h *ImportHandler) tmdbMovieByID(ctx context.Context, tmdbID int) (poster *string, year *int) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d?api_key=%s", tmdbID, h.cfg.TMDBKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()
	var raw struct {
		PosterPath  *string `json:"poster_path"`
		ReleaseDate string  `json:"release_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return
	}
	if raw.PosterPath != nil {
		s := "https://image.tmdb.org/t/p/w500" + *raw.PosterPath
		poster = &s
	}
	if len(raw.ReleaseDate) >= 4 {
		var y int
		fmt.Sscanf(raw.ReleaseDate[:4], "%d", &y)
		if y > 0 {
			year = &y
		}
	}
	return
}

func (h *ImportHandler) tmdbTVByID(ctx context.Context, tmdbID int) (poster *string, year *int, episodes *int) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/%d?api_key=%s", tmdbID, h.cfg.TMDBKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()
	var raw struct {
		PosterPath       *string `json:"poster_path"`
		FirstAirDate     string  `json:"first_air_date"`
		NumberOfEpisodes *int    `json:"number_of_episodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return
	}
	if raw.PosterPath != nil {
		s := "https://image.tmdb.org/t/p/w500" + *raw.PosterPath
		poster = &s
	}
	if len(raw.FirstAirDate) >= 4 {
		var y int
		fmt.Sscanf(raw.FirstAirDate[:4], "%d", &y)
		if y > 0 {
			year = &y
		}
	}
	episodes = raw.NumberOfEpisodes
	return
}

func (h *ImportHandler) tmdbTVSearch(ctx context.Context, title string, year *int) (poster *string, yearOut *int, episodes *int) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s",
		h.cfg.TMDBKey, url.QueryEscape(title))
	if year != nil {
		u += fmt.Sprintf("&first_air_date_year=%d", *year)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()
	var raw struct {
		Results []struct {
			ID         int     `json:"id"`
			PosterPath *string `json:"poster_path"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw.Results) == 0 {
		return
	}
	r := raw.Results[0]
	if r.PosterPath != nil {
		s := "https://image.tmdb.org/t/p/w500" + *r.PosterPath
		poster = &s
	}
	if len(r.FirstAirDate) >= 4 {
		var y int
		fmt.Sscanf(r.FirstAirDate[:4], "%d", &y)
		if y > 0 {
			yearOut = &y
		}
	}
	// Fetch episode count via detail endpoint
	_, _, episodes = h.tmdbTVByID(ctx, r.ID)
	time.Sleep(200 * time.Millisecond) // second TMDB call
	return
}

func (h *ImportHandler) googleBooksByID(ctx context.Context, gbID string) (poster *string, year *int, pages *int) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/%s", url.PathEscape(gbID))
	if h.cfg.GoogleBooksKey != "" {
		apiURL += "?key=" + h.cfg.GoogleBooksKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()
	var raw struct {
		VolumeInfo struct {
			PublishedDate string `json:"publishedDate"`
			PageCount     *int   `json:"pageCount"`
			ImageLinks    *struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"imageLinks"`
		} `json:"volumeInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return
	}
	vi := raw.VolumeInfo
	if vi.ImageLinks != nil {
		s := strings.Replace(vi.ImageLinks.Thumbnail, "http://", "https://", 1)
		poster = &s
	}
	if len(vi.PublishedDate) >= 4 {
		var y int
		fmt.Sscanf(vi.PublishedDate[:4], "%d", &y)
		if y > 0 {
			year = &y
		}
	}
	pages = vi.PageCount
	return
}

func (h *ImportHandler) googleBooksByTitle(ctx context.Context, title string) (poster *string, year *int, pages *int) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=1", url.QueryEscape(title))
	if h.cfg.GoogleBooksKey != "" {
		apiURL += "&key=" + h.cfg.GoogleBooksKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return
	}
	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()
	var raw struct {
		Items []struct {
			ID         string `json:"id"`
			VolumeInfo struct {
				PublishedDate string `json:"publishedDate"`
				PageCount     *int   `json:"pageCount"`
				ImageLinks    *struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw.Items) == 0 {
		return
	}
	vi := raw.Items[0].VolumeInfo
	if vi.ImageLinks != nil {
		s := strings.Replace(vi.ImageLinks.Thumbnail, "http://", "https://", 1)
		poster = &s
	}
	if len(vi.PublishedDate) >= 4 {
		var y int
		fmt.Sscanf(vi.PublishedDate[:4], "%d", &y)
		if y > 0 {
			year = &y
		}
	}
	pages = vi.PageCount
	return
}

// tmdbIntID extracts the integer ID from an external_id like "tmdb:12345".
func tmdbIntID(externalID *string) int {
	if externalID == nil {
		return 0
	}
	s := *externalID
	if !strings.HasPrefix(s, "tmdb:") {
		return 0
	}
	var id int
	fmt.Sscanf(strings.TrimPrefix(s, "tmdb:"), "%d", &id)
	return id
}

// googleBooksVolumeID extracts the volume ID from an external_id like "gb:AbCdEf".
func googleBooksVolumeID(externalID *string) string {
	if externalID == nil {
		return ""
	}
	s := *externalID
	if !strings.HasPrefix(s, "gb:") {
		return ""
	}
	return strings.TrimPrefix(s, "gb:")
}

// ── Letterboxd CSV import ──────────────────────────────────────────────────────

// POST /import/letterboxd  (multipart, field: "file")
// Accepts any Letterboxd CSV export (films.csv, diary.csv, watchlist.csv, ratings.csv).
func (h *ImportHandler) ImportLetterboxd(w http.ResponseWriter, r *http.Request) {
	data, err := readUpload(r)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(records) < 2 {
		jsonErr(w, http.StatusBadRequest, "invalid CSV file")
		return
	}

	// Build header index (case-insensitive)
	idx := headerIndex(records[0])
	nameColRaw, ok := idx["name"]
	if !ok {
		jsonErr(w, http.StatusBadRequest, "CSV must have a Name column")
		return
	}
	nameCol := nameColRaw.(int)

	yearCol   := idx["year"]
	ratingCol := idx["rating"]
	uriCol    := idx["letterboxd uri"]

	// If there's a Rating column we treat entries as watched films (completed).
	// Watchlist exports have no Rating column → want_to.
	_, hasRating := ratingCol.(int)
	defaultStatus := models.StatusWantTo
	if hasRating {
		defaultStatus = models.StatusCompleted
	}

	userID := auth.ClaimsFrom(r.Context()).UserID
	result := importResult{Errors: []string{}}
	var toEnrich []filmEnrichItem

	for _, row := range records[1:] {
		if len(row) <= nameCol {
			continue
		}
		title := strings.TrimSpace(row[nameCol])
		if title == "" {
			continue
		}

		var year *int
		if col, ok := yearCol.(int); ok && col < len(row) {
			if y, err := strconv.Atoi(strings.TrimSpace(row[col])); err == nil && y > 0 {
				year = &y
			}
		}

		var externalID *string
		if col, ok := uriCol.(int); ok && col < len(row) {
			slug := letterboxdSlug(row[col])
			if slug != "" {
				s := "lb:" + slug
				externalID = &s
			}
		}

		if externalID != nil {
			existing, _ := h.media.GetByExternalID(r.Context(), userID, *externalID)
			if existing != nil {
				result.Skipped++
				continue
			}
		}

		var rating *float64
		if col, ok := ratingCol.(int); ok && col < len(row) {
			if rv, err := strconv.ParseFloat(strings.TrimSpace(row[col]), 64); err == nil && rv > 0 {
				// Letterboxd uses 0.5–5.0; convert to 1–10
				v := rv * 2
				rating = &v
			}
		}

		in := repository.CreateMediaInput{
			UserID:     userID,
			MediaType:  models.MediaTypeFilm,
			ExternalID: externalID,
			Title:      title,
			Year:       year,
			Status:     defaultStatus,
		}

		created, err := h.media.Create(r.Context(), in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", title, err))
			continue
		}

		if rating != nil && created != nil {
			if _, err := h.media.Update(r.Context(), created.ID, userID, repository.UpdateMediaInput{Rating: rating}); err != nil {
				log.Printf("import: set rating for item %d: %v", created.ID, err)
			}
		}

		if created != nil {
			toEnrich = append(toEnrich, filmEnrichItem{id: created.ID, title: title, year: year})
		}

		result.Imported++
	}

	jsonOK(w, result)

	if len(toEnrich) > 0 {
		go h.enrichFilmPosters(context.Background(), toEnrich)
	}
}

// letterboxdSlug extracts the film slug from a Letterboxd URI.
// e.g. "https://letterboxd.com/film/oppenheimer-2023/" → "oppenheimer-2023"
func letterboxdSlug(uri string) string {
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// ── Goodreads CSV import ───────────────────────────────────────────────────────

var goodreadsShelfMap = map[string]models.MediaStatus{
	"read":              models.StatusCompleted,
	"currently-reading": models.StatusInProgress,
	"to-read":           models.StatusWantTo,
}

// POST /import/goodreads  (multipart, field: "file")
func (h *ImportHandler) ImportGoodreads(w http.ResponseWriter, r *http.Request) {
	data, err := readUpload(r)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(records) < 2 {
		jsonErr(w, http.StatusBadRequest, "invalid CSV file")
		return
	}

	idx := headerIndex(records[0])

	titleColRaw, ok := idx["title"]
	if !ok {
		jsonErr(w, http.StatusBadRequest, "CSV must have a Title column")
		return
	}
	titleCol := titleColRaw.(int)

	bookIDCol     := idx["book id"]
	authorCol     := idx["author l-f"]
	if _, ok := authorCol.(int); !ok {
		authorCol = idx["author"]
	}
	ratingCol     := idx["my rating"]
	pagesCol      := idx["number of pages"]
	yearCol       := idx["original publication year"]
	shelfCol      := idx["exclusive shelf"]

	userID := auth.ClaimsFrom(r.Context()).UserID
	result := importResult{Errors: []string{}}
	var toEnrich []bookEnrichItem

	for _, row := range records[1:] {
		if len(row) <= titleCol {
			continue
		}
		title := strings.TrimSpace(row[titleCol])
		if title == "" {
			continue
		}

		var externalID *string
		if col, ok := bookIDCol.(int); ok && col < len(row) {
			if id := strings.TrimSpace(row[col]); id != "" {
				s := "gr:" + id
				externalID = &s
			}
		}

		if externalID != nil {
			existing, _ := h.media.GetByExternalID(r.Context(), userID, *externalID)
			if existing != nil {
				result.Skipped++
				continue
			}
		}

		status := models.StatusWantTo
		if col, ok := shelfCol.(int); ok && col < len(row) {
			if s, mapped := goodreadsShelfMap[strings.TrimSpace(row[col])]; mapped {
				status = s
			}
		}

		var year *int
		if col, ok := yearCol.(int); ok && col < len(row) {
			if y, err := strconv.Atoi(strings.TrimSpace(row[col])); err == nil && y > 0 {
				year = &y
			}
		}

		var totalProgress *int
		if col, ok := pagesCol.(int); ok && col < len(row) {
			if p, err := strconv.Atoi(strings.TrimSpace(row[col])); err == nil && p > 0 {
				totalProgress = &p
			}
		}

		var authors []string
		if col, ok := authorCol.(int); ok && col < len(row) {
			// Goodreads stores as "Last, First" — reverse to "First Last"
			if a := strings.TrimSpace(row[col]); a != "" {
				authors = []string{reverseAuthorName(a)}
			}
		}

		extra := map[string]any{}
		if len(authors) > 0 {
			extra["authors"] = authors
		}

		var rating *float64
		if col, ok := ratingCol.(int); ok && col < len(row) {
			if rv, err := strconv.ParseFloat(strings.TrimSpace(row[col]), 64); err == nil && rv > 0 {
				// Goodreads uses 1–5; convert to 1–10
				v := rv * 2
				rating = &v
			}
		}

		in := repository.CreateMediaInput{
			UserID:        userID,
			MediaType:     models.MediaTypeBook,
			ExternalID:    externalID,
			Title:         title,
			Year:          year,
			Status:        status,
			TotalProgress: totalProgress,
			Metadata:      extra,
		}

		created, err := h.media.Create(r.Context(), in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", title, err))
			continue
		}

		if rating != nil && created != nil {
			if _, err := h.media.Update(r.Context(), created.ID, userID, repository.UpdateMediaInput{Rating: rating}); err != nil {
				log.Printf("import: set rating for item %d: %v", created.ID, err)
			}
		}

		if created != nil {
			toEnrich = append(toEnrich, bookEnrichItem{id: created.ID, title: title})
		}

		result.Imported++
	}

	jsonOK(w, result)

	if len(toEnrich) > 0 {
		go h.enrichBookPosters(context.Background(), toEnrich)
	}
}

// reverseAuthorName converts "Last, First" → "First Last".
func reverseAuthorName(name string) string {
	parts := strings.SplitN(name, ", ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]) + " " + strings.TrimSpace(parts[0])
	}
	return name
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// readUpload reads the "file" field from a multipart form (max 10 MB).
func readUpload(r *http.Request) ([]byte, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return nil, fmt.Errorf("file too large or invalid")
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("missing file field")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read file")
	}
	return data, nil
}

// headerIndex returns a map of lowercased column name → column index.
func headerIndex(headers []string) map[string]any {
	idx := make(map[string]any, len(headers))
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}
