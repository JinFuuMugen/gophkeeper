package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	authService "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
)

type AuthHandler struct {
	svc    *authService.Service
	logger *slog.Logger
}

func NewAuthHandler(svc *authService.Service, logger *slog.Logger) *AuthHandler {
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

		err := WriteError(w, http.StatusBadRequest, "invalid json")
		if err != nil {
			h.logger.Error("cannot write error", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		return
	}

	if req.Login == "" || req.Password == "" {
		err := WriteError(w, http.StatusBadRequest, "login and password required")
		if err != nil {
			h.logger.Error("cannot write error", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		return
	}

	id, err := h.svc.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, errdefs.ErrUserExists) {
			_ = WriteError(w, http.StatusConflict, "user already exists")
			return
		}

		h.logger.Error("register failed", "error", err)
		_ = WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{"user_id": id.String()})
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		err := WriteError(w, http.StatusBadRequest, "invalid json")
		if err != nil {
			h.logger.Error("cannot write error", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		return
	}

	if req.Login == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "login and password required")
		return
	}

	token, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		err := WriteError(w, http.StatusUnauthorized, "invalid credentials")
		if err != nil {
			h.logger.Error("cannot write error", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"access_token": token})
}
