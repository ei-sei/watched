package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeatureFlag struct {
	Key       string    `json:"key"`
	IsPremium bool      `json:"is_premium"`
	UpdatedAt time.Time `json:"updated_at"`
}

var validFlagKeys = map[string]bool{
	"stats":    true,
	"trending": true,
	"portal":   true,
}

type FlagsRepo struct{ db *pgxpool.Pool }

func NewFlagsRepo(db *pgxpool.Pool) *FlagsRepo { return &FlagsRepo{db: db} }

func IsValidFlagKey(key string) bool { return validFlagKeys[key] }

func (r *FlagsRepo) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	rows, err := r.db.Query(ctx, `SELECT key, is_premium, updated_at FROM feature_flags ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		if err := rows.Scan(&f.Key, &f.IsPremium, &f.UpdatedAt); err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	if flags == nil {
		flags = []FeatureFlag{}
	}
	return flags, rows.Err()
}

func (r *FlagsRepo) Set(ctx context.Context, key string, isPremium bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE feature_flags SET is_premium = $2, updated_at = NOW() WHERE key = $1`,
		key, isPremium,
	)
	return err
}

// GetMap returns a map of key -> is_premium for embedding in API responses.
func (r *FlagsRepo) GetMap(ctx context.Context) (map[string]bool, error) {
	flags, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(flags))
	for _, f := range flags {
		m[f.Key] = f.IsPremium
	}
	return m, nil
}

// IsPremium returns whether a specific feature is currently premium-gated.
func (r *FlagsRepo) IsPremium(ctx context.Context, key string) (bool, error) {
	var isPremium bool
	err := r.db.QueryRow(ctx,
		`SELECT is_premium FROM feature_flags WHERE key = $1`, key,
	).Scan(&isPremium)
	return isPremium, err
}
