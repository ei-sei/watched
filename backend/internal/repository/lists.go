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

type ListRepo struct{ db *pgxpool.Pool }

func NewListRepo(db *pgxpool.Pool) *ListRepo { return &ListRepo{db: db} }

const listColumns = `id, user_id, name, description, is_public, created_at, updated_at`

func scanList(row pgx.Row) (*models.UserList, error) {
	l := &models.UserList{}
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.Description, &l.IsPublic, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return l, err
}

func (r *ListRepo) List(ctx context.Context, userID int) ([]models.UserList, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+listColumns+` FROM user_lists WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []models.UserList
	for rows.Next() {
		var l models.UserList
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.Description, &l.IsPublic, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, nil
}

func (r *ListRepo) GetByID(ctx context.Context, id int) (*models.UserList, error) {
	l, err := scanList(r.db.QueryRow(ctx,
		`SELECT `+listColumns+` FROM user_lists WHERE id = $1`, id))
	if err != nil || l == nil {
		return l, err
	}

	items, err := r.listItems(ctx, id)
	if err != nil {
		return nil, err
	}
	l.Items = items
	return l, nil
}

func (r *ListRepo) listItems(ctx context.Context, listID int) ([]models.ListItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT li.id, li.list_id, li.media_item_id, li.position, li.added_at,
		        mi.id, mi.user_id, mi.media_type, mi.external_id, mi.title, mi.year, mi.poster_url,
		        mi.metadata, mi.status, mi.rating, mi.review_text, mi.started_at, mi.completed_at,
		        mi.created_at, mi.updated_at
		 FROM list_items li
		 JOIN media_items mi ON mi.id = li.media_item_id
		 WHERE li.list_id = $1
		 ORDER BY li.position`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ListItem
	for rows.Next() {
		var li models.ListItem
		var m models.MediaItem
		if err := rows.Scan(
			&li.ID, &li.ListID, &li.MediaItemID, &li.Position, &li.AddedAt,
			&m.ID, &m.UserID, &m.MediaType, &m.ExternalID, &m.Title, &m.Year, &m.PosterURL,
			&m.Metadata, &m.Status, &m.Rating, &m.ReviewText, &m.StartedAt, &m.CompletedAt,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		li.MediaItem = &m
		items = append(items, li)
	}
	return items, nil
}

func (r *ListRepo) Create(ctx context.Context, userID int, name string, description *string, isPublic bool) (*models.UserList, error) {
	l, err := scanList(r.db.QueryRow(ctx,
		`INSERT INTO user_lists (user_id, name, description, is_public)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+listColumns,
		userID, name, description, isPublic,
	))
	if err != nil || l == nil {
		return l, err
	}
	l.Items = []models.ListItem{}
	return l, nil
}

func (r *ListRepo) Update(ctx context.Context, id, userID int, name *string, description *string, isPublic *bool) (*models.UserList, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{id, userID}

	if name != nil {
		args = append(args, *name)
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if description != nil {
		args = append(args, *description)
		sets = append(sets, fmt.Sprintf("description = $%d", len(args)))
	}
	if isPublic != nil {
		args = append(args, *isPublic)
		sets = append(sets, fmt.Sprintf("is_public = $%d", len(args)))
	}

	query := fmt.Sprintf(
		`UPDATE user_lists SET %s WHERE id = $1 AND user_id = $2 RETURNING %s`,
		strings.Join(sets, ", "), listColumns,
	)
	return scanList(r.db.QueryRow(ctx, query, args...))
}

func (r *ListRepo) Delete(ctx context.Context, id, userID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM user_lists WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ListRepo) AddItem(ctx context.Context, listID, mediaItemID, position int) (*models.ListItem, error) {
	li := &models.ListItem{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO list_items (list_id, media_item_id, position)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING
		 RETURNING id, list_id, media_item_id, position, added_at`,
		listID, mediaItemID, position,
	).Scan(&li.ID, &li.ListID, &li.MediaItemID, &li.Position, &li.AddedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // already present, conflict silenced
	}
	return li, err
}

func (r *ListRepo) RemoveItem(ctx context.Context, listID, mediaItemID int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM list_items WHERE list_id = $1 AND media_item_id = $2`, listID, mediaItemID)
	return err
}

func (r *ListRepo) ReorderItems(ctx context.Context, listID int, orderedItemIDs []int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for pos, itemID := range orderedItemIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE list_items SET position = $3 WHERE list_id = $1 AND media_item_id = $2`,
			listID, itemID, pos,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
