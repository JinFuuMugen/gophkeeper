package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	authService "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
)

type Handler struct {
	svc    *authService.Service
	logger *slog.Logger
}

func NewHandler(svc *authService.Service, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// func (h *Handler) Routes() chi.Router {
// 	r := chi.NewRouter()
// 	r.Post("/register", h.register)
// 	r.Post("/login", h.login)
// 	return r
// }

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
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
		if err == errdefs.ErrUserExists {
			err := WriteError(w, http.StatusConflict, "user already exists")
			if err != nil {
				h.logger.Error("cannot write error", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{"user_id": id.String()})
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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
