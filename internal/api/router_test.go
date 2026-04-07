package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JinFuuMugen/GophKeeper/config"
	"github.com/JinFuuMugen/GophKeeper/internal/api"
	authsvc "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	itemsvc "github.com/JinFuuMugen/GophKeeper/internal/items/service"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeUserRepo struct{}

func (f *fakeUserRepo) ExistsLogin(ctx context.Context, login string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepo) CreateUser(ctx context.Context, id uuid.UUID, login, passwordHash string) error {
	return nil
}
func (f *fakeUserRepo) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	return models.User{}, nil
}

type fakeItemsRepo struct{}

func (f *fakeItemsRepo) UpsertItem(ctx context.Context, it models.Item) (models.Item, error) {
	return models.Item{}, nil
}
func (f *fakeItemsRepo) ListItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	return nil, nil
}
func (f *fakeItemsRepo) ListItemsSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.Item, error) {
	return nil, nil
}

func TestRouter_RootAndAuth(t *testing.T) {
	authService := authsvc.NewService(&fakeUserRepo{}, "secret", time.Minute)
	itemsService := itemsvc.NewService(&fakeItemsRepo{})
	cfg := &config.ServerConfig{
		JWTSecret:   "secret",
		AccessTTL:   15,
		Addr:        "localhost:8080",
		DatabaseURI: "dummy",
	}
	r := api.InitRouter(authService, itemsService, cfg, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 on root, got %d", rr.Code)
	}

	req2 := httptest.NewRequest("GET", "/items/", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when auth header missing, got %d", rr2.Code)
	}
}
