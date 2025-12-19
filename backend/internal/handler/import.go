package handler

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/models"
	"github.com/ei-sei/brsti/internal/repository"
)

type ImportHandler struct {
	media  *repository.MediaRepo
	cfg    *config.Config
	client *http.Client
}

func NewImportHandler(media *repository.MediaRepo, cfg *config.Config) *ImportHandler {
	return &ImportHandler{media: media, cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}}
}

// MAL status → our status
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

// ── XML import ────────────────────────────────────────────────────────────────

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

// POST /import/mal/file  (multipart, field: "file")
func (h *ImportHandler) ImportXML(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(5 << 20); err != nil { // 5MB
		jsonErr(w, http.StatusBadRequest, "file too large or invalid")
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "could not read file")
		return
	}

	var list malXMLList
	if err := xml.Unmarshal(data, &list); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid MAL XML file")
		return
	}

	userID := auth.ClaimsFrom(r.Context()).UserID
	result := h.upsertAnimeList(r.Context(), userID, list.Anime)
	jsonOK(w, result)
}

// ── Username import (MAL API v2) ──────────────────────────────────────────────

type malAPIResponse struct {
	Data []struct {
		Node struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			MainPicture *struct {
				Large string `json:"large"`
			} `json:"main_picture"`
		} `json:"node"`
		ListStatus struct {
			Status          string  `json:"status"`
			Score           float64 `json:"score"`
			NumEpsWatched   int     `json:"num_episodes_watched"`
			StartDate       string  `json:"start_date"`
			FinishDate      string  `json:"finish_date"`
		} `json:"list_status"`
	} `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// POST /import/mal/username  body: {"username":"..."}
func (h *ImportHandler) ImportUsername(w http.ResponseWriter, r *http.Request) {
	if h.cfg.MALClientID == "" {
		jsonErr(w, http.StatusServiceUnavailable, "MAL_CLIENT_ID not configured")
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	if err := decode(r, &body); err != nil || body.Username == "" {
		jsonErr(w, http.StatusBadRequest, "username required")
		return
	}

	animes, err := h.fetchMALList(r.Context(), body.Username)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "could not fetch MAL list: "+err.Error())
		return
	}

	userID := auth.ClaimsFrom(r.Context()).UserID

	// Convert API format to XML format struct for shared upsert logic
	var items []malXMLAnime
	for _, entry := range animes.Data {
		score := entry.ListStatus.Score
		img := ""
		if entry.Node.MainPicture != nil {
			img = entry.Node.MainPicture.Large
		}
		items = append(items, malXMLAnime{
			ID:     entry.Node.ID,
			Title:  entry.Node.Title,
			Image:  img,
			Score:  score,
			Status: entry.ListStatus.Status,
			Start:  entry.ListStatus.StartDate,
			Finish: entry.ListStatus.FinishDate,
		})
	}

	result := h.upsertAnimeList(r.Context(), userID, items)
	jsonOK(w, result)
}

func (h *ImportHandler) fetchMALList(ctx context.Context, username string) (*malAPIResponse, error) {
	url := fmt.Sprintf(
		"https://api.myanimelist.net/v2/users/%s/animelist?fields=list_status&limit=1000&nsfw=true",
		username,
	)

	var combined malAPIResponse
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-MAL-CLIENT-ID", h.cfg.MALClientID)

		resp, err := h.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("MAL user not found")
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("MAL API error: %s", resp.Status)
		}

		var page malAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, err
		}
		combined.Data = append(combined.Data, page.Data...)
		url = page.Paging.Next
	}
	return &combined, nil
}

// ── Jikan poster enrichment ───────────────────────────────────────────────────

func (h *ImportHandler) jikanPoster(ctx context.Context, malID int) *string {
	url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d", malID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
			Images struct {
				JPG struct {
					LargeImageURL *string `json:"large_image_url"`
				} `json:"jpg"`
			} `json:"images"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil
	}
	return raw.Data.Images.JPG.LargeImageURL
}

// ── Shared upsert logic ────────────────────────────────────────────────────────

type importResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (h *ImportHandler) upsertAnimeList(ctx context.Context, userID int, items []malXMLAnime) importResult {
	result := importResult{Errors: []string{}}

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

		// Skip if already in library
		existing, _ := h.media.GetByExternalID(ctx, userID, externalID)
		if existing != nil {
			result.Skipped++
			continue
		}

		// Fetch poster from Jikan — rate limit to 1 req/s
		poster := h.jikanPoster(ctx, a.ID)
		time.Sleep(350 * time.Millisecond)

		var currentProgress, totalProgress *int
		if a.Watched > 0 {
			currentProgress = &a.Watched
		}
		if a.Episodes > 0 {
			totalProgress = &a.Episodes
		}

		in := repository.CreateMediaInput{
			UserID:          userID,
			MediaType:       models.MediaTypeAnime,
			ExternalID:      &externalID,
			Title:           a.Title,
			PosterURL:       poster,
			Status:          status,
			Metadata:        map[string]any{"mal_id": a.ID, "episodes": a.Episodes},
			CurrentProgress: currentProgress,
			TotalProgress:   totalProgress,
		}

		created, err := h.media.Create(ctx, in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", a.Title, err))
			continue
		}

		if rating != nil && created != nil {
			_, _ = h.media.Update(ctx, created.ID, userID, repository.UpdateMediaInput{Rating: rating})
		}

		result.Imported++
	}

	return result
}
