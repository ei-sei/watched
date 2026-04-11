package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ei-sei/brsti/internal/config"
)

type HealthHandler struct {
	cfg    *config.Config
	client *http.Client
}

func NewHealthHandler(cfg *config.Config) *HealthHandler {
	return &HealthHandler{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type ServiceStatus struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

func (h *HealthHandler) pingService(name, url string) ServiceStatus {
	start := time.Now()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return ServiceStatus{Name: name, OK: false, Error: "bad url"}
	}
	resp, err := h.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ServiceStatus{Name: name, OK: false, LatencyMS: latency, Error: "unreachable"}
	}
	resp.Body.Close()
	ok := resp.StatusCode < 500
	s := ServiceStatus{Name: name, OK: ok, LatencyMS: latency}
	if !ok {
		s.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return s
}

// GET /admin/health
func (h *HealthHandler) ExternalServices(w http.ResponseWriter, r *http.Request) {
	type check struct {
		name string
		url  string
		skip bool
	}

	checks := []check{
		{
			name: "TMDB",
			url:  fmt.Sprintf("https://api.themoviedb.org/3/configuration?api_key=%s", h.cfg.TMDBKey),
			skip: h.cfg.TMDBKey == "",
		},
		{
			name: "Jikan",
			url:  "https://api.jikan.moe/v4/anime/1",
			skip: false,
		},
		{
			name: "Google Books",
			url:  fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/zyTCAlFPjgYC?key=%s", h.cfg.GoogleBooksKey),
			skip: h.cfg.GoogleBooksKey == "",
		},
		{
			name: "OpenLibrary",
			url:  "https://openlibrary.org/api/books?bibkeys=ISBN:9780140328721&format=json",
			skip: false,
		},
		{
			name: "AniList",
			url:  "https://graphql.anilist.co",
			skip: false,
		},
	}

	results := make([]ServiceStatus, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		if c.skip {
			results[i] = ServiceStatus{Name: c.name, OK: false, Error: "not configured"}
			continue
		}
		wg.Add(1)
		go func(i int, c check) {
			defer wg.Done()
			results[i] = h.pingService(c.name, c.url)
		}(i, c)
	}
	wg.Wait()

	jsonOK(w, results)
}
