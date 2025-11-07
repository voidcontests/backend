package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/voidcontests/api/internal/app/distributor"
	"github.com/voidcontests/api/internal/app/router"
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/lib/logger/prettyslog"
	"github.com/voidcontests/api/internal/lib/logger/sl"
	broker "github.com/voidcontests/api/internal/storage/broker/redis"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
	"github.com/voidcontests/api/internal/version"
	"github.com/voidcontests/api/pkg/scheduler"
	"github.com/voidcontests/api/pkg/ton"
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

	slog.Info("api: starting...", slog.String("env", a.config.Env), version.CommitAttr, version.BranchAttr)

	db, err := postgres.New(ctx, &a.config.Postgres)
	if err != nil {
		slog.Error("postgresql: could not connect establish connection", sl.Err(err))
		return
	}

	slog.Info("postgresql: ok")

	rc := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", a.config.Redis.Address, a.config.Redis.Port),
		Password: a.config.Redis.Password,
		DB:       a.config.Redis.Db,
	})
	defer rc.Close()

	if err := rc.Ping(ctx).Err(); err != nil {
		slog.Error("redis: could not establish connection", sl.Err(err))
		return
	}

	slog.Info("redis: ok")

	repo := repository.New(db)
	brok := broker.New(rc)
	tc, err := ton.NewClient(ctx, &a.config.Ton)
	if err != nil {
		slog.Error("ton: could not establish connection", sl.Err(err))
		return
	}

	slog.Info("ton: ok")

	r := router.New(a.config, repo, brok, tc)

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

	interval := 1 * time.Minute
	task := distributor.New(repo, tc)
	scheduler := scheduler.New(interval, task)

	go func() {
		scheduler.Start(ctx)
		defer scheduler.Stop()
	}()

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
