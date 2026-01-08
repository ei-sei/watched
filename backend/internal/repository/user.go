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

func (r *UserRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// Invite code helpers
func (r *UserRepo) ListInvites(ctx context.Context) ([]models.InviteCode, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, created_at, used_at FROM invite_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []models.InviteCode
	for rows.Next() {
		var c models.InviteCode
		if err := rows.Scan(&c.Code, &c.CreatedAt, &c.UsedAt); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, nil
}

func (r *UserRepo) CreateInvite(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO invite_codes (code) VALUES ($1)`, code)
	return err
}

func (r *UserRepo) DeleteInvite(ctx context.Context, code string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM invite_codes WHERE code = $1 AND used_at IS NULL`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("invite code not found or already used")
	}
	return nil
}

type AdminStats struct {
	TotalUsers         int    `json:"total_users"`
	TotalItems         int    `json:"total_items"`
	TotalEpisodes      int    `json:"total_episodes"`
	TotalChapters      int    `json:"total_chapters"`
	UnusedInvites      int    `json:"unused_invites"`
	DBSizeBytes        int64  `json:"db_size_bytes"`
	DBSizeHuman        string `json:"db_size_human"`
	DBSizeLimitBytes   int64  `json:"db_size_limit_bytes"`
	DBHealthy          bool   `json:"db_healthy"`
	MemTotalKB         int64  `json:"mem_total_kb"`
	MemUsedKB          int64  `json:"mem_used_kb"`
	MemAvailableKB     int64  `json:"mem_available_kb"`
	DiskTotalBytes     int64  `json:"disk_total_bytes"`
	DiskUsedBytes      int64  `json:"disk_used_bytes"`
}

func (r *UserRepo) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	s := &AdminStats{DBHealthy: true}
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM media_items),
			(SELECT COUNT(*) FROM tv_episode_logs),
			(SELECT COUNT(*) FROM book_chapter_logs),
			(SELECT COUNT(*) FROM invite_codes WHERE used_at IS NULL),
			pg_database_size(current_database()),
			pg_size_pretty(pg_database_size(current_database()))
	`).Scan(&s.TotalUsers, &s.TotalItems, &s.TotalEpisodes, &s.TotalChapters, &s.UnusedInvites, &s.DBSizeBytes, &s.DBSizeHuman)
	if err != nil {
		s.DBHealthy = false
	}
	return s, nil
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
