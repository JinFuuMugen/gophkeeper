package items

import (
	"context"
	"fmt"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/cryptokit"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type repo interface {
	UpsertItem(ctx context.Context, it models.Item) (models.Item, error)
	ListItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error)
	ListItemsSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error)
}

type Service struct {
	repo  repo
	crypt *cryptokit.MasterCrypt
}

func NewService(repo repo, crypt *cryptokit.MasterCrypt) *Service {
	return &Service{repo: repo, crypt: crypt}
}

func (s *Service) Upsert(ctx context.Context, it models.Item) (models.Item, error) {
	if it.UserID == uuid.Nil {
		return models.Item{}, fmt.Errorf("user id required")
	}
	if it.Type == "" {
		return models.Item{}, fmt.Errorf("type required")
	}

	if len(it.Data) == 0 && !it.Deleted {
		return models.Item{}, fmt.Errorf("data required")
	}

	if it.ID == uuid.Nil {
		it.ID = uuid.New()
	}
	if it.Version <= 0 {
		it.Version = 1
	}

	aad := []byte(it.UserID.String() + "|" + it.ID.String() + "|" + it.Type + "|" + fmt.Sprint(it.Version))

	if it.Deleted {
		it.Encrypted = []byte("deleted")
	} else {
		enc, err := s.crypt.Encrypt(it.Data, aad)
		if err != nil {
			return models.Item{}, fmt.Errorf("encrypt item: %w", err)
		}
		it.Encrypted = enc
	}

	saved, err := s.repo.UpsertItem(ctx, it)
	if err != nil {
		return models.Item{}, err
	}

	if saved.Deleted {
		saved.Data = nil
		return saved, nil
	}

	plain, err := s.crypt.Decrypt(saved.Encrypted, aad)
	if err != nil {
		return models.Item{}, fmt.Errorf("decrypt saved item: %w", err)
	}
	saved.Data = plain

	return saved, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	items, err := s.repo.ListItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		if items[i].Deleted {
			items[i].Data = nil
			continue
		}
		aad := []byte(items[i].UserID.String() + "|" + items[i].ID.String() + "|" + items[i].Type + "|" + fmt.Sprint(items[i].Version))
		plain, err := s.crypt.Decrypt(items[i].Encrypted, aad)
		if err != nil {
			return nil, fmt.Errorf("decrypt item %s: %w", items[i].ID, err)
		}
		items[i].Data = plain
	}

	return items, nil
}

func (s *Service) SyncSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	items, err := s.repo.ListItemsSince(ctx, userID, since)
	if err != nil {
		return nil, err
	}

	for i := range items {
		if items[i].Deleted {
			items[i].Data = nil
			continue
		}
		aad := []byte(items[i].UserID.String() + "|" + items[i].ID.String() + "|" + items[i].Type + "|" + fmt.Sprint(items[i].Version))
		plain, err := s.crypt.Decrypt(items[i].Encrypted, aad)
		if err != nil {
			return nil, fmt.Errorf("decrypt item %s: %w", items[i].ID, err)
		}
		items[i].Data = plain
	}
	return items, nil

}
