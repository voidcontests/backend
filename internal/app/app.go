package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/cascadecontests/backend/internal/config"
	"github.com/cascadecontests/backend/internal/lib/logger/prettyslog"
	"github.com/cascadecontests/backend/internal/lib/logger/sl"
	"github.com/cascadecontests/backend/internal/router"
	"github.com/cascadecontests/backend/internal/ton"
	"github.com/tonkeeper/tongo/liteapi"
	"github.com/tonkeeper/tongo/tonconnect"
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

	var err error
	ton.Networks[ton.MainnetID], err = liteapi.NewClientWithDefaultMainnet()
	if err != nil {
		slog.Error("failed init mainnet liteapi client", sl.Err(err))
		return
	}

	ton.Networks[ton.TestnetID], err = liteapi.NewClientWithDefaultMainnet()
	if err != nil {
		slog.Error("failed init testnet liteapi client", sl.Err(err))
		return
	}

	// TODO: handle errors
	mainnet, _ := tonconnect.NewTonConnect(ton.Mainnet(), a.config.TonProof.PayloadSignatureKey, tonconnect.WithLifeTimePayload(a.config.TonProof.PayloadLifetimeSeconds.Nanoseconds()), tonconnect.WithLifeTimeProof(int64(a.config.TonProof.ProofLifetimeSeconds.Nanoseconds())))
	testnet, _ := tonconnect.NewTonConnect(ton.Testnet(), a.config.TonProof.PayloadSignatureKey, tonconnect.WithLifeTimePayload(a.config.TonProof.PayloadLifetimeSeconds.Nanoseconds()), tonconnect.WithLifeTimeProof(int64(a.config.TonProof.ProofLifetimeSeconds.Nanoseconds())))

	r := router.New(a.config, mainnet, testnet)

	server := &http.Server{
		Addr:         a.config.Server.Address,
		Handler:      r.InitRoutes(),
		ReadTimeout:  a.config.Server.Timeout,
		WriteTimeout: a.config.Server.Timeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
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
