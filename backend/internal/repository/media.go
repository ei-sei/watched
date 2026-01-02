package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepo struct{ db *pgxpool.Pool }

func NewMediaRepo(db *pgxpool.Pool) *MediaRepo { return &MediaRepo{db: db} }

const mediaColumns = `id, user_id, media_type, external_id, title, year, poster_url,
	metadata, status, rating, review_text, started_at, completed_at,
	current_progress, total_progress, created_at, updated_at`

func scanMedia(row pgx.Row) (*models.MediaItem, error) {
	m := &models.MediaItem{}
	err := row.Scan(
		&m.ID, &m.UserID, &m.MediaType, &m.ExternalID, &m.Title, &m.Year, &m.PosterURL,
		&m.Metadata, &m.Status, &m.Rating, &m.ReviewText, &m.StartedAt, &m.CompletedAt,
		&m.CurrentProgress, &m.TotalProgress, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

type MediaFilter struct {
	MediaType *models.MediaType
	Status    *models.MediaStatus
	Search    *string
	Sort      string // created_at | updated_at | rating | title | year
	Order     string // asc | desc
	Page      int
	PerPage   int
	NoLimit   bool // bypass pagination, return all results
}

func (r *MediaRepo) List(ctx context.Context, userID int, f MediaFilter) (*models.PaginatedMedia, error) {
	if !f.NoLimit {
		if f.Page < 1 {
			f.Page = 1
		}
		if f.PerPage < 1 || f.PerPage > 100 {
			f.PerPage = 20
		}
	}

	args := []any{userID}
	where := []string{"user_id = $1"}

	if f.MediaType != nil {
		args = append(args, *f.MediaType)
		where = append(where, fmt.Sprintf("media_type = $%d", len(args)))
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.Search != nil {
		args = append(args, "%"+*f.Search+"%")
		where = append(where, fmt.Sprintf("title ILIKE $%d", len(args)))
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM media_items WHERE "+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	// Whitelist sort columns to prevent SQL injection
	allowedSort := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"rating":     "rating",
		"title":      "title",
		"year":       "year",
	}
	sortCol, ok := allowedSort[f.Sort]
	if !ok {
		sortCol = "created_at"
	}
	sortDir := "DESC"
	if f.Order == "asc" {
		sortDir = "ASC"
	}
	// NULLs last for rating/year
	nullsClause := ""
	if sortCol == "rating" || sortCol == "year" {
		nullsClause = " NULLS LAST"
	}

	var query string
	if f.NoLimit {
		query = fmt.Sprintf(
			`SELECT %s FROM media_items WHERE %s ORDER BY %s %s%s`,
			mediaColumns, whereClause, sortCol, sortDir, nullsClause,
		)
	} else {
		offset := (f.Page - 1) * f.PerPage
		args = append(args, f.PerPage, offset)
		query = fmt.Sprintf(
			`SELECT %s FROM media_items WHERE %s ORDER BY %s %s%s LIMIT $%d OFFSET $%d`,
			mediaColumns, whereClause, sortCol, sortDir, nullsClause, len(args)-1, len(args),
		)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.MediaItem, 0)
	for rows.Next() {
		m := models.MediaItem{}
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.MediaType, &m.ExternalID, &m.Title, &m.Year, &m.PosterURL,
			&m.Metadata, &m.Status, &m.Rating, &m.ReviewText, &m.StartedAt, &m.CompletedAt,
			&m.CurrentProgress, &m.TotalProgress, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}

	pages := 1
	if !f.NoLimit && f.PerPage > 0 {
		pages = total / f.PerPage
		if total%f.PerPage != 0 {
			pages++
		}
	}

	return &models.PaginatedMedia{
		Items:   items,
		Total:   total,
		Page:    f.Page,
		PerPage: f.PerPage,
		Pages:   pages,
	}, nil
}

func (r *MediaRepo) GetByID(ctx context.Context, id, userID int) (*models.MediaItem, error) {
	return scanMedia(r.db.QueryRow(ctx,
		`SELECT `+mediaColumns+` FROM media_items WHERE id = $1 AND user_id = $2`, id, userID,
	))
}

func (r *MediaRepo) GetByExternalID(ctx context.Context, userID int, externalID string) (*models.MediaItem, error) {
	return scanMedia(r.db.QueryRow(ctx,
		`SELECT `+mediaColumns+` FROM media_items WHERE user_id = $1 AND external_id = $2`, userID, externalID,
	))
}

type CreateMediaInput struct {
	UserID          int
	MediaType       models.MediaType
	ExternalID      *string
	Title           string
	Year            *int
	PosterURL       *string
	Metadata        map[string]any
	Status          models.MediaStatus
	CurrentProgress *int
	TotalProgress   *int
}

func (r *MediaRepo) Create(ctx context.Context, in CreateMediaInput) (*models.MediaItem, error) {
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return scanMedia(r.db.QueryRow(ctx,
		`INSERT INTO media_items (user_id, media_type, external_id, title, year, poster_url, metadata, status, current_progress, total_progress)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING `+mediaColumns,
		in.UserID, in.MediaType, in.ExternalID, in.Title, in.Year, in.PosterURL, in.Metadata, in.Status,
		in.CurrentProgress, in.TotalProgress,
	))
}

type UpdateMediaInput struct {
	Status          *models.MediaStatus
	Rating          *float64
	ReviewText      *string
	StartedAt       *string // DATE string yyyy-mm-dd or nil to clear
	CompletedAt     *string
	CurrentProgress *int
	TotalProgress   *int
	Metadata        map[string]any
}

func (r *MediaRepo) Update(ctx context.Context, id, userID int, in UpdateMediaInput) (*models.MediaItem, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{id, userID}

	if in.Status != nil {
		args = append(args, *in.Status)
		sets = append(sets, fmt.Sprintf("status = $%d", len(args)))
	}
	if in.Rating != nil {
		args = append(args, *in.Rating)
		sets = append(sets, fmt.Sprintf("rating = $%d", len(args)))
	}
	if in.ReviewText != nil {
		args = append(args, *in.ReviewText)
		sets = append(sets, fmt.Sprintf("review_text = $%d", len(args)))
	}
	if in.StartedAt != nil {
		args = append(args, *in.StartedAt)
		sets = append(sets, fmt.Sprintf("started_at = $%d", len(args)))
	}
	if in.CompletedAt != nil {
		args = append(args, *in.CompletedAt)
		sets = append(sets, fmt.Sprintf("completed_at = $%d", len(args)))
	}
	if in.CurrentProgress != nil {
		args = append(args, *in.CurrentProgress)
		sets = append(sets, fmt.Sprintf("current_progress = $%d", len(args)))
	}
	if in.TotalProgress != nil {
		args = append(args, *in.TotalProgress)
		sets = append(sets, fmt.Sprintf("total_progress = $%d", len(args)))
	}
	if in.Metadata != nil {
		args = append(args, in.Metadata)
		sets = append(sets, fmt.Sprintf("metadata = $%d", len(args)))
	}

	query := fmt.Sprintf(
		`UPDATE media_items SET %s WHERE id = $1 AND user_id = $2 RETURNING %s`,
		strings.Join(sets, ", "), mediaColumns,
	)
	return scanMedia(r.db.QueryRow(ctx, query, args...))
}

func (r *MediaRepo) Delete(ctx context.Context, id, userID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM media_items WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Stats helpers

type StatusCount struct {
	Status models.MediaStatus
	Count  int
}

func (r *MediaRepo) CountByStatus(ctx context.Context, userID int, mt models.MediaType) ([]StatusCount, error) {
	rows, err := r.db.Query(ctx,
		`SELECT status, COUNT(*) FROM media_items WHERE user_id = $1 AND media_type = $2 GROUP BY status`,
		userID, mt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StatusCount
	for rows.Next() {
		var sc StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, nil
}

func (r *MediaRepo) AverageRating(ctx context.Context, userID int, mt models.MediaType) (*float64, error) {
	var avg *float64
	err := r.db.QueryRow(ctx,
		`SELECT AVG(rating) FROM media_items WHERE user_id = $1 AND media_type = $2 AND rating IS NOT NULL`,
		userID, mt,
	).Scan(&avg)
	return avg, err
}

type RatingBucket struct {
	Rating int `json:"rating"`
	Count  int `json:"count"`
}

type StatsSummary struct {
	Films              FilmStats      `json:"films"`
	TVShows            TVStats        `json:"tv_shows"`
	Books              BookStats      `json:"books"`
	Anime              AnimeStats     `json:"anime"`
	RatingDistribution []RatingBucket `json:"rating_distribution"`
	LongestStreakDays  int            `json:"longest_streak_days"`
	CurrentStreakDays  int            `json:"current_streak_days"`
	EstimatedMinutes   int            `json:"estimated_minutes"`
	CompletionRate     float64        `json:"completion_rate"`
}

type FilmStats struct {
	Total      int      `json:"total"`
	ThisMonth  int      `json:"this_month"`
	AvgRating  *float64 `json:"avg_rating"`
}

type TVStats struct {
	Total             int `json:"total"`
	InProgress        int `json:"in_progress"`
	EpisodesThisMonth int `json:"episodes_this_month"`
}

type BookStats struct {
	Total             int `json:"total"`
	InProgress        int `json:"in_progress"`
	ChaptersThisMonth int `json:"chapters_this_month"`
}

type AnimeStats struct {
	Total             int `json:"total"`
	InProgress        int `json:"in_progress"`
	EpisodesThisMonth int `json:"episodes_this_month"`
}

func (r *MediaRepo) GetSummary(ctx context.Context, userID int) (*StatsSummary, error) {
	s := &StatsSummary{}

	// Films
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE completed_at >= date_trunc('month', now())),
			AVG(rating) FILTER (WHERE rating IS NOT NULL)
		FROM media_items
		WHERE user_id = $1 AND media_type = 'film'
	`, userID).Scan(&s.Films.Total, &s.Films.ThisMonth, &s.Films.AvgRating)
	if err != nil {
		return nil, err
	}

	// TV
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			(SELECT COUNT(*) FROM tv_episode_logs e
			 JOIN media_items m ON m.id = e.media_item_id
			 WHERE m.user_id = $1 AND m.media_type = 'tv_show'
			 AND e.watched_at >= date_trunc('month', now()))
		FROM media_items
		WHERE user_id = $1 AND media_type = 'tv_show'
	`, userID).Scan(&s.TVShows.Total, &s.TVShows.InProgress, &s.TVShows.EpisodesThisMonth)
	if err != nil {
		return nil, err
	}

	// Books
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			(SELECT COUNT(*) FROM book_chapter_logs c
			 JOIN media_items m ON m.id = c.media_item_id
			 WHERE m.user_id = $1
			 AND c.completed_at >= date_trunc('month', now()))
		FROM media_items
		WHERE user_id = $1 AND media_type = 'book'
	`, userID).Scan(&s.Books.Total, &s.Books.InProgress, &s.Books.ChaptersThisMonth)
	if err != nil {
		return nil, err
	}

	// Anime
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			(SELECT COUNT(*) FROM tv_episode_logs e
			 JOIN media_items m ON m.id = e.media_item_id
			 WHERE m.user_id = $1 AND m.media_type = 'anime'
			 AND e.watched_at >= date_trunc('month', now()))
		FROM media_items
		WHERE user_id = $1 AND media_type = 'anime'
	`, userID).Scan(&s.Anime.Total, &s.Anime.InProgress, &s.Anime.EpisodesThisMonth)
	if err != nil {
		return nil, err
	}

	// Completion rate (completed / all started, i.e. not want_to)
	var completed, started int
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status != 'want_to')
		FROM media_items WHERE user_id = $1
	`, userID).Scan(&completed, &started)
	if err != nil {
		return nil, err
	}
	if started > 0 {
		s.CompletionRate = float64(completed) / float64(started)
	}

	// Estimated time (minutes): films×110 + tv eps×42 + anime eps×24
	err = r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM media_items WHERE user_id = $1 AND media_type = 'film' AND status = 'completed') * 110 +
			(SELECT COUNT(*) FROM tv_episode_logs e JOIN media_items m ON m.id = e.media_item_id WHERE m.user_id = $1 AND m.media_type = 'tv_show') * 42 +
			(SELECT COUNT(*) FROM tv_episode_logs e JOIN media_items m ON m.id = e.media_item_id WHERE m.user_id = $1 AND m.media_type = 'anime') * 24
	`, userID).Scan(&s.EstimatedMinutes)
	if err != nil {
		return nil, err
	}

	// Rating distribution
	rows, err := r.db.Query(ctx, `
		SELECT ROUND(rating)::int, COUNT(*)
		FROM media_items
		WHERE user_id = $1 AND rating IS NOT NULL
		GROUP BY ROUND(rating)::int
		ORDER BY ROUND(rating)::int
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b RatingBucket
		if err := rows.Scan(&b.Rating, &b.Count); err != nil {
			return nil, err
		}
		s.RatingDistribution = append(s.RatingDistribution, b)
	}
	if s.RatingDistribution == nil {
		s.RatingDistribution = []RatingBucket{}
	}

	// Streaks — activity = any episode logged or item completed
	err = r.db.QueryRow(ctx, `
		WITH activity_dates AS (
			SELECT DISTINCT DATE(e.watched_at) AS d
			FROM tv_episode_logs e
			JOIN media_items m ON m.id = e.media_item_id
			WHERE m.user_id = $1 AND e.watched_at IS NOT NULL
			UNION
			SELECT DISTINCT DATE(c.completed_at)
			FROM book_chapter_logs c
			JOIN media_items m ON m.id = c.media_item_id
			WHERE m.user_id = $1 AND c.completed_at IS NOT NULL
			UNION
			SELECT DISTINCT DATE(completed_at)
			FROM media_items
			WHERE user_id = $1 AND completed_at IS NOT NULL
		),
		groups AS (
			SELECT d, d - (ROW_NUMBER() OVER (ORDER BY d))::int * INTERVAL '1 day' AS grp
			FROM activity_dates
		),
		streak_lengths AS (
			SELECT grp, COUNT(*) AS len, MAX(d) AS last_day
			FROM groups GROUP BY grp
		)
		SELECT
			COALESCE(MAX(len), 0),
			COALESCE((
				SELECT len FROM streak_lengths
				WHERE last_day >= CURRENT_DATE - 1
				ORDER BY last_day DESC LIMIT 1
			), 0)
		FROM streak_lengths
	`, userID).Scan(&s.LongestStreakDays, &s.CurrentStreakDays)
	if err != nil {
		return nil, err
	}

	return s, nil
}
