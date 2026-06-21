package aiagent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type fakeProvider struct {
	enabled  bool
	decision aiplayer.Decision
	input    aiplayer.DecisionInput
}

func (p *fakeProvider) Enabled() bool {
	return p.enabled
}

func (p *fakeProvider) Decide(_ context.Context, input aiplayer.DecisionInput) (aiplayer.Decision, error) {
	p.input = input
	return p.decision, nil
}

func TestRunnerBuildsDecisionInputAndReturnsIntent(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "vote:seat_2", Reason: "先看二号", Speech: "我先挂二号。"},
	}
	runner := Runner{
		Game:      "werewolf",
		Level:     aiplayer.LevelLLM,
		SessionID: "session-1",
		Profile:   Profile{Name: "白川", Personality: "谨慎"},
		Provider:  provider,
		Build: func(event gameactor.AgentEvent, memory Memory) map[string]any {
			return map[string]any{"phase": event.Phase, "memory": memory.Events}
		},
	}
	event := gameactor.AgentEvent{
		ID:           "req_1",
		RoomID:       "room_1",
		PlayerID:     "ai_1",
		Type:         gameactor.AgentRequiredAction,
		Phase:        "vote",
		LegalActions: []aiplayer.LegalAction{{ID: "vote:seat_2", Label: "投票给座位2"}},
	}

	intent, err := runner.Decide(context.Background(), event)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if intent.Kind != gameactor.IntentRequiredAction || intent.ActionID != "vote:seat_2" || intent.Speech != "我先挂二号。" {
		t.Fatalf("unexpected intent: %+v", intent)
	}
	if provider.input.Game != "werewolf" || provider.input.SessionID != "session-1" || provider.input.PlayerName != "白川" {
		t.Fatalf("unexpected provider input: %+v", provider.input)
	}
	if len(runner.Memory.Events) != 1 {
		t.Fatalf("expected memory event, got %+v", runner.Memory.Events)
	}
}

func TestRunnerRejectsIllegalAction(t *testing.T) {
	provider := &fakeProvider{enabled: true, decision: aiplayer.Decision{ActionID: "vote:seat_9"}}
	runner := Runner{Game: "werewolf", Level: aiplayer.LevelLLM, Provider: provider}
	event := gameactor.AgentEvent{
		ID:           "req_1",
		Type:         gameactor.AgentRequiredAction,
		LegalActions: []aiplayer.LegalAction{{ID: "vote:seat_2", Label: "投票给座位2"}},
	}

	if _, err := runner.Decide(context.Background(), event); err == nil || err.Error() != "illegal_action" {
		t.Fatalf("expected illegal_action, got %v", err)
	}
}

func TestAgentProcessesEventsAndReturnsIntent(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "speak", Speech: "先说到这。"},
	}
	runner := &Runner{
		Game:      "undercover",
		Level:     aiplayer.LevelLLM,
		SessionID: "session-1",
		Profile:   Profile{Name: "北风"},
		Provider:  provider,
	}
	intents := make(chan gameactor.PlayerIntent, 1)
	agent := NewAgent(runner, 1, func(_ context.Context, intent gameactor.PlayerIntent) error {
		intents <- intent
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = agent.Run(ctx)
	}()

	event := gameactor.AgentEvent{
		ID:           "req_1",
		RoomID:       "room_1",
		PlayerID:     "ai_1",
		Type:         gameactor.AgentOptionalSpeech,
		Phase:        "describe",
		LegalActions: []aiplayer.LegalAction{{ID: "speak", Label: "发言"}},
	}
	if err := agent.Submit(context.Background(), event); err != nil {
		t.Fatalf("submit: %v", err)
	}

	select {
	case intent := <-intents:
		if intent.Kind != gameactor.IntentOptionalSpeech || intent.ActionID != "speak" || intent.Speech != "先说到这。" {
			t.Fatalf("unexpected intent: %+v", intent)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for intent")
	}
}

func TestAgentReportsDecisionErrors(t *testing.T) {
	provider := &fakeProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "illegal"},
	}
	runner := &Runner{Game: "werewolf", Level: aiplayer.LevelLLM, Provider: provider}
	errs := make(chan error, 1)
	agent := NewAgent(runner, 1, nil)
	agent.SetErrorHandler(func(_ gameactor.AgentEvent, err error) {
		errs <- err
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = agent.Run(ctx)
	}()

	if err := agent.Submit(context.Background(), gameactor.AgentEvent{
		ID:           "req_1",
		Type:         gameactor.AgentRequiredAction,
		LegalActions: []aiplayer.LegalAction{{ID: "vote:seat_1", Label: "投票"}},
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}

	select {
	case err := <-errs:
		if !errors.Is(err, errors.New("illegal_action")) && err.Error() != "illegal_action" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestAgentRejectsSubmitAfterClose(t *testing.T) {
	agent := NewAgent(&Runner{}, 1, nil)
	agent.Close()
	if err := agent.Submit(context.Background(), gameactor.AgentEvent{}); !errors.Is(err, gameactor.ErrActorClosed) {
		t.Fatalf("expected closed error, got %v", err)
	}
}
