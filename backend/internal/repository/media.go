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
	metadata, status, rating, review_text, started_at, completed_at, created_at, updated_at`

func scanMedia(row pgx.Row) (*models.MediaItem, error) {
	m := &models.MediaItem{}
	err := row.Scan(
		&m.ID, &m.UserID, &m.MediaType, &m.ExternalID, &m.Title, &m.Year, &m.PosterURL,
		&m.Metadata, &m.Status, &m.Rating, &m.ReviewText, &m.StartedAt, &m.CompletedAt,
		&m.CreatedAt, &m.UpdatedAt,
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
}

func (r *MediaRepo) List(ctx context.Context, userID int, f MediaFilter) (*models.PaginatedMedia, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
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

	offset := (f.Page - 1) * f.PerPage
	args = append(args, f.PerPage, offset)
	query := fmt.Sprintf(
		`SELECT %s FROM media_items WHERE %s ORDER BY %s %s%s LIMIT $%d OFFSET $%d`,
		mediaColumns, whereClause, sortCol, sortDir, nullsClause, len(args)-1, len(args),
	)

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
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}

	pages := total / f.PerPage
	if total%f.PerPage != 0 {
		pages++
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
	UserID     int
	MediaType  models.MediaType
	ExternalID *string
	Title      string
	Year       *int
	PosterURL  *string
	Metadata   map[string]any
	Status     models.MediaStatus
}

func (r *MediaRepo) Create(ctx context.Context, in CreateMediaInput) (*models.MediaItem, error) {
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return scanMedia(r.db.QueryRow(ctx,
		`INSERT INTO media_items (user_id, media_type, external_id, title, year, poster_url, metadata, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+mediaColumns,
		in.UserID, in.MediaType, in.ExternalID, in.Title, in.Year, in.PosterURL, in.Metadata, in.Status,
	))
}

type UpdateMediaInput struct {
	Status      *models.MediaStatus
	Rating      *float64
	ReviewText  *string
	StartedAt   *string // DATE string yyyy-mm-dd or nil to clear
	CompletedAt *string
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
