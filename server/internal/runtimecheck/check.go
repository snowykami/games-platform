package runtimecheck

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/snowykami/games-platform/server/internal/config"
)

func LogStartup(ctx context.Context, cfg config.Config) {
	checkDatabase(ctx, cfg.Database)
	checkRedis(ctx, cfg.Redis)
}

func checkDatabase(ctx context.Context, cfg config.DatabaseConfig) {
	if !cfg.Enabled() {
		slog.Info("database check skipped", "configured", false)
		return
	}

	startedAt := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(checkCtx, cfg.URL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(checkCtx); err != nil {
		slog.Error("database ping failed", "error", err)
		return
	}

	slog.Info("database connected", "driver", "postgres", "duration", time.Since(startedAt).String())
}

func checkRedis(ctx context.Context, cfg config.RedisConfig) {
	if !cfg.Enabled() {
		slog.Info("redis check skipped", "configured", false)
		return
	}

	startedAt := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		slog.Error("redis url invalid", "error", err)
		return
	}

	client := redis.NewClient(options)
	defer client.Close()

	if err := client.Ping(checkCtx).Err(); err != nil {
		slog.Error("redis ping failed", "error", err)
		return
	}

	slog.Info("redis connected", "duration", time.Since(startedAt).String())
}
