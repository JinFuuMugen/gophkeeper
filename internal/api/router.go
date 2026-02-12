package api

import (
	"log/slog"
	"net/http"

	"github.com/JinFuuMugen/GophKeeper/config"
	authService "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/handlers"
	itemsService "github.com/JinFuuMugen/GophKeeper/internal/items/service"
	"github.com/JinFuuMugen/GophKeeper/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func InitRouter(authService *authService.Service, itemsService *itemsService.Service, cfg *config.ServerConfig, logger *slog.Logger) chi.Router {
	rout := chi.NewMux()

	authHandler := handlers.NewAuthHandler(authService, logger)
	itemsHandler := handlers.NewItemsHandler(itemsService, logger)

	rout.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	})

	rout.Post("/register", authHandler.Register)
	rout.Post("/login", authHandler.Login)

	rout.Route("/items", func(rr chi.Router) {
		rr.Use(func(next http.Handler) http.Handler { return middleware.RequireAuth(cfg.JWTSecret, next) })

		rr.Post("/", itemsHandler.UpsertItem)
		rr.Get("/", itemsHandler.ListItems)
		rr.Get("/sync", itemsHandler.SyncItems)
	})

	return rout
}
