package handler

import (
	"net/http"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/ei-sei/brsti/internal/repository"
)

type StatsHandler struct {
	media *repository.MediaRepo
}

func NewStatsHandler(media *repository.MediaRepo) *StatsHandler {
	return &StatsHandler{media: media}
}

type mediaTypeStats struct {
	Type    models.MediaType                  `json:"type"`
	Counts  []repository.StatusCount          `json:"counts"`
	AvgRating *float64                        `json:"avg_rating"`
}

// GET /stats/summary
func (h *StatsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	s, err := h.media.GetSummary(r.Context(), uid)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	jsonOK(w, s)
}

// GET /stats
func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)

	byType, err := h.media.AllTypeStats(r.Context(), uid)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	types := []models.MediaType{models.MediaTypeFilm, models.MediaTypeTVShow, models.MediaTypeBook, models.MediaTypeAnime}
	out := make([]mediaTypeStats, 0, len(types))
	for _, mt := range types {
		ts := byType[mt]
		if ts.Counts == nil {
			ts.Counts = []repository.StatusCount{}
		}
		out = append(out, mediaTypeStats{Type: mt, Counts: ts.Counts, AvgRating: ts.AvgRating})
	}

	jsonOK(w, map[string]any{"by_type": out})
}
