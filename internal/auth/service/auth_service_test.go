package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authsvc "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeUserRepo struct {
	existsLoginFn    func(ctx context.Context, login string) (bool, error)
	createUserFn     func(ctx context.Context, id uuid.UUID, login, passwordHash string) error
	getUserByLoginFn func(ctx context.Context, login string) (models.User, error)
}

func (f *fakeUserRepo) ExistsLogin(ctx context.Context, login string) (bool, error) {
	return f.existsLoginFn(ctx, login)
}
func (f *fakeUserRepo) CreateUser(ctx context.Context, id uuid.UUID, login, passwordHash string) error {
	return f.createUserFn(ctx, id, login, passwordHash)
}
func (f *fakeUserRepo) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	return f.getUserByLoginFn(ctx, login)
}

func TestRegister_UserExists(t *testing.T) {
	repo := &fakeUserRepo{
		existsLoginFn:    func(ctx context.Context, login string) (bool, error) { return true, nil },
		createUserFn:     func(ctx context.Context, id uuid.UUID, login, passwordHash string) error { return nil },
		getUserByLoginFn: func(ctx context.Context, login string) (models.User, error) { return models.User{}, nil },
	}

	svc := authsvc.NewService(repo, "secret", time.Minute)
	_, err := svc.Register(context.Background(), "alice", "pass")
	if !errors.Is(err, errdefs.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got: %v", err)
	}
}

func TestRegister_OK(t *testing.T) {
	created := false
	repo := &fakeUserRepo{
		existsLoginFn: func(ctx context.Context, login string) (bool, error) { return false, nil },
		createUserFn: func(ctx context.Context, id uuid.UUID, login, passwordHash string) error {
			created = true
			if id == uuid.Nil {
				t.Fatalf("expected non-nil id")
			}
			if login != "alice" {
				t.Fatalf("unexpected login: %s", login)
			}
			if passwordHash == "" {
				t.Fatalf("expected password hash")
			}
			return nil
		},
		getUserByLoginFn: func(ctx context.Context, login string) (models.User, error) { return models.User{}, nil },
	}

	svc := authsvc.NewService(repo, "secret", time.Minute)
	id, err := svc.Register(context.Background(), "alice", "pass")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id == uuid.Nil || !created {
		t.Fatalf("expected created user, id=%v created=%v", id, created)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := &fakeUserRepo{
		existsLoginFn: func(ctx context.Context, login string) (bool, error) { return false, nil },
		createUserFn:  func(ctx context.Context, id uuid.UUID, login, passwordHash string) error { return nil },
		getUserByLoginFn: func(ctx context.Context, login string) (models.User, error) {

			return models.User{
				ID:           uuid.New(),
				Login:        "alice",
				PasswordHash: "$2a$10$8jXXNKOunGOHANizOCN5luKyYy/3uHdNZNiyE/7bwN76EXKB6g6IK",
				KDFSalt:      []byte{1, 2, 3},
			}, nil
		},
	}

	svc := authsvc.NewService(repo, "secret", time.Minute)
	_, _, err := svc.Login(context.Background(), "alice", "pass")
	if !errors.Is(err, errdefs.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestLogin_OK_ReturnsTokenAndUser(t *testing.T) {
	repo := &fakeUserRepo{
		existsLoginFn: func(ctx context.Context, login string) (bool, error) { return false, nil },
		createUserFn:  func(ctx context.Context, id uuid.UUID, login, passwordHash string) error { return nil },
		getUserByLoginFn: func(ctx context.Context, login string) (models.User, error) {

			return models.User{
				ID:           uuid.New(),
				Login:        "lesha",
				PasswordHash: "$2a$10$f.BLdhPKuJ/fUXYI3w/uiuvjqf79SlWsFzW7dRYXM1KxEQLbplI5u",
				KDFSalt:      []byte{9, 9, 9},
			}, nil
		},
	}

	svc := authsvc.NewService(repo, "secret", time.Minute)
	token, u, err := svc.Login(context.Background(), "lesha", "12345")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if u.ID == uuid.Nil || len(u.KDFSalt) == 0 {
		t.Fatalf("expected user with salt")
	}
}
