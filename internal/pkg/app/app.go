package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/voidcontests/backend/internal/app/router"
	"github.com/voidcontests/backend/internal/app/runner"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/lib/logger/prettyslog"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository"
	"github.com/voidcontests/backend/internal/repository/postgres"
)

type App struct {
	config *config.Config
}

func New(config *config.Config) *App {
	return &App{config}
}

func (a *App) Run() {
	ctx := context.Background()

	var logger *slog.Logger
	switch a.config.Env {
	case config.EnvLocal:
		logger = prettyslog.Init()
	case config.EnvDevelopment:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case config.EnvProduction:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	slog.SetDefault(logger)

	slog.Info("api: starting...", slog.String("env", a.config.Env))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("postgresql: could not connect establish connection", sl.Err(err))
		return
	}

	slog.Info("postgresql: ok")

	ok := runner.Ping()
	if !ok {
		slog.Error("runner: could not establich connection")
		return
	}

	slog.Info("runner: ok")

	repo := repository.New(db)
	r := router.New(a.config, repo)

	server := &http.Server{
		Addr:         a.config.Server.Address,
		Handler:      r.InitRoutes(),
		ReadTimeout:  a.config.Server.Timeout,
		WriteTimeout: a.config.Server.Timeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	go func() {
		var err error
		if err = server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("failed to start server", sl.Err(err))
				os.Exit(1)
			}
		}
	}()

	slog.Info("api: started", slog.String("address", server.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("api: shutting down...")

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error("api: error occurred on server shutting down", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("api: server stopped")
}
