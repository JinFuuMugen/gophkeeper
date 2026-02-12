package repo

import (
	"context"
	"errors"
	"fmt"

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
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, errdefs.ErrUserNotFound
	}
	return u, fmt.Errorf("cannot get user by login %s: %w", login, err)
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
