package aiagent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type Controller struct {
	game     string
	provider aiplayer.Provider
	ctx      context.Context
	cancel   context.CancelFunc
	registry *Registry
	broker   *RequestBroker
	timeout  time.Duration
	mu       sync.RWMutex
	closed   bool
}

type DecisionRequest struct {
	RoomID        string
	PlayerID      string
	RequestPrefix string
	SessionID     string
	Phase         string
	Type          gameactor.AgentEventType
	Profile       Profile
	State         map[string]any
	Actions       []aiplayer.LegalAction
	Unlock        func()
	Lock          func()
	Stale         func(aiplayer.Decision) error
}

func NewController(game string, provider aiplayer.Provider, timeout time.Duration) *Controller {
	if timeout <= 0 {
		timeout = aiplayer.DecisionTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		game:     game,
		provider: provider,
		ctx:      ctx,
		cancel:   cancel,
		registry: NewRegistry(nil),
		broker:   NewRequestBroker(timeout),
		timeout:  timeout,
	}
}

func (c *Controller) Enabled() bool {
	if c == nil || c.provider == nil || !c.provider.Enabled() {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed
}

func (c *Controller) Decide(request DecisionRequest) (aiplayer.Decision, error) {
	if !c.Enabled() {
		return aiplayer.Decision{}, ErrLLMNotConfigured
	}
	requestID := request.RequestPrefix + "_" + requestIDToken()
	responseCh, errorCh := c.broker.Register(requestID)
	agent, err := c.registry.EnsureWith(c.ctx, request.RoomID, request.PlayerID, func(string, string) *Agent {
		runner := &Runner{
			Game:      c.game,
			Level:     aiplayer.LevelLLM,
			SessionID: request.SessionID,
			Profile:   request.Profile,
			Provider:  c.provider,
			Build: func(event gameactor.AgentEvent, memory Memory) map[string]any {
				state, _ := event.PublicState.(map[string]any)
				if state == nil {
					state = map[string]any{}
				}
				state["aiAgentMemory"] = append([]string{}, memory.Events...)
				return state
			},
		}
		agent := NewAgent(runner, 8, c.broker.IntentHandler)
		agent.SetTimeout(c.timeout)
		agent.SetErrorHandler(c.broker.ErrorHandler)
		return agent
	})
	if err != nil {
		c.broker.Unregister(requestID)
		return aiplayer.Decision{}, err
	}
	event := gameactor.AgentEvent{
		ID:           requestID,
		RoomID:       request.RoomID,
		PlayerID:     request.PlayerID,
		Type:         request.Type,
		Phase:        request.Phase,
		PublicState:  request.State,
		LegalActions: request.Actions,
		Deadline:     time.Now().Add(c.timeout),
	}

	if request.Unlock != nil {
		request.Unlock()
	}
	if err := agent.Submit(context.Background(), event); err != nil {
		c.broker.Unregister(requestID)
		if request.Lock != nil {
			request.Lock()
		}
		return aiplayer.Decision{}, err
	}
	decision, err := c.broker.WaitDecision(requestID, responseCh, errorCh)
	if request.Lock != nil {
		request.Lock()
	}
	if err != nil {
		return aiplayer.Decision{}, err
	}
	if request.Stale != nil {
		if err := request.Stale(decision); err != nil {
			return decision, err
		}
	}
	return decision, nil
}

func (c *Controller) Remove(roomID string, playerID string) {
	if c != nil && c.registry != nil {
		c.registry.Remove(roomID, playerID)
	}
}

func (c *Controller) RemoveRoom(roomID string) {
	if c != nil && c.registry != nil {
		c.registry.RemoveRoom(roomID)
	}
}

func (c *Controller) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
}

func requestIDToken() string {
	var bytes [5]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}
