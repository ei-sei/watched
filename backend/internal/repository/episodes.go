package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EpisodeRepo struct{ db *pgxpool.Pool }

func NewEpisodeRepo(db *pgxpool.Pool) *EpisodeRepo { return &EpisodeRepo{db: db} }

const episodeColumns = `id, media_item_id, season_number, episode_number, watched_at, rating, note`

func scanEpisodeFields(e *models.TvEpisodeLog, scan func(...any) error) error {
	return scan(&e.ID, &e.MediaItemID, &e.SeasonNumber, &e.EpisodeNumber, &e.WatchedAt, &e.Rating, &e.Note)
}

func scanEpisode(row pgx.Row) (*models.TvEpisodeLog, error) {
	e := &models.TvEpisodeLog{}
	err := scanEpisodeFields(e, row.Scan)
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
		if err := scanEpisodeFields(&e, rows.Scan); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, nil
}

func (r *EpisodeRepo) Upsert(ctx context.Context, mediaItemID, season, episode int, watchedAt *string, rating *float64, note *string) (*models.TvEpisodeLog, error) {
	return scanEpisode(r.db.QueryRow(ctx,
		`INSERT INTO tv_episode_logs (media_item_id, season_number, episode_number, watched_at, rating, note)
		 VALUES ($1, $2, $3, COALESCE($4, NOW()), $5, $6)
		 ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
		     watched_at = COALESCE(EXCLUDED.watched_at, NOW()),
		     rating     = EXCLUDED.rating,
		     note       = EXCLUDED.note
		 RETURNING `+episodeColumns,
		mediaItemID, season, episode, watchedAt, rating, note,
	))
}

// BulkInsertWatched inserts episode rows 1..count for a given season,
// all with the same watched_at timestamp. Skips any that already exist.
func (r *EpisodeRepo) BulkInsertWatched(ctx context.Context, mediaItemID, season, count int, watchedAt *string) error {
	if count <= 0 {
		return nil
	}
	episodes := make([]int, count)
	for i := range episodes {
		episodes[i] = i + 1
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO tv_episode_logs (media_item_id, season_number, episode_number, watched_at)
		 SELECT $1, $2, ep, COALESCE($3::timestamptz, NOW())
		 FROM unnest($4::int[]) AS ep
		 ON CONFLICT (media_item_id, season_number, episode_number) DO NOTHING`,
		mediaItemID, season, watchedAt, episodes,
	)
	return err
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

// SetSeasonProgress sets the watched episode count for a season to exactly `count`.
// Episodes 1..count are inserted (skipping existing), and any above count are deleted.
func (r *EpisodeRepo) SetSeasonProgress(ctx context.Context, mediaItemID, season, count int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("SetSeasonProgress: rollback: %v", err)
		}
	}()

	// Delete episodes above the new count
	if _, err = tx.Exec(ctx,
		`DELETE FROM tv_episode_logs
		 WHERE media_item_id = $1 AND season_number = $2 AND episode_number > $3`,
		mediaItemID, season, count,
	); err != nil {
		return fmt.Errorf("delete above: %w", err)
	}

	// Insert episodes 1..count (skip existing)
	if count > 0 {
		episodes := make([]int, count)
		for i := range episodes {
			episodes[i] = i + 1
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO tv_episode_logs (media_item_id, season_number, episode_number, watched_at)
			 SELECT $1, $2, ep, NOW()
			 FROM unnest($3::int[]) AS ep
			 ON CONFLICT (media_item_id, season_number, episode_number) DO NOTHING`,
			mediaItemID, season, episodes,
		); err != nil {
			return fmt.Errorf("bulk insert: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *EpisodeRepo) CountWatched(ctx context.Context, mediaItemID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tv_episode_logs WHERE media_item_id = $1 AND watched_at IS NOT NULL`,
		mediaItemID,
	).Scan(&n)
	return n, err
}
