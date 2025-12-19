package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ei-sei/brsti/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct{ db *pgxpool.Pool }

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

const userColumns = `id, username, email, password_hash, display_name, avatar_url,
	is_admin, is_premium, is_public, failed_attempts, locked_until, created_at, updated_at`

func scanUser(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.IsAdmin, &u.IsPremium, &u.IsPublic, &u.FailedAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id int) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = $1`, username))
}

func (r *UserRepo) Create(ctx context.Context, username, passwordHash string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name)
		 VALUES ($1, $2, $1)
		 RETURNING `+userColumns,
		username, passwordHash))
}

func (r *UserRepo) UpdateLoginFail(ctx context.Context, id, attempts int, lockedUntil *time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET failed_attempts = $2, locked_until = $3, updated_at = NOW() WHERE id = $1`,
		id, attempts, lockedUntil)
	return err
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id int, displayName, avatarURL *string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET
		    display_name = COALESCE($2, display_name),
		    avatar_url   = COALESCE($3, avatar_url),
		    updated_at   = NOW()
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, displayName, avatarURL))
}

func (r *UserRepo) UpdatePublic(ctx context.Context, id int, isPublic bool) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET is_public = $2, updated_at = NOW()
		 WHERE id = $1 RETURNING `+userColumns,
		id, isPublic))
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int, hash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, hash)
	return err
}

func (r *UserRepo) UpdateFlags(ctx context.Context, id int, isPremium, isAdmin *bool) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET
		    is_premium = COALESCE($2, is_premium),
		    is_admin   = COALESCE($3, is_admin),
		    updated_at = NOW()
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, isPremium, isAdmin))
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
			&u.IsAdmin, &u.IsPremium, &u.IsPublic, &u.FailedAttempts, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Invite code helpers
func (r *UserRepo) CreateInvite(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO invite_codes (code) VALUES ($1)`, code)
	return err
}

func (r *UserRepo) UseInvite(ctx context.Context, code string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE invite_codes SET used_at = NOW() WHERE code = $1 AND used_at IS NULL`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("invalid or already used invite code")
	}
	return nil
}
