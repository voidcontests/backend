package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tonkeeper/tongo/liteapi"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/backend/internal/app/router"
	"github.com/voidcontests/backend/internal/app/runner"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/lib/logger/prettyslog"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository"
	"github.com/voidcontests/backend/internal/repository/postgres"
	"github.com/voidcontests/backend/internal/ton"
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

	slog.Info("starting API server...", slog.String("env", a.config.Env))

	slog.Debug("connecting to liteapi mainnet server")

	var err error
	ton.Networks[ton.MainnetID], err = liteapi.NewClientWithDefaultMainnet()
	if err != nil {
		slog.Error("failed init mainnet liteapi client", sl.Err(err))
		return
	}

	slog.Debug("connecting to liteapi testnet server")

	ton.Networks[ton.TestnetID], err = liteapi.NewClientWithDefaultTestnet()
	if err != nil {
		slog.Error("failed init testnet liteapi client", sl.Err(err))
		return
	}

	slog.Debug("successfully connected to liteapi servers")

	mainnet, _ := tonconnect.NewTonConnect(ton.Mainnet(), a.config.TonProof.PayloadSignatureKey, tonconnect.WithLifeTimePayload(a.config.TonProof.PayloadLifetimeSeconds.Nanoseconds()), tonconnect.WithLifeTimeProof(int64(a.config.TonProof.ProofLifetimeSeconds.Nanoseconds())))
	testnet, _ := tonconnect.NewTonConnect(ton.Testnet(), a.config.TonProof.PayloadSignatureKey, tonconnect.WithLifeTimePayload(a.config.TonProof.PayloadLifetimeSeconds.Nanoseconds()), tonconnect.WithLifeTimeProof(int64(a.config.TonProof.ProofLifetimeSeconds.Nanoseconds())))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("could not connect to postgresql", sl.Err(err))
		return
	}

	slog.Info("successfully connected to postgresql")

	ok := runner.Ping()
	if !ok {
		slog.Error("could not connect to runner service")
		return
	}

	slog.Info("runner is ok")

	repo := repository.New(db)
	r := router.New(a.config, repo, mainnet, testnet)

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

	slog.Info("server started", slog.String("address", server.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("server shutting down")

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error("error occurred on server shutting down", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("API server stopped")
}
