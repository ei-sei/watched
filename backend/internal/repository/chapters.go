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

type ChapterRepo struct{ db *pgxpool.Pool }

func NewChapterRepo(db *pgxpool.Pool) *ChapterRepo { return &ChapterRepo{db: db} }

const chapterColumns = `id, media_item_id, chapter_number, chapter_title,
	start_page, end_page, status, note, started_at, completed_at`

func scanChapter(row pgx.Row) (*models.BookChapterLog, error) {
	c := &models.BookChapterLog{}
	err := row.Scan(
		&c.ID, &c.MediaItemID, &c.ChapterNumber, &c.ChapterTitle,
		&c.StartPage, &c.EndPage, &c.Status, &c.Note, &c.StartedAt, &c.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *ChapterRepo) List(ctx context.Context, mediaItemID int) ([]models.BookChapterLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+chapterColumns+` FROM book_chapter_logs WHERE media_item_id = $1
		 ORDER BY chapter_number`, mediaItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.BookChapterLog
	for rows.Next() {
		var c models.BookChapterLog
		if err := rows.Scan(
			&c.ID, &c.MediaItemID, &c.ChapterNumber, &c.ChapterTitle,
			&c.StartPage, &c.EndPage, &c.Status, &c.Note, &c.StartedAt, &c.CompletedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, c)
	}
	return logs, nil
}

type UpsertChapterInput struct {
	ChapterNumber int
	ChapterTitle  *string
	StartPage     *int
	EndPage       *int
	Status        models.ChapterStatus
	Note          *string
	StartedAt     *string
	CompletedAt   *string
}

func (r *ChapterRepo) Upsert(ctx context.Context, mediaItemID int, in UpsertChapterInput) (*models.BookChapterLog, error) {
	return scanChapter(r.db.QueryRow(ctx,
		`INSERT INTO book_chapter_logs
		     (media_item_id, chapter_number, chapter_title, start_page, end_page, status, note, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (media_item_id, chapter_number) DO UPDATE SET
		     chapter_title = EXCLUDED.chapter_title,
		     start_page    = EXCLUDED.start_page,
		     end_page      = EXCLUDED.end_page,
		     status        = EXCLUDED.status,
		     note          = EXCLUDED.note,
		     started_at    = EXCLUDED.started_at,
		     completed_at  = EXCLUDED.completed_at
		 RETURNING `+chapterColumns,
		mediaItemID, in.ChapterNumber, in.ChapterTitle, in.StartPage, in.EndPage,
		in.Status, in.Note, in.StartedAt, in.CompletedAt,
	))
}

func (r *ChapterRepo) Update(ctx context.Context, id, mediaItemID int, in UpsertChapterInput) (*models.BookChapterLog, error) {
	sets := []string{}
	args := []any{id, mediaItemID}

	maybeSet := func(val any) string {
		args = append(args, val)
		return fmt.Sprintf("$%d", len(args))
	}

	if in.ChapterTitle != nil {
		sets = append(sets, "chapter_title = "+maybeSet(in.ChapterTitle))
	}
	if in.StartPage != nil {
		sets = append(sets, "start_page = "+maybeSet(in.StartPage))
	}
	if in.EndPage != nil {
		sets = append(sets, "end_page = "+maybeSet(in.EndPage))
	}
	sets = append(sets, "status = "+maybeSet(in.Status))
	if in.Note != nil {
		sets = append(sets, "note = "+maybeSet(in.Note))
	}
	if in.StartedAt != nil {
		sets = append(sets, "started_at = "+maybeSet(in.StartedAt))
	}
	if in.CompletedAt != nil {
		sets = append(sets, "completed_at = "+maybeSet(in.CompletedAt))
	}

	if len(sets) == 0 {
		return r.getByID(ctx, id, mediaItemID)
	}

	query := fmt.Sprintf(
		`UPDATE book_chapter_logs SET %s WHERE id = $1 AND media_item_id = $2 RETURNING %s`,
		strings.Join(sets, ", "), chapterColumns,
	)
	return scanChapter(r.db.QueryRow(ctx, query, args...))
}

func (r *ChapterRepo) getByID(ctx context.Context, id, mediaItemID int) (*models.BookChapterLog, error) {
	return scanChapter(r.db.QueryRow(ctx,
		`SELECT `+chapterColumns+` FROM book_chapter_logs WHERE id = $1 AND media_item_id = $2`,
		id, mediaItemID,
	))
}

func (r *ChapterRepo) Delete(ctx context.Context, id, mediaItemID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM book_chapter_logs WHERE id = $1 AND media_item_id = $2`, id, mediaItemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ChapterRepo) CountByStatus(ctx context.Context, mediaItemID int) (map[models.ChapterStatus]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT status, COUNT(*) FROM book_chapter_logs WHERE media_item_id = $1 GROUP BY status`,
		mediaItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[models.ChapterStatus]int{}
	for rows.Next() {
		var s models.ChapterStatus
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, nil
}
