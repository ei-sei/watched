package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ei-sei/brsti/internal/config"
	"github.com/go-chi/chi/v5"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.TrimSpace(s)
	return s
}

type TrendingItem struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Poster      string  `json:"poster,omitempty"`
	Score       float64 `json:"score,omitempty"`
	Year        int     `json:"year,omitempty"`
	ExternalID  string  `json:"external_id,omitempty"`
	MediaType   string  `json:"media_type,omitempty"`
	Synopsis    string  `json:"synopsis,omitempty"`
}

type TrendingSection struct {
	Label string         `json:"label"`
	Items []TrendingItem `json:"items"`
}

type trendingCacheEntry struct {
	sections  []TrendingSection
	expiresAt time.Time
}

type TrendingHandler struct {
	cfg    *config.Config
	client *http.Client
	mu     sync.Mutex
	cache  map[string]trendingCacheEntry
}

func NewTrendingHandler(cfg *config.Config) *TrendingHandler {
	return &TrendingHandler{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]trendingCacheEntry),
	}
}

var trendingTTL = map[string]time.Duration{
	"movies": 24 * time.Hour,
	"tv":     24 * time.Hour,
	"anime":  12 * time.Hour,
}

// GET /trending/{category}
func (h *TrendingHandler) Get(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	ttl, ok := trendingTTL[category]
	if !ok {
		jsonErr(w, http.StatusBadRequest, "invalid category")
		return
	}

	h.mu.Lock()
	if entry, hit := h.cache[category]; hit && time.Now().Before(entry.expiresAt) {
		sections := entry.sections
		h.mu.Unlock()
		jsonOK(w, sections)
		return
	}
	h.mu.Unlock()

	var (
		sections []TrendingSection
		err      error
	)
	switch category {
	case "movies":
		sections, err = h.fetchTMDB(r.Context(), "movie")
	case "tv":
		sections, err = h.fetchTMDB(r.Context(), "tv")
	case "anime":
		sections, err = h.fetchAniList(r.Context())
	}
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "failed to fetch trending data")
		return
	}

	h.mu.Lock()
	h.cache[category] = trendingCacheEntry{sections: sections, expiresAt: time.Now().Add(ttl)}
	h.mu.Unlock()

	jsonOK(w, sections)
}

func (h *TrendingHandler) fetchTMDB(ctx context.Context, mediaType string) ([]TrendingSection, error) {
	if h.cfg.TMDBKey == "" {
		return nil, fmt.Errorf("TMDB key not configured")
	}

	u := fmt.Sprintf(
		"https://api.themoviedb.org/3/trending/%s/week?api_key=%s&language=en-US",
		mediaType, h.cfg.TMDBKey,
	)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Results []struct {
			ID           int     `json:"id"`
			Title        string  `json:"title"`
			Name         string  `json:"name"`
			PosterPath   *string `json:"poster_path"`
			VoteAverage  float64 `json:"vote_average"`
			ReleaseDate  string  `json:"release_date"`
			FirstAirDate string  `json:"first_air_date"`
			Overview     string  `json:"overview"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	appType := "film"
	if mediaType == "tv" {
		appType = "tv_show"
	}

	items := make([]TrendingItem, 0, len(raw.Results))
	for _, r := range raw.Results {
		title := r.Title
		date := r.ReleaseDate
		if mediaType == "tv" {
			title = r.Name
			date = r.FirstAirDate
		}
		var poster string
		if r.PosterPath != nil {
			poster = "https://image.tmdb.org/t/p/w342" + *r.PosterPath
		}
		year := 0
		if len(date) >= 4 {
			fmt.Sscanf(date[:4], "%d", &year)
		}
		items = append(items, TrendingItem{
			ID:         r.ID,
			Title:      title,
			Poster:     poster,
			Score:      r.VoteAverage,
			Year:       year,
			ExternalID: fmt.Sprintf("tmdb:%d", r.ID),
			MediaType:  appType,
			Synopsis:   stripHTML(r.Overview),
		})
	}

	return []TrendingSection{{Label: "Trending This Week", Items: items}}, nil
}

// ── AniList ───────────────────────────────────────────────────────────────────

type aniListMedia struct {
	ID          int    `json:"id"`
	IDMal       int    `json:"idMal"`
	Description string `json:"description"`
	Title       struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
	} `json:"title"`
	CoverImage struct {
		ExtraLarge string `json:"extraLarge"`
		Large      string `json:"large"`
	} `json:"coverImage"`
	AverageScore int `json:"averageScore"`
	StartDate    struct {
		Year int `json:"year"`
	} `json:"startDate"`
}

func (h *TrendingHandler) fetchAniList(ctx context.Context) ([]TrendingSection, error) {
	curSeason, curYear := currentAniListSeason()
	nxtSeason, nxtYear := advanceAniListSeason(curSeason, curYear)

	query := fmt.Sprintf(`{
		trending: Page(page:1, perPage:20) {
			media(sort:TRENDING_DESC, type:ANIME, isAdult:false) {
				id idMal description(asHtml:false) title{romaji english} coverImage{extraLarge large} averageScore startDate{year}
			}
		}
		seasonal: Page(page:1, perPage:20) {
			media(sort:POPULARITY_DESC, type:ANIME, season:%s, seasonYear:%d, isAdult:false) {
				id idMal description(asHtml:false) title{romaji english} coverImage{extraLarge large} averageScore startDate{year}
			}
		}
		upcoming: Page(page:1, perPage:20) {
			media(sort:POPULARITY_DESC, type:ANIME, season:%s, seasonYear:%d, isAdult:false) {
				id idMal description(asHtml:false) title{romaji english} coverImage{extraLarge large} averageScore startDate{year}
			}
		}
	}`, curSeason, curYear, nxtSeason, nxtYear)

	payload, _ := json.Marshal(map[string]string{"query": query})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Trending struct{ Media []aniListMedia `json:"media"` } `json:"trending"`
			Seasonal struct{ Media []aniListMedia `json:"media"` } `json:"seasonal"`
			Upcoming struct{ Media []aniListMedia `json:"media"` } `json:"upcoming"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return []TrendingSection{
		{
			Label: "Trending Now",
			Items: mapAniListItems(raw.Data.Trending.Media),
		},
		{
			Label: fmt.Sprintf("Popular This Season (%s %d)", aniListSeasonName(curSeason), curYear),
			Items: mapAniListItems(raw.Data.Seasonal.Media),
		},
		{
			Label: fmt.Sprintf("Upcoming (%s %d)", aniListSeasonName(nxtSeason), nxtYear),
			Items: mapAniListItems(raw.Data.Upcoming.Media),
		},
	}, nil
}

func mapAniListItems(media []aniListMedia) []TrendingItem {
	items := make([]TrendingItem, 0, len(media))
	for _, m := range media {
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		poster := m.CoverImage.ExtraLarge
		if poster == "" {
			poster = m.CoverImage.Large
		}
		extID := fmt.Sprintf("anilist:%d", m.ID)
		if m.IDMal != 0 {
			extID = fmt.Sprintf("mal:%d", m.IDMal)
		}
		items = append(items, TrendingItem{
			ID:         m.ID,
			Title:      title,
			Poster:     poster,
			Score:      float64(m.AverageScore) / 10.0,
			Year:       m.StartDate.Year,
			ExternalID: extID,
			MediaType:  "anime",
			Synopsis:   stripHTML(m.Description),
		})
	}
	return items
}

func currentAniListSeason() (season string, year int) {
	now := time.Now()
	year = now.Year()
	switch {
	case now.Month() <= 3:
		return "WINTER", year
	case now.Month() <= 6:
		return "SPRING", year
	case now.Month() <= 9:
		return "SUMMER", year
	default:
		return "FALL", year
	}
}

func advanceAniListSeason(season string, year int) (string, int) {
	switch season {
	case "WINTER":
		return "SPRING", year
	case "SPRING":
		return "SUMMER", year
	case "SUMMER":
		return "FALL", year
	default: // FALL
		return "WINTER", year + 1
	}
}

func aniListSeasonName(s string) string {
	switch s {
	case "WINTER":
		return "Winter"
	case "SPRING":
		return "Spring"
	case "SUMMER":
		return "Summer"
	default:
		return "Fall"
	}
}
