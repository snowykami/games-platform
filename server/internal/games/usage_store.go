package games

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageStat struct {
	GameSlug   string    `json:"gameSlug"`
	UseCount   int       `json:"useCount"`
	LastUsedAt time.Time `json:"lastUsedAt"`
}

type UsageStore struct {
	mu    sync.RWMutex
	stats map[string]map[string]UsageStat
	db    *pgxpool.Pool
}

func NewUsageStore(ctx context.Context, databaseURL string) (*UsageStore, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return NewMemoryUsageStore(), nil
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("game usage database connection failed: %w", err)
	}
	store := NewMemoryUsageStore()
	store.db = pool
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func NewMemoryUsageStore() *UsageStore {
	return &UsageStore{stats: map[string]map[string]UsageStat{}}
}

func (s *UsageStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *UsageStore) RecordUse(ctx context.Context, userID string, gameSlug string) error {
	userID = strings.TrimSpace(userID)
	gameSlug = strings.TrimSpace(gameSlug)
	if userID == "" || gameSlug == "" {
		return errors.New("invalid_usage_record")
	}
	now := time.Now().UTC()

	if s.db != nil {
		recordCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		_, err := s.db.Exec(recordCtx, `
INSERT INTO game_usage (user_id, game_slug, use_count, last_used_at)
VALUES ($1, $2, 1, $3)
ON CONFLICT (user_id, game_slug)
DO UPDATE SET use_count = game_usage.use_count + 1, last_used_at = EXCLUDED.last_used_at
`, userID, gameSlug, now)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	userStats := s.stats[userID]
	if userStats == nil {
		userStats = map[string]UsageStat{}
		s.stats[userID] = userStats
	}
	stat := userStats[gameSlug]
	stat.GameSlug = gameSlug
	stat.UseCount++
	stat.LastUsedAt = now
	userStats[gameSlug] = stat
	return nil
}

func (s *UsageStore) StatsForUser(ctx context.Context, userID string) (map[string]UsageStat, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return map[string]UsageStat{}, nil
	}

	if s.db != nil {
		queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		rows, err := s.db.Query(queryCtx, `SELECT game_slug, use_count, last_used_at FROM game_usage WHERE user_id = $1`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		stats := map[string]UsageStat{}
		for rows.Next() {
			var stat UsageStat
			if err := rows.Scan(&stat.GameSlug, &stat.UseCount, &stat.LastUsedAt); err != nil {
				return nil, err
			}
			stats[stat.GameSlug] = stat
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return stats, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneUsageStats(s.stats[userID]), nil
}

func SortByUsage(definitions []Definition, stats map[string]UsageStat) []Definition {
	sorted := append([]Definition{}, definitions...)
	originalIndex := map[string]int{}
	for index, definition := range definitions {
		originalIndex[definition.Slug] = index
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		leftStat, leftUsed := stats[left.Slug]
		rightStat, rightUsed := stats[right.Slug]
		if leftUsed != rightUsed {
			return leftUsed
		}
		if leftUsed && rightUsed {
			if !leftStat.LastUsedAt.Equal(rightStat.LastUsedAt) {
				return leftStat.LastUsedAt.After(rightStat.LastUsedAt)
			}
			if leftStat.UseCount != rightStat.UseCount {
				return leftStat.UseCount > rightStat.UseCount
			}
		}
		return originalIndex[left.Slug] < originalIndex[right.Slug]
	})
	return sorted
}

func Exists(slug string) bool {
	for _, definition := range List() {
		if definition.Slug == slug {
			return true
		}
	}
	return false
}

func (s *UsageStore) migrate(ctx context.Context) error {
	migrateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const schema = `
CREATE TABLE IF NOT EXISTS game_usage (
	user_id TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
	game_slug TEXT NOT NULL,
	use_count INTEGER NOT NULL DEFAULT 0,
	last_used_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, game_slug)
);

CREATE INDEX IF NOT EXISTS game_usage_user_last_used_idx ON game_usage(user_id, last_used_at DESC);
`
	if _, err := s.db.Exec(migrateCtx, schema); err != nil {
		return fmt.Errorf("game usage database migration failed: %w", err)
	}
	return nil
}

func cloneUsageStats(stats map[string]UsageStat) map[string]UsageStat {
	cloned := map[string]UsageStat{}
	for slug, stat := range stats {
		cloned[slug] = stat
	}
	return cloned
}
