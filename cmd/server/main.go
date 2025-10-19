package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"subs_tracker/internal/config"
	httpGateway "subs_tracker/internal/gateways/http"
	subsRepository "subs_tracker/internal/repository/subscription/postgres"
	usecaseInternal "subs_tracker/internal/usecase"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.LoadConfig()
	pgCfg := cfg.Pg
	log := setupLogger(cfg.Env)

	log.Info("starting subs tracker", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	databaseUrl := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		pgCfg.User,
		pgCfg.Password,
		pgCfg.Host,
		pgCfg.Port,
		pgCfg.Db)

	pool, err := pgxpool.New(ctx, databaseUrl)
	if err != nil {
		log.Error("failed to init storage", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	log.Debug("init database")

	sr := subsRepository.NewSubRepository(pool)

	useCases := httpGateway.UseCases{
		Sub: usecaseInternal.NewSubscription(sr),
	}

	server := httpGateway.NewServer(useCases,
		*cfg,
		log,
		httpGateway.WithHost(cfg.Server.Host),
		httpGateway.WithPort(uint16(cfg.Server.Port)),
		httpGateway.WithLogger(log),
		httpGateway.WithTimeout(cfg.Server.Timeout),
	)

	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	log.Info("starting server", slog.String("address", addr))
	if err := server.Run(ctx); err != nil {
		log.Error("server stopped with error", slog.Any("error", err))
		return
	}
	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch strings.ToLower(env) {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}
