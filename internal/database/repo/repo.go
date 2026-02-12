package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(ctx context.Context, databaseURI string) (*Repo, error) {
	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("cannot create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cannot ping database: %w", err)
	}

	return &Repo{pool: pool}, nil
}

func (r *Repo) CreateUser(ctx context.Context, id uuid.UUID, login, passwordHash string) error {
	const q = `INSERT INTO users (id, login, password_hash) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, q, id, login, passwordHash)
	if err == nil {
		return nil
	}

	return fmt.Errorf("cannot create user: %w", err)
}

func (r *Repo) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	const q = `SELECT id, login, password_hash, created_at FROM users WHERE login=$1`

	var u models.User
	err := r.pool.QueryRow(ctx, q, login).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if err == nil {
		return u, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, errdefs.ErrUserNotFound
	}
	return models.User{}, fmt.Errorf("cannot get user by login %s: %w", login, err)
}

func (r *Repo) ExistsLogin(ctx context.Context, login string) (bool, error) {
	const q = `SELECT 1 FROM users WHERE login=$1`
	var one int
	err := r.pool.QueryRow(ctx, q, login).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *Repo) UpsertItem(ctx context.Context, it models.Item) (models.Item, error) {
	const q = `
INSERT INTO items (id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
ON CONFLICT (id) DO UPDATE
SET
	type = EXCLUDED.type,
	encrypted = EXCLUDED.encrypted,
	metadata = EXCLUDED.metadata,
	version = EXCLUDED.version,
	deleted = EXCLUDED.deleted,
	updated_at = now()
WHERE items.user_id = EXCLUDED.user_id AND items.version < EXCLUDED.version
RETURNING id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at
`
	var out models.Item
	err := r.pool.QueryRow(
		ctx,
		q,
		it.ID,
		it.UserID,
		it.Type,
		it.Encrypted,
		it.Metadata,
		it.Version,
		it.Deleted,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.Type,
		&out.Encrypted,
		&out.Metadata,
		&out.Version,
		&out.Deleted,
		&out.CreatedAt,
		&out.UpdatedAt,
	)

	if err == nil {
		return out, nil
	}

	return r.GetItem(ctx, it.UserID, it.ID)
}

func (r *Repo) GetItem(ctx context.Context, userID, id uuid.UUID) (models.Item, error) {
	const q = `
SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at
FROM items
WHERE user_id=$1 AND id=$2
`
	var out models.Item
	if err := r.pool.QueryRow(ctx, q, userID, id).Scan(
		&out.ID,
		&out.UserID,
		&out.Type,
		&out.Encrypted,
		&out.Metadata,
		&out.Version,
		&out.Deleted,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return models.Item{}, fmt.Errorf("cannot get item: %w", err)
	}
	return out, nil
}

func (r *Repo) ListItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	const q = `
SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at
FROM items
WHERE user_id=$1
ORDER BY updated_at ASC
`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("cannot list items: %w", err)
	}
	defer rows.Close()

	var out []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(
			&it.ID,
			&it.UserID,
			&it.Type,
			&it.Encrypted,
			&it.Metadata,
			&it.Version,
			&it.Deleted,
			&it.CreatedAt,
			&it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("cannot scan item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate items: %w", err)
	}
	return out, nil
}

func (r *Repo) ListItemsSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error) {
	const q = `
SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at
FROM items
WHERE user_id=$1 AND updated_at > $2
ORDER BY updated_at ASC
`
	rows, err := r.pool.Query(ctx, q, userID, since)
	if err != nil {
		return nil, fmt.Errorf("cannot list items since: %w", err)
	}
	defer rows.Close()

	var out []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(
			&it.ID,
			&it.UserID,
			&it.Type,
			&it.Encrypted,
			&it.Metadata,
			&it.Version,
			&it.Deleted,
			&it.CreatedAt,
			&it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("cannot scan item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate items: %w", err)
	}
	return out, nil
}
