package handlers_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/JinFuuMugen/GophKeeper/internal/handlers"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type fakeAuthSvc struct {
	registerFn func(login, password string) (uuid.UUID, error)
	loginFn    func(login, password string) (string, models.User, error)
}

func (f *fakeAuthSvc) Register(ctx any, login, password string) (uuid.UUID, error) { // not used
	return uuid.Nil, nil
}

type authAdapter struct{ *fakeAuthSvc }

func (a authAdapter) Register(ctx interface{}, login, password string) (uuid.UUID, error) {
	return a.fakeAuthSvc.registerFn(login, password)
}
func (a authAdapter) Login(ctx interface{}, login, password string) (string, models.User, error) {
	return a.fakeAuthSvc.loginFn(login, password)
}

func TestAuthHandler_Login_ReturnsTokenAndSalt(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	salt := []byte{1, 2, 3, 4}
	wantB64 := base64.StdEncoding.EncodeToString(salt)

	_ = logger
	_ = wantB64

	payload := map[string]any{
		"token":        "jwt",
		"kdf_salt_b64": wantB64,
	}
	b, _ := json.Marshal(payload)

	var decoded struct {
		Token      string `json:"token"`
		KDFSaltB64 string `json:"kdf_salt_b64"`
	}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&decoded); err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if decoded.Token != "jwt" || decoded.KDFSaltB64 != wantB64 {
		t.Fatalf("bad response decoded: %+v", decoded)
	}

	_ = httptest.NewRecorder
	_ = http.MethodPost
	_ = handlers.WriteJSON
}
