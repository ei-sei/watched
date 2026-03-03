package handler

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
)

type PortalHandler struct {
	portal *repository.PortalRepo
	client *http.Client
}

func NewPortalHandler(portal *repository.PortalRepo) *PortalHandler {
	return &PortalHandler{
		portal: portal,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// GET /portal
func (h *PortalHandler) List(w http.ResponseWriter, r *http.Request) {
	links, err := h.portal.List(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, links)
}

// GET /portal/status
func (h *PortalHandler) Status(w http.ResponseWriter, r *http.Request) {
	links, err := h.portal.List(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	type result struct {
		ID         int  `json:"id"`
		OK         bool `json:"ok"`
		StatusCode int  `json:"status_code"`
	}

	results := make([]result, len(links))
	var wg sync.WaitGroup
	for i, l := range links {
		wg.Add(1)
		go func(i int, url string, id int) {
			defer wg.Done()
			code := h.ping(url)
			results[i] = result{ID: id, OK: code > 0 && code < 500, StatusCode: code}
		}(i, l.URL, l.ID)
	}
	wg.Wait()

	jsonOK(w, results)
}

func (h *PortalHandler) ping(url string) int {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := h.client.Do(req)
	if err != nil {
		return 0
	}
	resp.Body.Close()
	return resp.StatusCode
}

// POST /portal
func (h *PortalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Category string `json:"category"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name == "" || body.URL == "" || (body.Category != "movies_tv" && body.Category != "anime") {
		jsonErr(w, http.StatusBadRequest, "name, url, and valid category are required")
		return
	}

	claims := auth.ClaimsFrom(r.Context())
	link, err := h.portal.Create(r.Context(), body.Name, body.URL, body.Category, claims.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonCreated(w, link)
}

// PATCH /portal/{id}
func (h *PortalHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Category string `json:"category"`
	}
	if err := decode(r, &body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name == "" || body.URL == "" || (body.Category != "movies_tv" && body.Category != "anime") {
		jsonErr(w, http.StatusBadRequest, "name, url, and valid category are required")
		return
	}

	link, err := h.portal.Update(r.Context(), id, body.Name, body.URL, body.Category)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, link)
}

// DELETE /portal/{id}
func (h *PortalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.portal.Delete(r.Context(), id); err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
