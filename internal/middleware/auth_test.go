package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/auth"
	"github.com/JinFuuMugen/GophKeeper/internal/middleware"
	"github.com/google/uuid"
)

func TestRequireAuth_MissingHeader(t *testing.T) {
	h := middleware.RequireAuth("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not reach handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_OK(t *testing.T) {
	userID := uuid.New()

	token, err := auth.NewToken(userID.String(), "secret", time.Minute)
	if err != nil {
		t.Fatalf("token gen err: %v", err)
	}

	called := false
	h := middleware.RequireAuth("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		got, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		if got != userID {
			t.Fatalf("expected userID %v, got %v", userID, got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatalf("expected handler called")
	}
}
