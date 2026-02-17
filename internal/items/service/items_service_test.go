package items_test

import (
	"context"
	"testing"
	"time"

	itemsvc "github.com/JinFuuMugen/GophKeeper/internal/items/service"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeRepo struct {
	upsertFn func(ctx context.Context, it models.Item) (models.Item, error)
	listFn   func(ctx context.Context, userID uuid.UUID) ([]models.Item, error)
	sinceFn  func(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error)
}

func (f *fakeRepo) UpsertItem(ctx context.Context, it models.Item) (models.Item, error) {
	return f.upsertFn(ctx, it)
}
func (f *fakeRepo) ListItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	return f.listFn(ctx, userID)
}
func (f *fakeRepo) ListItemsSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error) {
	return f.sinceFn(ctx, userID, since)
}

func TestUpsert_RequiresUserID(t *testing.T) {
	svc := itemsvc.NewService(&fakeRepo{})
	_, err := svc.Upsert(context.Background(), models.Item{UserID: uuid.Nil, Type: "text"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpsert_RequiresType(t *testing.T) {
	svc := itemsvc.NewService(&fakeRepo{})
	_, err := svc.Upsert(context.Background(), models.Item{UserID: uuid.New()})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpsert_RequiresEncryptedWhenNotDeleted(t *testing.T) {
	svc := itemsvc.NewService(&fakeRepo{})
	_, err := svc.Upsert(context.Background(), models.Item{UserID: uuid.New(), Type: "text", Encrypted: nil, Deleted: false})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpsert_OK_PassesToRepo(t *testing.T) {
	u := uuid.New()
	id := uuid.New()

	repo := &fakeRepo{
		upsertFn: func(ctx context.Context, it models.Item) (models.Item, error) {
			if it.UserID != u || it.ID == uuid.Nil || it.Type != "text" || len(it.Encrypted) == 0 {
				t.Fatalf("bad item passed to repo: %+v", it)
			}
			it.ID = id
			it.CreatedAt = time.Now()
			it.UpdatedAt = it.CreatedAt
			return it, nil
		},
	}

	svc := itemsvc.NewService(repo)
	out, err := svc.Upsert(context.Background(), models.Item{
		UserID:    u,
		Type:      "text",
		Encrypted: []byte{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ID != id {
		t.Fatalf("expected id %v got %v", id, out.ID)
	}
}

func TestList_RequiresUserID(t *testing.T) {
	svc := itemsvc.NewService(&fakeRepo{})
	_, err := svc.List(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}
