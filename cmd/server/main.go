package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"subs_tracker/internal/config"
	httpGateway "subs_tracker/internal/gateways/http"
	subsRepository "subs_tracker/internal/repository/subscription/postgres"
	usecaseInternal "subs_tracker/internal/usecase"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		log.Error("failed to init storage", slog.String(
			"Error",
			fmt.Sprint("error")))
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

	log.Info("starting server", slog.String("address", cfg.Server.Host+":"+strconv.Itoa(cfg.Server.Port)))
	err = server.Run()
	if err != nil {
		log.Error(err.Error())
		return
	}

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
	}
	return log
}
