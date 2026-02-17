package items

import (
	"context"
	"fmt"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type repo interface {
	UpsertItem(ctx context.Context, it models.Item) (models.Item, error)
	ListItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error)
	ListItemsSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error)
}

type Service struct {
	repo repo
}

func NewService(repo repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upsert(ctx context.Context, it models.Item) (models.Item, error) {
	if it.UserID == uuid.Nil {
		return models.Item{}, fmt.Errorf("user id required")
	}

	if it.Type == "" {
		return models.Item{}, fmt.Errorf("type required")
	}

	if len(it.Encrypted) == 0 && !it.Deleted {
		return models.Item{}, fmt.Errorf("encrypted payload required")
	}

	if it.ID == uuid.Nil {
		it.ID = uuid.New()
	}

	if it.Version <= 0 {
		it.Version = 1
	}

	return s.repo.UpsertItem(ctx, it)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	return s.repo.ListItems(ctx, userID)
}

func (s *Service) SyncSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	return s.repo.ListItemsSince(ctx, userID, since)
}
