package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/auth"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type userRepo interface {
	ExistsLogin(ctx context.Context, login string) (bool, error)
	CreateUser(ctx context.Context, id uuid.UUID, login, passwordHash string) error
	GetUserByLogin(ctx context.Context, login string) (models.User, error)
}

type Service struct {
	repo      userRepo
	jwtSecret string
	accessTTL time.Duration
}

func NewService(repo userRepo, jwtSecret string, accessTTL time.Duration) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		accessTTL: accessTTL,
	}
}

func (s *Service) Register(ctx context.Context, login, password string) (uuid.UUID, error) {

	id := uuid.New()

	hash, err := auth.HashPassword(password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password failed: %w", err)
	}

	if err := s.repo.CreateUser(ctx, id, login, hash); err != nil {
		if err == errdefs.ErrUserExists {
			return uuid.Nil, err
		} else {
			return uuid.Nil, fmt.Errorf("create user failed: %w", err)
		}
	}

	return id, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (string, models.User, error) {
	u, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", u, fmt.Errorf("login failed: %w", err)
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return "", models.User{}, errdefs.ErrInvalidCredentials
	}

	token, err := auth.NewToken(u.ID.String(), s.jwtSecret, s.accessTTL)
	if err != nil {
		return "", models.User{}, fmt.Errorf("token generation failed: %w", err)
	}

	return token, u, nil
}
