package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/models"
)

type SearchHandler struct {
	cfg    *config.Config
	client *http.Client
}

func NewSearchHandler(cfg *config.Config) *SearchHandler {
	return &SearchHandler{
		cfg:    cfg,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

// GET /search?q=...&type=film|tv_show|book
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		jsonErr(w, http.StatusBadRequest, "q is required")
		return
	}
	mediaType := r.URL.Query().Get("type") // optional filter

	type result struct {
		items []models.SearchResult
		err   error
	}

	var wg sync.WaitGroup
	results := make(chan result, 4)

	fetch := func(fn func(context.Context, string) ([]models.SearchResult, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := fn(r.Context(), q)
			results <- result{items, err}
		}()
	}

	if mediaType == "" || mediaType == string(models.MediaTypeFilm) || mediaType == string(models.MediaTypeTVShow) {
		fetch(h.searchTMDB)
	}
	if mediaType == "" || mediaType == string(models.MediaTypeBook) {
		fetch(h.searchOpenLibrary)
		fetch(h.searchGoogleBooks)
	}
	if mediaType == "" || mediaType == string(models.MediaTypeAnime) {
		fetch(h.searchAnime)
	}

	wg.Wait()
	close(results)

	var all []models.SearchResult
	for res := range results {
		if res.err != nil {
			continue
		}
		all = append(all, res.items...)
	}
	if all == nil {
		all = []models.SearchResult{}
	}

	// When doing an unfiltered search, TMDB TV results and Jikan anime results
	// overlap. Deduplicate by dropping TMDB TV entries whose title matches a
	// Jikan result — Jikan is the authoritative source for anime.
	if mediaType == "" {
		animeTitles := make(map[string]struct{})
		for _, r := range all {
			if r.Source == "jikan" {
				animeTitles[normTitle(r.Title)] = struct{}{}
			}
		}
		if len(animeTitles) > 0 {
			filtered := all[:0]
			for _, r := range all {
				if r.Source == "tmdb" && r.MediaType == string(models.MediaTypeTVShow) {
					if _, dup := animeTitles[normTitle(r.Title)]; dup {
						continue
					}
				}
				filtered = append(filtered, r)
			}
			all = filtered
		}
	}

	rankResults(all, q)
	jsonOK(w, all)
}

func (h *SearchHandler) searchTMDB(ctx context.Context, q string) ([]models.SearchResult, error) {
	if h.cfg.TMDBKey == "" {
		return nil, nil
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=%s&query=%s&language=en-US",
		h.cfg.TMDBKey, url.QueryEscape(q))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Results []struct {
			ID           int     `json:"id"`
			MediaType    string  `json:"media_type"`
			Title        string  `json:"title"`
			Name         string  `json:"name"`
			ReleaseDate  string  `json:"release_date"`
			FirstAirDate string  `json:"first_air_date"`
			PosterPath   *string `json:"poster_path"`
			Overview     string  `json:"overview"`
			Popularity   float64 `json:"popularity"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var out []models.SearchResult
	for _, r := range raw.Results {
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}

		mt := string(models.MediaTypeFilm)
		title := r.Title
		date := r.ReleaseDate
		if r.MediaType == "tv" {
			mt = string(models.MediaTypeTVShow)
			title = r.Name
			date = r.FirstAirDate
		}

		var year *int
		if len(date) >= 4 {
			var y int
			fmt.Sscanf(date[:4], "%d", &y)
			year = &y
		}

		var poster *string
		if r.PosterPath != nil {
			s := "https://image.tmdb.org/t/p/w500" + *r.PosterPath
			poster = &s
		}

		desc := r.Overview
		out = append(out, models.SearchResult{
			Source:      "tmdb",
			MediaType:   mt,
			ExternalID:  fmt.Sprintf("tmdb:%d", r.ID),
			Title:       title,
			Year:        year,
			PosterURL:   poster,
			Description: &desc,
			Popularity:  r.Popularity,
		})
	}
	return out, nil
}

func (h *SearchHandler) searchOpenLibrary(ctx context.Context, q string) ([]models.SearchResult, error) {
	u := fmt.Sprintf("https://openlibrary.org/search.json?q=%s&limit=10&fields=key,title,author_name,first_publish_year,cover_i,number_of_pages_median",
		url.QueryEscape(q))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Docs []struct {
			Key                  string   `json:"key"`
			Title                string   `json:"title"`
			AuthorName           []string `json:"author_name"`
			FirstPublishYear     *int     `json:"first_publish_year"`
			CoverI               *int     `json:"cover_i"`
			NumberOfPagesMedian  *int     `json:"number_of_pages_median"`
		} `json:"docs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var out []models.SearchResult
	for _, d := range raw.Docs {
		var poster *string
		if d.CoverI != nil {
			s := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", *d.CoverI)
			poster = &s
		}
		extra := map[string]any{"authors": d.AuthorName}
		if d.NumberOfPagesMedian != nil && *d.NumberOfPagesMedian > 0 {
			extra["page_count"] = *d.NumberOfPagesMedian
		}
		out = append(out, models.SearchResult{
			Source:     "openlibrary",
			MediaType:  string(models.MediaTypeBook),
			ExternalID: "ol:" + d.Key,
			Title:      d.Title,
			Year:       d.FirstPublishYear,
			PosterURL:  poster,
			Extra:      extra,
		})
	}
	return out, nil
}

func (h *SearchHandler) searchGoogleBooks(ctx context.Context, q string) ([]models.SearchResult, error) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=10", url.QueryEscape(q))
	if h.cfg.GoogleBooksKey != "" {
		apiURL += "&key=" + h.cfg.GoogleBooksKey
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books: status %d", resp.StatusCode)
	}

	var raw struct {
		Items []struct {
			ID         string `json:"id"`
			VolumeInfo struct {
				Title               string   `json:"title"`
				Authors             []string `json:"authors"`
				PublishedDate       string   `json:"publishedDate"`
				Description         string   `json:"description"`
				PageCount           *int     `json:"pageCount"`
				ImageLinks          *struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var out []models.SearchResult
	for _, item := range raw.Items {
		vi := item.VolumeInfo
		var year *int
		if len(vi.PublishedDate) >= 4 {
			var y int
			fmt.Sscanf(vi.PublishedDate[:4], "%d", &y)
			year = &y
		}
		var poster *string
		if vi.ImageLinks != nil {
			s := strings.Replace(vi.ImageLinks.Thumbnail, "http://", "https://", 1)
			poster = &s
		}
		desc := vi.Description
		extra := map[string]any{"authors": vi.Authors}
		if vi.PageCount != nil && *vi.PageCount > 0 {
			extra["page_count"] = *vi.PageCount
		}
		out = append(out, models.SearchResult{
			Source:      "googlebooks",
			MediaType:   string(models.MediaTypeBook),
			ExternalID:  "gb:" + item.ID,
			Title:       vi.Title,
			Year:        year,
			PosterURL:   poster,
			Description: &desc,
			Extra:       extra,
		})
	}
	return out, nil
}

func (h *SearchHandler) searchJikan(ctx context.Context, q string) ([]models.SearchResult, error) {
	u := fmt.Sprintf("https://api.jikan.moe/v4/anime?q=%s&limit=20&sfw=false", url.QueryEscape(q))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Data []struct {
			MalID  int    `json:"mal_id"`
			Title  string `json:"title"`
			Year   *int   `json:"year"`
			Aired  struct {
				From *string `json:"from"`
			} `json:"aired"`
			Images struct {
				JPG struct {
					LargeImageURL *string `json:"large_image_url"`
				} `json:"jpg"`
			} `json:"images"`
			Synopsis string   `json:"synopsis"`
			Score    *float64 `json:"score"`
			Episodes *int     `json:"episodes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var out []models.SearchResult
	for _, a := range raw.Data {
		poster := a.Images.JPG.LargeImageURL
		desc := a.Synopsis
		extra := map[string]any{
			"mal_id":   a.MalID,
			"score":    a.Score,
			"episodes": a.Episodes,
		}
		pop := 0.0
		if a.Score != nil {
			pop = *a.Score * 10 // scale 0–10 → 0–100 to roughly match TMDB popularity range
		}
		year := a.Year
		if year == nil && a.Aired.From != nil && len(*a.Aired.From) >= 4 {
			var y int
			fmt.Sscanf((*a.Aired.From)[:4], "%d", &y)
			if y > 0 {
				year = &y
			}
		}
		out = append(out, models.SearchResult{
			Source:      "jikan",
			MediaType:   string(models.MediaTypeAnime),
			ExternalID:  fmt.Sprintf("mal:%d", a.MalID),
			Title:       a.Title,
			Year:        year,
			PosterURL:   poster,
			Description: &desc,
			Extra:       extra,
			Popularity:  pop,
		})
	}
	return out, nil
}

// searchAnime tries AniList first, falls back to Jikan if AniList returns nothing.
func (h *SearchHandler) searchAnime(ctx context.Context, q string) ([]models.SearchResult, error) {
	results, err := h.searchAniList(ctx, q)
	if err != nil || len(results) == 0 {
		return h.searchJikan(ctx, q)
	}
	return results, nil
}

func (h *SearchHandler) searchAniList(ctx context.Context, q string) ([]models.SearchResult, error) {
	payload, _ := json.Marshal(map[string]any{
		"query": `query($s:String){Page(page:1,perPage:20){media(search:$s,type:ANIME,sort:SEARCH_MATCH){id idMal title{romaji english} episodes coverImage{large} averageScore description startDate{year}}}}`,
		"variables": map[string]string{"s": q},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Page struct {
				Media []struct {
					ID    int  `json:"id"`
					IdMal *int `json:"idMal"`
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
					} `json:"title"`
					Episodes   *int `json:"episodes"`
					CoverImage struct {
						Large string `json:"large"`
					} `json:"coverImage"`
					AverageScore *int   `json:"averageScore"`
					Description  string `json:"description"`
					StartDate    struct {
						Year *int `json:"year"`
					} `json:"startDate"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var out []models.SearchResult
	for _, m := range raw.Data.Page.Media {
		if m.IdMal == nil {
			continue // no MAL ID — skip to keep external_id format consistent
		}
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		var poster *string
		if m.CoverImage.Large != "" {
			s := m.CoverImage.Large
			poster = &s
		}
		pop := 0.0
		if m.AverageScore != nil {
			pop = float64(*m.AverageScore) // AniList scores are 0–100, same scale as TMDB popularity
		}
		desc := m.Description
		extra := map[string]any{
			"mal_id":   *m.IdMal,
			"episodes": m.Episodes,
			"score":    m.AverageScore,
		}
		out = append(out, models.SearchResult{
			Source:      "anilist",
			MediaType:   string(models.MediaTypeAnime),
			ExternalID:  fmt.Sprintf("mal:%d", *m.IdMal),
			Title:       title,
			Year:        m.StartDate.Year,
			PosterURL:   poster,
			Description: &desc,
			Extra:       extra,
			Popularity:  pop,
		})
	}
	return out, nil
}

// rankResults sorts results so that:
//  1. Exact title matches come first
//  2. Prefix matches come next
//  3. Higher popularity / score ranks higher within each tier
//  4. Results missing a poster are pushed to the bottom within their tier
func rankResults(results []models.SearchResult, q string) {
	qNorm := normTitle(q)
	score := func(r models.SearchResult) float64 {
		t := normTitle(r.Title)
		var s float64
		switch {
		case t == qNorm:
			s = 3000
		case strings.HasPrefix(t, qNorm):
			s = 2000
		case strings.Contains(t, qNorm):
			s = 1000
		}
		s += r.Popularity
		if r.PosterURL == nil || *r.PosterURL == "" {
			s -= 500
		}
		return s
	}

	// Insertion sort — result sets are small (< 60 items) so this is fine.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && score(results[j]) > score(results[j-1]); j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

// normTitle lowercases and strips common punctuation for loose title matching.
func normTitle(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
