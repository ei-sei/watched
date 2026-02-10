package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepo struct{ db *pgxpool.Pool }

func NewMediaRepo(db *pgxpool.Pool) *MediaRepo { return &MediaRepo{db: db} }

const mediaColumns = `id, user_id, media_type, external_id, title, year, poster_url,
	metadata, status, rating, review_text, started_at, completed_at,
	current_progress, total_progress, created_at, updated_at`

func scanMediaFields(m *models.MediaItem, scan func(...any) error) error {
	return scan(
		&m.ID, &m.UserID, &m.MediaType, &m.ExternalID, &m.Title, &m.Year, &m.PosterURL,
		&m.Metadata, &m.Status, &m.Rating, &m.ReviewText, &m.StartedAt, &m.CompletedAt,
		&m.CurrentProgress, &m.TotalProgress, &m.CreatedAt, &m.UpdatedAt,
	)
}

func scanMedia(row pgx.Row) (*models.MediaItem, error) {
	m := &models.MediaItem{}
	err := scanMediaFields(m, row.Scan)
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

	var (
		query string
		total int
	)
	if f.NoLimit {
		// No pagination — skip COUNT entirely; derive total from result length.
		query = fmt.Sprintf(
			`SELECT %s FROM media_items WHERE %s ORDER BY %s %s%s`,
			mediaColumns, whereClause, sortCol, sortDir, nullsClause,
		)
	} else {
		// Use COUNT(*) OVER() to get total and rows in a single roundtrip.
		offset := (f.Page - 1) * f.PerPage
		args = append(args, f.PerPage, offset)
		query = fmt.Sprintf(
			`SELECT %s, COUNT(*) OVER() FROM media_items WHERE %s ORDER BY %s %s%s LIMIT $%d OFFSET $%d`,
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
		if f.NoLimit {
			if err := scanMediaFields(&m, rows.Scan); err != nil {
				return nil, err
			}
		} else {
			if err := scanMediaFields(&m, func(dest ...any) error {
				return rows.Scan(append(dest, &total)...)
			}); err != nil {
				return nil, err
			}
		}
		items = append(items, m)
	}

	if f.NoLimit {
		total = len(items)
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
	StartedAt       *string // DATE yyyy-mm-dd
	CompletedAt     *string // DATE yyyy-mm-dd
}

func (r *MediaRepo) Create(ctx context.Context, in CreateMediaInput) (*models.MediaItem, error) {
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return scanMedia(r.db.QueryRow(ctx,
		`INSERT INTO media_items (user_id, media_type, external_id, title, year, poster_url, metadata, status, current_progress, total_progress, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING `+mediaColumns,
		in.UserID, in.MediaType, in.ExternalID, in.Title, in.Year, in.PosterURL, in.Metadata, in.Status,
		in.CurrentProgress, in.TotalProgress, in.StartedAt, in.CompletedAt,
	))
}

type UpdateMediaInput struct {
	Title           *string
	Status          *models.MediaStatus
	Rating          *float64
	ReviewText      *string
	StartedAt       *string // DATE string yyyy-mm-dd or nil to clear
	CompletedAt     *string
	CurrentProgress *int
	TotalProgress   *int
	Metadata        map[string]any
	Year            *int
	PosterURL       *string
}

func (r *MediaRepo) Update(ctx context.Context, id, userID int, in UpdateMediaInput) (*models.MediaItem, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{id, userID}

	if in.Title != nil {
		args = append(args, *in.Title)
		sets = append(sets, fmt.Sprintf("title = $%d", len(args)))
	}
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
	if in.Year != nil {
		args = append(args, *in.Year)
		sets = append(sets, fmt.Sprintf("year = $%d", len(args)))
	}
	if in.PosterURL != nil {
		args = append(args, *in.PosterURL)
		sets = append(sets, fmt.Sprintf("poster_url = $%d", len(args)))
	}

	query := fmt.Sprintf(
		`UPDATE media_items SET %s WHERE id = $1 AND user_id = $2 RETURNING %s`,
		strings.Join(sets, ", "), mediaColumns,
	)
	return scanMedia(r.db.QueryRow(ctx, query, args...))
}

func (r *MediaRepo) DeleteAllByType(ctx context.Context, userID int, mediaType models.MediaType) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM media_items WHERE user_id = $1 AND media_type = $2`, userID, mediaType)
	if err != nil {
		return 0, fmt.Errorf("delete all by type: %w", err)
	}
	return tag.RowsAffected(), nil
}

type MissingPosterItem struct {
	ID    int
	MalID int
}

// GetAnimeMissingPosters returns anime items for a user that have a mal_id in
// their metadata but no poster_url set.
func (r *MediaRepo) GetAnimeMissingPosters(ctx context.Context, userID int) ([]MissingPosterItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, (metadata->>'mal_id')::int
		 FROM media_items
		 WHERE user_id = $1
		   AND media_type = 'anime'
		   AND poster_url IS NULL
		   AND metadata->>'mal_id' IS NOT NULL`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get anime missing posters: %w", err)
	}
	defer rows.Close()

	var out []MissingPosterItem
	for rows.Next() {
		var item MissingPosterItem
		if err := rows.Scan(&item.ID, &item.MalID); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// MissingMetadataItem is returned by GetMissingMetadata.
type MissingMetadataItem struct {
	ID            int
	UserID        int
	MediaType     string
	Title         string
	Year          *int
	PosterURL     *string
	ExternalID    *string
	Metadata      map[string]any
	TotalProgress *int
}

func (m *MissingMetadataItem) IsPosterMissing() bool { return m.PosterURL == nil }

// GetMissingMetadata returns all items for a user that are missing a poster,
// a year, or (for episodic/book types) a total_progress value.
func (r *MediaRepo) GetMissingMetadata(ctx context.Context, userID int) ([]MissingMetadataItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, media_type, title, year, poster_url, external_id, metadata, total_progress
		 FROM media_items
		 WHERE user_id = $1
		   AND (
		         poster_url IS NULL
		      OR year IS NULL
		      OR (media_type IN ('anime', 'tv_show', 'book') AND total_progress IS NULL)
		   )`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get missing metadata: %w", err)
	}
	defer rows.Close()

	var out []MissingMetadataItem
	for rows.Next() {
		var item MissingMetadataItem
		var metaBytes []byte
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.MediaType, &item.Title,
			&item.Year, &item.PosterURL, &item.ExternalID, &metaBytes, &item.TotalProgress,
		); err != nil {
			continue
		}
		if metaBytes != nil {
			_ = json.Unmarshal(metaBytes, &item.Metadata)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *MediaRepo) SetPoster(ctx context.Context, id int, posterURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE media_items SET poster_url = $2, updated_at = NOW() WHERE id = $1`, id, posterURL)
	return err
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

type TypeStats struct {
	Counts    []StatusCount
	AvgRating *float64
}

// AllTypeStats returns status counts and average rating for every media type in
// a single query instead of one query per type per metric.
func (r *MediaRepo) AllTypeStats(ctx context.Context, userID int) (map[models.MediaType]TypeStats, error) {
	rows, err := r.db.Query(ctx, `
		WITH counts AS (
			SELECT media_type, status, COUNT(*) AS cnt
			FROM media_items WHERE user_id = $1
			GROUP BY media_type, status
		),
		avgs AS (
			SELECT media_type, AVG(rating) AS avg_rating
			FROM media_items WHERE user_id = $1 AND rating IS NOT NULL
			GROUP BY media_type
		)
		SELECT c.media_type, c.status, c.cnt, a.avg_rating
		FROM counts c
		LEFT JOIN avgs a ON a.media_type = c.media_type
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[models.MediaType]TypeStats{}
	for rows.Next() {
		var mt models.MediaType
		var st models.MediaStatus
		var count int
		var avg *float64
		if err := rows.Scan(&mt, &st, &count, &avg); err != nil {
			return nil, err
		}
		ts := out[mt]
		ts.Counts = append(ts.Counts, StatusCount{Status: st, Count: count})
		ts.AvgRating = avg // same for every row of this type
		out[mt] = ts
	}
	return out, rows.Err()
}

type RatingBucket struct {
	Rating int `json:"rating"`
	Count  int `json:"count"`
}

type MonthlyActivity struct {
	Month string `json:"month"` // "2026-01"
	Films int    `json:"films"`
	TV    int    `json:"tv"`
	Books int    `json:"books"`
	Anime int    `json:"anime"`
}

type StatsSummary struct {
	Films              FilmStats        `json:"films"`
	TVShows            TVStats          `json:"tv_shows"`
	Books              BookStats        `json:"books"`
	Anime              AnimeStats       `json:"anime"`
	RatingDistribution []RatingBucket   `json:"rating_distribution"`
	MonthlyActivity    []MonthlyActivity `json:"monthly_activity"`
	LongestStreakDays  int              `json:"longest_streak_days"`
	CurrentStreakDays  int              `json:"current_streak_days"`
	EstimatedMinutes   int              `json:"estimated_minutes"`
	CompletionRate     float64          `json:"completion_rate"`
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
	// Single JOIN across tv_episode_logs — previously joined twice (TV + anime separately).
	err = r.db.QueryRow(ctx, `
		WITH film_mins AS (
			SELECT COUNT(*) * 110 AS mins
			FROM media_items
			WHERE user_id = $1 AND media_type = 'film' AND status = 'completed'
		),
		episode_mins AS (
			SELECT COALESCE(SUM(CASE WHEN m.media_type = 'tv_show' THEN 42 ELSE 24 END), 0) AS mins
			FROM tv_episode_logs e
			JOIN media_items m ON m.id = e.media_item_id
			WHERE m.user_id = $1 AND m.media_type IN ('tv_show', 'anime')
		)
		SELECT film_mins.mins + episode_mins.mins
		FROM film_mins, episode_mins
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

	// Monthly activity — last 12 months, broken down by category
	monthRows, err := r.db.Query(ctx, `
		SELECT
			TO_CHAR(activity_date, 'YYYY-MM') AS month,
			COUNT(*) FILTER (WHERE category = 'film')    AS films,
			COUNT(*) FILTER (WHERE category = 'tv_show') AS tv,
			COUNT(*) FILTER (WHERE category = 'book')    AS books,
			COUNT(*) FILTER (WHERE category = 'anime')   AS anime
		FROM (
			-- TV + anime episodes watched
			SELECT e.watched_at::date AS activity_date, m.media_type::text AS category
			FROM tv_episode_logs e
			JOIN media_items m ON m.id = e.media_item_id
			WHERE m.user_id = $1 AND e.watched_at IS NOT NULL
			UNION ALL
			-- Book chapters completed
			SELECT c.completed_at::date AS activity_date, 'book' AS category
			FROM book_chapter_logs c
			JOIN media_items m ON m.id = c.media_item_id
			WHERE m.user_id = $1 AND c.completed_at IS NOT NULL
			UNION ALL
			-- Films completed (episodes handle TV/anime progress)
			SELECT completed_at AS activity_date, 'film' AS category
			FROM media_items
			WHERE user_id = $1 AND media_type = 'film'
			  AND status = 'completed' AND completed_at IS NOT NULL
		) acts(activity_date, category)
		WHERE activity_date >= CURRENT_DATE - INTERVAL '11 months'
		GROUP BY month
		ORDER BY month
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("monthly activity query: %w", err)
	}
	defer monthRows.Close()
	activityMap := make(map[string]MonthlyActivity)
	for monthRows.Next() {
		var m MonthlyActivity
		if err := monthRows.Scan(&m.Month, &m.Films, &m.TV, &m.Books, &m.Anime); err != nil {
			return nil, fmt.Errorf("scan monthly activity: %w", err)
		}
		activityMap[m.Month] = m
	}
	// Fill all 12 months so the frontend always gets a complete array
	now := time.Now()
	for i := 11; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		key := t.Format("2006-01")
		if entry, ok := activityMap[key]; ok {
			s.MonthlyActivity = append(s.MonthlyActivity, entry)
		} else {
			s.MonthlyActivity = append(s.MonthlyActivity, MonthlyActivity{Month: key})
		}
	}

	return s, nil
}

// AutoMarkInactive moves in_progress items to on_hold when no activity for 30+ days.
// Only applies to items that have at least one episode/chapter logged.
func (r *MediaRepo) AutoMarkInactive(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		UPDATE media_items
		SET status = 'on_hold', updated_at = NOW()
		WHERE status = 'in_progress'
		  AND media_type IN ('tv_show', 'anime', 'book')
		  AND (
		    (media_type IN ('tv_show', 'anime')
		     AND EXISTS (SELECT 1 FROM tv_episode_logs WHERE media_item_id = media_items.id AND watched_at IS NOT NULL)
		     AND (SELECT MAX(watched_at) FROM tv_episode_logs WHERE media_item_id = media_items.id) < NOW() - INTERVAL '30 days')
		    OR
		    (media_type = 'book'
		     AND EXISTS (SELECT 1 FROM book_chapter_logs WHERE media_item_id = media_items.id AND completed_at IS NOT NULL)
		     AND (SELECT MAX(completed_at) FROM book_chapter_logs WHERE media_item_id = media_items.id) < NOW() - INTERVAL '30 days')
		  )
	`)
	return err
}
