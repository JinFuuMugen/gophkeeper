package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (uuid.UUID, error)
	Login(ctx context.Context, login, password string) (token string, u models.User, err error)
}

type AuthHandler struct {
	svc    AuthService
	logger *slog.Logger
}

func NewAuthHandler(svc AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		svc:    svc,
		logger: logger,
	}
}

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Login == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "login and password required")
		return
	}

	id, err := h.svc.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, errdefs.ErrUserExists) {
			WriteError(w, http.StatusConflict, "user already exists")
			return
		}

		h.logger.Error("register failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{"user_id": id.String()})
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token      string `json:"token"`
	KDFSaltB64 string `json:"kdf_salt_b64"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Login == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "login and password required")
		return
	}

	token, u, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	saltB64 := base64.StdEncoding.EncodeToString(u.KDFSalt)

	WriteJSON(w, http.StatusOK, loginResponse{Token: token, KDFSaltB64: saltB64})
}
