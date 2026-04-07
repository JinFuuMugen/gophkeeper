package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/handlers"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeAuthService struct {
	registerFn func(ctx context.Context, login, password string) (uuid.UUID, error)
	loginFn    func(ctx context.Context, login, password string) (string, models.User, error)
}

func (f *fakeAuthService) Register(ctx context.Context, login, password string) (uuid.UUID, error) {
	return f.registerFn(ctx, login, password)
}
func (f *fakeAuthService) Login(ctx context.Context, login, password string) (string, models.User, error) {
	return f.loginFn(ctx, login, password)
}

func newAuthHandler(svc handlers.AuthService) *handlers.AuthHandler {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	return handlers.NewAuthHandler(svc, logger)
}

func decodeErr(body []byte) string {
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	if s, ok := m["error"].(string); ok {
		return s
	}
	return ""
}

func TestAuthRegister_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			t.Fatalf("service should not be called")
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAuthRegister_EmptyFields(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			t.Fatalf("service should not be called")
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"login":"","password":""}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != "login and password required" {
		t.Fatalf("expected %q, got %q", "login and password required", msg)
	}
}

func TestAuthRegister_UserExists(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, errdefs.ErrUserExists
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"login":"alice","password":"pass"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != "user already exists" {
		t.Fatalf("expected %q, got %q", "user already exists", msg)
	}
}

func TestAuthRegister_InternalError(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, errors.New("boom")
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"login":"alice","password":"pass"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != "internal error" {
		t.Fatalf("expected %q, got %q", "internal error", msg)
	}
}

func TestAuthRegister_Success(t *testing.T) {
	wantID := uuid.New()
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			if login != "alice" || password != "pass" {
				t.Fatalf("unexpected credentials: %s/%s", login, password)
			}
			return wantID, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"login":"alice","password":"pass"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["user_id"] != wantID.String() {
		t.Fatalf("expected user_id %s, got %v", wantID.String(), resp["user_id"])
	}
}

func TestAuthLogin_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			t.Fatalf("service should not be called")
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != "invalid json" {
		t.Fatalf("expected %q, got %q", "invalid json", msg)
	}
}

func TestAuthLogin_EmptyFields(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			t.Fatalf("service should not be called")
			return "", models.User{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"login":"","password":""}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != "login and password required" {
		t.Fatalf("expected %q, got %q", "login and password required", msg)
	}
}

func TestAuthLogin_InternalError(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			return "", models.User{}, errors.New("boom")
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"login":"alice","password":"pass"}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if msg := decodeErr(rr.Body.Bytes()); msg != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("expected %q, got %q", http.StatusText(http.StatusInternalServerError), msg)
	}
}

func TestAuthLogin_Success(t *testing.T) {
	wantToken := "tok123"
	salt := []byte{1, 2, 3, 4}
	wantSaltB64 := base64.StdEncoding.EncodeToString(salt)

	h := newAuthHandler(&fakeAuthService{
		registerFn: func(ctx context.Context, login, password string) (uuid.UUID, error) {
			return uuid.Nil, nil
		},
		loginFn: func(ctx context.Context, login, password string) (string, models.User, error) {
			if login != "alice" || password != "pass" {
				t.Fatalf("unexpected credentials: %s/%s", login, password)
			}
			return wantToken, models.User{KDFSalt: salt}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"login":"alice","password":"pass"}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] != wantToken {
		t.Fatalf("expected token %q, got %v", wantToken, resp["token"])
	}
	if resp["kdf_salt_b64"] != wantSaltB64 {
		t.Fatalf("expected kdf_salt_b64 %q, got %v", wantSaltB64, resp["kdf_salt_b64"])
	}
}
