package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		fetch(h.searchJikan)
	}

	wg.Wait()
	close(results)

	var all []models.SearchResult
	for res := range results {
		if res.err == nil {
			all = append(all, res.items...)
		}
	}
	if all == nil {
		all = []models.SearchResult{}
	}
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
		})
	}
	return out, nil
}

func (h *SearchHandler) searchOpenLibrary(ctx context.Context, q string) ([]models.SearchResult, error) {
	u := fmt.Sprintf("https://openlibrary.org/search.json?q=%s&limit=10&fields=key,title,author_name,first_publish_year,cover_i",
		url.QueryEscape(q))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Docs []struct {
			Key              string   `json:"key"`
			Title            string   `json:"title"`
			AuthorName       []string `json:"author_name"`
			FirstPublishYear *int     `json:"first_publish_year"`
			CoverI           *int     `json:"cover_i"`
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

	var raw struct {
		Items []struct {
			ID         string `json:"id"`
			VolumeInfo struct {
				Title               string   `json:"title"`
				Authors             []string `json:"authors"`
				PublishedDate       string   `json:"publishedDate"`
				Description         string   `json:"description"`
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
			s := vi.ImageLinks.Thumbnail
			poster = &s
		}
		desc := vi.Description
		extra := map[string]any{"authors": vi.Authors}
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
		out = append(out, models.SearchResult{
			Source:      "jikan",
			MediaType:   string(models.MediaTypeAnime),
			ExternalID:  fmt.Sprintf("mal:%d", a.MalID),
			Title:       a.Title,
			Year:        a.Year,
			PosterURL:   poster,
			Description: &desc,
			Extra:       extra,
		})
	}
	return out, nil
}
