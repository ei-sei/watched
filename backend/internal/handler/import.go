package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
			_, _ = h.media.Update(ctx, created.ID, userID, repository.UpdateMediaInput{Rating: rating})
		}

		if created != nil {
			// Bulk-insert episode logs so the episode tracker matches progress.
			// Completed: all episodes watched_at = completed_at.
			// In-progress: episodes 1..watched, watched_at = started_at.
			switch status {
			case models.StatusCompleted:
				if a.Episodes > 0 {
					_ = h.episodes.BulkInsertWatched(ctx, created.ID, 1, a.Episodes, completedAt)
				}
			case models.StatusInProgress:
				if a.Watched > 0 {
					_ = h.episodes.BulkInsertWatched(ctx, created.ID, 1, a.Watched, startedAt)
				}
			}

			if poster == nil {
				toEnrich = append(toEnrich, enrichItem{id: created.ID, malID: a.ID, userID: userID})
			}
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

// enrichPosters runs in a background goroutine after the response is sent.
// Tries AniList first (higher rate limit), falls back to Jikan.
func (h *ImportHandler) enrichPosters(ctx context.Context, items []enrichItem) {
	for _, item := range items {
		poster := h.anilistPoster(ctx, item.malID)
		if poster == nil {
			poster = h.jikanPoster(ctx, item.malID)
		}
		if poster != nil {
			_ = h.media.SetPoster(ctx, item.id, *poster)
		}
		time.Sleep(700 * time.Millisecond)
	}
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

		payload, _ := json.Marshal(map[string]any{
			"query":     `query($ids:[Int]){Page(perPage:50){media(idMal_in:$ids,type:ANIME){idMal title{romaji english}}}}`,
			"variables": map[string]any{"ids": batch},
		})
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
			_, _ = h.media.Update(ctx, dbID, userID, repository.UpdateMediaInput{
				Title:    &title,
				Metadata: meta,
			})
		}

		// Stay within AniList's 90 req/min rate limit
		time.Sleep(700 * time.Millisecond)
	}
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
			_, _ = h.media.Update(r.Context(), created.ID, userID, repository.UpdateMediaInput{Rating: rating})
		}

		result.Imported++
	}

	jsonOK(w, result)
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
			_, _ = h.media.Update(r.Context(), created.ID, userID, repository.UpdateMediaInput{Rating: rating})
		}

		result.Imported++
	}

	jsonOK(w, result)
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
