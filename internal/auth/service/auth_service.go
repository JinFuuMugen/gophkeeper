package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/auth"
	"github.com/JinFuuMugen/GophKeeper/internal/database/repo"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/google/uuid"
)

type Service struct {
	repo      *repo.Repo
	jwtSecret string
	accessTTL time.Duration
}

func NewService(repo *repo.Repo, jwtSecret string, accessTTL time.Duration) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, accessTTL: accessTTL}
}

func (s *Service) Register(ctx context.Context, login, password string) (uuid.UUID, error) {
	exists, err := s.repo.ExistsLogin(ctx, login)
	if err != nil {
		return uuid.Nil, fmt.Errorf("cannot register user: %w", err)
	}

	if exists {
		return uuid.Nil, fmt.Errorf("cannot register user: %w", errdefs.ErrUserExists)
	}

	id := uuid.New()
	hash, err := auth.HashPassword(password)

	if err != nil {
		return uuid.Nil, fmt.Errorf("cannot register user: %w", err)
	}

	if err := s.repo.CreateUser(ctx, id, login, hash); err != nil {
		return uuid.Nil, fmt.Errorf("cannot register user: %w", err)
	}
	return id, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	u, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", fmt.Errorf("cannot login user: %w", err)
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return "", fmt.Errorf("cannot login user: %w", errdefs.ErrInvalidCredentials)
	}
	return auth.NewToken(u.ID.String(), s.jwtSecret, s.accessTTL)
}
