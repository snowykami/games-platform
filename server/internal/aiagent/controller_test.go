package aiagent

import (
	"errors"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func TestControllerDecideReturnsDecision(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "speak", Speech: "我想一下。"},
	}
	controller := NewController("uno", provider, time.Second)
	defer controller.Close()
	unlocked := false
	locked := false

	decision, err := controller.Decide(DecisionRequest{
		RoomID:        "room_1",
		PlayerID:      "ai_1",
		RequestPrefix: "uno",
		SessionID:     "uno:room_1:ai_1",
		Phase:         "playing",
		Type:          gameactor.AgentOptionalSpeech,
		Profile:       Profile{Name: "北风"},
		Actions:       []aiplayer.LegalAction{{ID: "speak", Label: "发言"}},
		Unlock: func() {
			unlocked = true
		},
		Lock: func() {
			locked = true
		},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if decision.ActionID != "speak" || decision.Speech != "我想一下。" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if !unlocked || !locked {
		t.Fatalf("expected unlock and lock callbacks, unlocked=%t locked=%t", unlocked, locked)
	}
}

func TestControllerDecideReturnsStaleError(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "play", Speech: "出这张。"},
	}
	controller := NewController("uno", provider, time.Second)
	defer controller.Close()
	staleErr := errors.New("ai_agent_decision_stale")

	_, err := controller.Decide(DecisionRequest{
		RoomID:        "room_1",
		PlayerID:      "ai_1",
		RequestPrefix: "uno",
		SessionID:     "uno:room_1:ai_1",
		Phase:         "playing",
		Type:          gameactor.AgentRequiredAction,
		Profile:       Profile{Name: "北风"},
		Actions:       []aiplayer.LegalAction{{ID: "play", Label: "出牌"}},
		Stale: func(decision aiplayer.Decision) error {
			if decision.ActionID != "play" {
				t.Fatalf("expected stale hook decision, got %+v", decision)
			}
			return staleErr
		},
	})
	if !errors.Is(err, staleErr) {
		t.Fatalf("expected stale error, got %v", err)
	}
}

func TestControllerDecideAfterCloseFailsFast(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "speak", Speech: "我想一下。"},
	}
	controller := NewController("uno", provider, time.Second)
	controller.Close()
	startedAt := time.Now()

	_, err := controller.Decide(DecisionRequest{
		RoomID:        "room_1",
		PlayerID:      "ai_1",
		RequestPrefix: "uno",
		SessionID:     "uno:room_1:ai_1",
		Phase:         "playing",
		Type:          gameactor.AgentOptionalSpeech,
		Profile:       Profile{Name: "北风"},
		Actions:       []aiplayer.LegalAction{{ID: "speak", Label: "发言"}},
	})
	if !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("expected closed controller error, got %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("expected fail-fast after close, took %s", elapsed)
	}
}
