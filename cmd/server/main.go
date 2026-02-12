package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JinFuuMugen/GophKeeper/config"
	"github.com/JinFuuMugen/GophKeeper/internal/api"
	authService "github.com/JinFuuMugen/GophKeeper/internal/auth/service"
	"github.com/JinFuuMugen/GophKeeper/internal/database/repo"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {

	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadServerConfig()
	if err != nil {
		logger.Error("cannot load server config", "error", err)
	}
	logger.Info("server config loaded")

	accessTTL := time.Duration(cfg.AccessTTL) * time.Minute

	ctx := context.Background()

	repo, err := repo.NewRepo(ctx, cfg.DatabaseURI)
	if err != nil {
		logger.Error("cannot init db repository", "error", err)
	}
	logger.Info("repo inited")

	authService := authService.NewService(repo, cfg.JWTSecret, accessTTL)
	logger.Info("auth service inited")

	rout := api.InitRouter(authService, logger)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
	)
	defer stop()

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      rout,
		ReadTimeout:  10,
		WriteTimeout: 10,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	logger.Info("server started")

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Error("cannot start server", "error", err)
			panic(err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	} else {
		logger.Info("http server stopped")
	}

}
