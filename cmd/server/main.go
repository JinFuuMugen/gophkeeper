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
	itemsService "github.com/JinFuuMugen/GophKeeper/internal/items/service"
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
		os.Exit(1)
	}
	logger.Info("server config loaded")

	if err := config.ValidateConfig(cfg); err != nil {
		logger.Error("cannot validate config", "error", err)
		os.Exit(1)
	}

	accessTTL := time.Duration(cfg.AccessTTL) * time.Minute
	ctx := context.Background()

	repo, err := repo.NewRepo(ctx, cfg.DatabaseURI)
	if err != nil {
		logger.Error("cannot init db repository", "error", err)
		os.Exit(1)
	}
	logger.Info("repo inited")

	authSvc := authService.NewService(repo, cfg.JWTSecret, accessTTL)
	logger.Info("auth service inited")

	itemsSvc := itemsService.NewService(repo)
	logger.Info("items service inited")

	rout := api.InitRouter(authSvc, itemsSvc, cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
	)
	defer stop()

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      rout,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		var runErr error

		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			logger.Info("server starting with TLS", "addr", cfg.Addr, "cert", cfg.TLSCertFile, "key", cfg.TLSKeyFile)
			runErr = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			logger.Info("server starting without TLS", "addr", cfg.Addr)
			runErr = srv.ListenAndServe()
		}

		if runErr != nil && runErr != http.ErrServerClosed {
			errCh <- runErr
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
