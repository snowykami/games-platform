package aiagent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func TestRequestBrokerWaitsForDecisionAndCleansUp(t *testing.T) {
	broker := NewRequestBroker(time.Second)
	responseCh, errorCh := broker.Register("req_1")

	go func() {
		_ = broker.IntentHandler(context.Background(), gameactor.PlayerIntent{
			RequestID: "req_1",
			ActionID:  "play:card_1",
			Speech:    "红色继续。",
		})
	}()

	decision, err := broker.WaitDecision("req_1", responseCh, errorCh)
	if err != nil {
		t.Fatalf("wait decision: %v", err)
	}
	if decision.ActionID != "play:card_1" || decision.Speech != "红色继续。" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if broker.Count() != 0 {
		t.Fatalf("expected cleanup, got %d pending requests", broker.Count())
	}
}

func TestRequestBrokerWaitsForErrorAndCleansUp(t *testing.T) {
	broker := NewRequestBroker(time.Second)
	responseCh, errorCh := broker.Register("req_1")
	expected := errors.New("llm_failed")

	go broker.ErrorHandler(gameactor.AgentEvent{ID: "req_1"}, expected)

	_, err := broker.WaitDecision("req_1", responseCh, errorCh)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
	if broker.Count() != 0 {
		t.Fatalf("expected cleanup, got %d pending requests", broker.Count())
	}
}

func TestRequestBrokerTimeoutCleansUp(t *testing.T) {
	broker := NewRequestBroker(time.Millisecond)
	responseCh, errorCh := broker.Register("req_1")

	_, err := broker.WaitDecision("req_1", responseCh, errorCh)
	if err == nil || err.Error() != "ai_agent_decision_timeout" {
		t.Fatalf("expected timeout, got %v", err)
	}
	if broker.Count() != 0 {
		t.Fatalf("expected cleanup, got %d pending requests", broker.Count())
	}
}
