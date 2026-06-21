package aiagent

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type RequestBroker struct {
	timeout time.Duration

	mu        sync.Mutex
	responses map[string]chan gameactor.PlayerIntent
	errors    map[string]chan error
}

func NewRequestBroker(timeout time.Duration) *RequestBroker {
	if timeout <= 0 {
		timeout = aiplayer.DecisionTimeout
	}
	return &RequestBroker{
		timeout:   timeout,
		responses: map[string]chan gameactor.PlayerIntent{},
		errors:    map[string]chan error{},
	}
}

func (b *RequestBroker) Register(requestID string) (<-chan gameactor.PlayerIntent, <-chan error) {
	responseCh := make(chan gameactor.PlayerIntent, 1)
	errorCh := make(chan error, 1)
	b.mu.Lock()
	b.responses[requestID] = responseCh
	b.errors[requestID] = errorCh
	b.mu.Unlock()
	return responseCh, errorCh
}

func (b *RequestBroker) Unregister(requestID string) {
	b.mu.Lock()
	delete(b.responses, requestID)
	delete(b.errors, requestID)
	b.mu.Unlock()
}

func (b *RequestBroker) WaitDecision(requestID string, responseCh <-chan gameactor.PlayerIntent, errorCh <-chan error) (aiplayer.Decision, error) {
	defer b.Unregister(requestID)
	timer := time.NewTimer(b.timeout + time.Second)
	defer timer.Stop()
	select {
	case intent := <-responseCh:
		return aiplayer.Decision{
			ActionID: intent.ActionID,
			Reason:   intent.Reason,
			Speech:   intent.Speech,
			Thinking: intent.Thinking,
			Notes:    intent.Notes,
			Source:   "llm",
		}, nil
	case err := <-errorCh:
		return aiplayer.Decision{}, err
	case <-timer.C:
		return aiplayer.Decision{}, errors.New("ai_agent_decision_timeout")
	}
}

func (b *RequestBroker) IntentHandler(_ context.Context, intent gameactor.PlayerIntent) error {
	b.mu.Lock()
	responseCh := b.responses[intent.RequestID]
	b.mu.Unlock()
	if responseCh == nil {
		return nil
	}
	select {
	case responseCh <- intent:
	default:
	}
	return nil
}

func (b *RequestBroker) ErrorHandler(event gameactor.AgentEvent, err error) {
	b.mu.Lock()
	errorCh := b.errors[event.ID]
	b.mu.Unlock()
	if errorCh == nil {
		return
	}
	select {
	case errorCh <- err:
	default:
	}
}

func (b *RequestBroker) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.responses)
}
