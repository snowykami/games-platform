package games

import (
	"context"
	"testing"
	"time"
)

func TestSortByUsagePrefersRecentUseThenCount(t *testing.T) {
	definitions := []Definition{
		{Slug: "uno"},
		{Slug: "gomoku"},
		{Slug: "xiangqi"},
		{Slug: "mahjong"},
	}
	now := time.Now().UTC()
	stats := map[string]UsageStat{
		"gomoku":  {GameSlug: "gomoku", UseCount: 8, LastUsedAt: now.Add(-time.Hour)},
		"xiangqi": {GameSlug: "xiangqi", UseCount: 1, LastUsedAt: now},
		"mahjong": {GameSlug: "mahjong", UseCount: 4, LastUsedAt: now.Add(-time.Hour)},
	}

	sorted := SortByUsage(definitions, stats)
	got := []string{}
	for _, definition := range sorted {
		got = append(got, definition.Slug)
	}
	want := []string{"xiangqi", "gomoku", "mahjong", "uno"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected sorted slugs %v, got %v", want, got)
		}
	}
}

func TestUsageStoreRecordUseIncrementsPerUser(t *testing.T) {
	store := NewMemoryUsageStore()
	ctx := context.Background()
	if err := store.RecordUse(ctx, "user_a", "uno"); err != nil {
		t.Fatalf("record first use: %v", err)
	}
	if err := store.RecordUse(ctx, "user_a", "uno"); err != nil {
		t.Fatalf("record second use: %v", err)
	}
	if err := store.RecordUse(ctx, "user_b", "uno"); err != nil {
		t.Fatalf("record other user use: %v", err)
	}

	userAStats, err := store.StatsForUser(ctx, "user_a")
	if err != nil {
		t.Fatalf("stats user a: %v", err)
	}
	userBStats, err := store.StatsForUser(ctx, "user_b")
	if err != nil {
		t.Fatalf("stats user b: %v", err)
	}
	if userAStats["uno"].UseCount != 2 {
		t.Fatalf("expected user_a count 2, got %d", userAStats["uno"].UseCount)
	}
	if userBStats["uno"].UseCount != 1 {
		t.Fatalf("expected user_b count 1, got %d", userBStats["uno"].UseCount)
	}
}
