package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PortalLink struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Category  string    `json:"category"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type PortalRepo struct{ db *pgxpool.Pool }

func NewPortalRepo(db *pgxpool.Pool) *PortalRepo { return &PortalRepo{db: db} }

func (r *PortalRepo) List(ctx context.Context) ([]PortalLink, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, url, category, created_by, created_at
		 FROM portal_links ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []PortalLink
	for rows.Next() {
		var l PortalLink
		if err := rows.Scan(&l.ID, &l.Name, &l.URL, &l.Category, &l.CreatedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	if links == nil {
		links = []PortalLink{}
	}
	return links, rows.Err()
}

func (r *PortalRepo) Create(ctx context.Context, name, url, category string, createdBy int) (*PortalLink, error) {
	var l PortalLink
	err := r.db.QueryRow(ctx,
		`INSERT INTO portal_links (name, url, category, created_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, url, category, created_by, created_at`,
		name, url, category, createdBy,
	).Scan(&l.ID, &l.Name, &l.URL, &l.Category, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *PortalRepo) Update(ctx context.Context, id int, name, url, category string) (*PortalLink, error) {
	var l PortalLink
	err := r.db.QueryRow(ctx,
		`UPDATE portal_links SET name = $2, url = $3, category = $4
		 WHERE id = $1
		 RETURNING id, name, url, category, created_by, created_at`,
		id, name, url, category,
	).Scan(&l.ID, &l.Name, &l.URL, &l.Category, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *PortalRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM portal_links WHERE id = $1`, id)
	return err
}
