package repository

import (
	"context"
	"errors"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RewatchRepo struct{ db *pgxpool.Pool }

func NewRewatchRepo(db *pgxpool.Pool) *RewatchRepo { return &RewatchRepo{db: db} }

func (r *RewatchRepo) List(ctx context.Context, userID, mediaID int) ([]models.Rewatch, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, media_id, started_at::text, finished_at::text, created_at
		 FROM rewatches WHERE user_id = $1 AND media_id = $2
		 ORDER BY created_at DESC`,
		userID, mediaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Rewatch
	for rows.Next() {
		var rw models.Rewatch
		if err := rows.Scan(&rw.ID, &rw.UserID, &rw.MediaID, &rw.StartedAt, &rw.FinishedAt, &rw.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	if out == nil {
		out = []models.Rewatch{}
	}
	return out, rows.Err()
}

func (r *RewatchRepo) Create(ctx context.Context, userID, mediaID int, startedAt, finishedAt *string) (*models.Rewatch, error) {
	rw := &models.Rewatch{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO rewatches (user_id, media_id, started_at, finished_at)
		 VALUES ($1, $2, $3::date, $4::date)
		 RETURNING id, user_id, media_id, started_at::text, finished_at::text, created_at`,
		userID, mediaID, startedAt, finishedAt,
	).Scan(&rw.ID, &rw.UserID, &rw.MediaID, &rw.StartedAt, &rw.FinishedAt, &rw.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rw, nil
}

func (r *RewatchRepo) Delete(ctx context.Context, userID, id int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM rewatches WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *RewatchRepo) Count(ctx context.Context, userID, mediaID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM rewatches WHERE user_id = $1 AND media_id = $2`,
		userID, mediaID,
	).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}
