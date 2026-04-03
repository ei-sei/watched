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

// GET /stats
func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := userIDFrom(r)

	types := []models.MediaType{models.MediaTypeFilm, models.MediaTypeTVShow, models.MediaTypeBook, models.MediaTypeAnime}
	out := make([]mediaTypeStats, 0, len(types))

	for _, mt := range types {
		counts, err := h.media.CountByStatus(ctx, uid, mt)
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		avg, err := h.media.AverageRating(ctx, uid, mt)
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		if counts == nil {
			counts = []repository.StatusCount{}
		}
		out = append(out, mediaTypeStats{Type: mt, Counts: counts, AvgRating: avg})
	}

	jsonOK(w, map[string]any{"by_type": out})
}
