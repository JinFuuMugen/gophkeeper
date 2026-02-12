package api

import (
	"log/slog"
	"net/http"

	authService "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func InitRouter(authService *authService.Service, logger *slog.Logger) chi.Router {
	rout := chi.NewMux()

	h := handlers.NewHandler(authService, logger)

	rout.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	})

	rout.Post("/register", h.Register)
	rout.Post("/login", h.Login)

	return rout
}
