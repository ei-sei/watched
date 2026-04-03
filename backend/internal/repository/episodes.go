package repository

import (
	"context"
	"errors"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EpisodeRepo struct{ db *pgxpool.Pool }

func NewEpisodeRepo(db *pgxpool.Pool) *EpisodeRepo { return &EpisodeRepo{db: db} }

const episodeColumns = `id, media_item_id, season_number, episode_number, watched_at, rating, note`

func scanEpisode(row pgx.Row) (*models.TvEpisodeLog, error) {
	e := &models.TvEpisodeLog{}
	err := row.Scan(&e.ID, &e.MediaItemID, &e.SeasonNumber, &e.EpisodeNumber, &e.WatchedAt, &e.Rating, &e.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (r *EpisodeRepo) List(ctx context.Context, mediaItemID int) ([]models.TvEpisodeLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+episodeColumns+` FROM tv_episode_logs WHERE media_item_id = $1
		 ORDER BY season_number, episode_number`, mediaItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TvEpisodeLog
	for rows.Next() {
		var e models.TvEpisodeLog
		if err := rows.Scan(&e.ID, &e.MediaItemID, &e.SeasonNumber, &e.EpisodeNumber, &e.WatchedAt, &e.Rating, &e.Note); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, nil
}

func (r *EpisodeRepo) Upsert(ctx context.Context, mediaItemID, season, episode int, watchedAt *string, rating *float64, note *string) (*models.TvEpisodeLog, error) {
	return scanEpisode(r.db.QueryRow(ctx,
		`INSERT INTO tv_episode_logs (media_item_id, season_number, episode_number, watched_at, rating, note)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
		     watched_at = EXCLUDED.watched_at,
		     rating     = EXCLUDED.rating,
		     note       = EXCLUDED.note
		 RETURNING `+episodeColumns,
		mediaItemID, season, episode, watchedAt, rating, note,
	))
}

func (r *EpisodeRepo) Delete(ctx context.Context, id, mediaItemID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM tv_episode_logs WHERE id = $1 AND media_item_id = $2`, id, mediaItemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *EpisodeRepo) CountWatched(ctx context.Context, mediaItemID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tv_episode_logs WHERE media_item_id = $1 AND watched_at IS NOT NULL`,
		mediaItemID,
	).Scan(&n)
	return n, err
}
