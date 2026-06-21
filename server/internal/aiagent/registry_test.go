package aiagent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func TestRegistryEnsuresSingleAgentPerPlayer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	created := 0
	registry := NewRegistry(func(string, string) *Agent {
		created++
		return NewAgent(&Runner{
			Game:     "undercover",
			Level:    aiplayer.LevelLLM,
			Provider: &fakeProvider{enabled: true, decision: aiplayer.Decision{ActionID: "speak"}},
		}, 1, nil)
	})

	first, err := registry.Ensure(ctx, "room_1", "ai_1")
	if err != nil {
		t.Fatalf("ensure first: %v", err)
	}
	second, err := registry.Ensure(ctx, "room_1", "ai_1")
	if err != nil {
		t.Fatalf("ensure second: %v", err)
	}
	if first != second || created != 1 || registry.Count() != 1 {
		t.Fatalf("expected one reused agent, created=%d count=%d", created, registry.Count())
	}
}

func TestRegistrySubmitStartsAndUsesAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	intents := make(chan gameactor.PlayerIntent, 1)
	registry := NewRegistry(func(string, string) *Agent {
		return NewAgent(&Runner{
			Game:     "undercover",
			Level:    aiplayer.LevelLLM,
			Provider: &fakeProvider{enabled: true, decision: aiplayer.Decision{ActionID: "speak", Speech: "我补一句。"}},
		}, 1, func(_ context.Context, intent gameactor.PlayerIntent) error {
			intents <- intent
			return nil
		})
	})

	err := registry.Submit(ctx, gameactor.AgentEvent{
		ID:           "req_1",
		RoomID:       "room_1",
		PlayerID:     "ai_1",
		Type:         gameactor.AgentOptionalSpeech,
		LegalActions: []aiplayer.LegalAction{{ID: "speak", Label: "发言"}},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	select {
	case intent := <-intents:
		if intent.PlayerID != "ai_1" || intent.Speech != "我补一句。" {
			t.Fatalf("unexpected intent: %+v", intent)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for intent")
	}
}

func TestRegistryRemoveClosesAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := NewRegistry(func(string, string) *Agent {
		return NewAgent(&Runner{}, 1, nil)
	})
	agent, err := registry.Ensure(ctx, "room_1", "ai_1")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	registry.Remove("room_1", "ai_1")
	if registry.Count() != 0 {
		t.Fatalf("expected empty registry, got %d", registry.Count())
	}
	if err := agent.Submit(ctx, gameactor.AgentEvent{}); !errors.Is(err, gameactor.ErrActorClosed) {
		t.Fatalf("expected closed agent, got %v", err)
	}
}

func TestRegistryRejectsMissingFactory(t *testing.T) {
	registry := NewRegistry(nil)
	_, err := registry.Ensure(context.Background(), "room_1", "ai_1")
	if !errors.Is(err, ErrAgentFactoryMissing) {
		t.Fatalf("expected missing factory, got %v", err)
	}
}

func TestRegistryEnsureWithUsesCallFactory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := NewRegistry(nil)
	agent, err := registry.EnsureWith(ctx, "room_1", "ai_1", func(string, string) *Agent {
		return NewAgent(&Runner{}, 1, nil)
	})
	if err != nil {
		t.Fatalf("ensure with: %v", err)
	}
	if agent == nil || registry.Count() != 1 {
		t.Fatalf("expected one created agent, count=%d", registry.Count())
	}
}
