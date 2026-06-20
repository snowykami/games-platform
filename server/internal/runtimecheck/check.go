package runtimecheck

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/snowykami/games-platform/server/internal/config"
)

func RequireStartup(ctx context.Context, cfg config.Config) error {
	if err := checkDatabase(ctx, cfg.Database); err != nil {
		return err
	}
	if err := checkRedis(ctx, cfg.Redis); err != nil {
		return err
	}
	return nil
}

func checkDatabase(ctx context.Context, cfg config.DatabaseConfig) error {
	if !cfg.Enabled() {
		return errors.New("DB_URL is required")
	}

	startedAt := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(checkCtx, cfg.URL)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(checkCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	slog.Info("database connected", "driver", "postgres", "duration", time.Since(startedAt).String())
	return nil
}

func checkRedis(ctx context.Context, cfg config.RedisConfig) error {
	if !cfg.Enabled() {
		return errors.New("REDIS_URL is required")
	}

	startedAt := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return fmt.Errorf("redis url invalid: %w", err)
	}

	client := redis.NewClient(options)
	defer client.Close()

	if err := client.Ping(checkCtx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	slog.Info("redis connected", "duration", time.Since(startedAt).String())
	return nil
}
