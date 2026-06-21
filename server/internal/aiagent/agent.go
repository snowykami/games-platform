package aiagent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

var ErrAgentFactoryMissing = errors.New("ai_agent_factory_missing")
var ErrLLMNotConfigured = errors.New("llm_not_configured")

type Profile struct {
	Name        string
	Personality string
	SpeechStyle string
}

type Memory struct {
	Events []string
	Limit  int
}

func (m *Memory) Remember(event string) {
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}
	limit := m.Limit
	if limit <= 0 {
		limit = 24
	}
	m.Events = append(m.Events, event)
	if len(m.Events) > limit {
		m.Events = m.Events[len(m.Events)-limit:]
	}
}

type ContextBuilder func(event gameactor.AgentEvent, memory Memory) map[string]any

type IntentHandler func(context.Context, gameactor.PlayerIntent) error
type ErrorHandler func(gameactor.AgentEvent, error)

type Runner struct {
	Game      string
	Level     aiplayer.Level
	SessionID string
	Profile   Profile
	Provider  aiplayer.Provider
	Memory    Memory
	Build     ContextBuilder
	Now       func() time.Time
}

func (r *Runner) Decide(ctx context.Context, event gameactor.AgentEvent) (gameactor.PlayerIntent, error) {
	if r.Provider == nil || !r.Provider.Enabled() {
		return gameactor.PlayerIntent{}, ErrLLMNotConfigured
	}
	if len(event.LegalActions) == 0 {
		return gameactor.PlayerIntent{}, errors.New("no_legal_actions")
	}
	state := map[string]any{}
	if r.Build != nil {
		state = r.Build(event, r.Memory)
	}
	input := aiplayer.DecisionInput{
		Game:        r.Game,
		Level:       r.Level,
		SessionID:   r.SessionID,
		PlayerName:  r.Profile.Name,
		Personality: r.Profile.Personality,
		SpeechStyle: r.Profile.SpeechStyle,
		State:       state,
		Actions:     event.LegalActions,
	}
	decision, err := r.Provider.Decide(ctx, input)
	if err != nil {
		return gameactor.PlayerIntent{}, err
	}
	if !aiplayer.ValidateAction(decision.ActionID, event.LegalActions) {
		return gameactor.PlayerIntent{}, errors.New("illegal_action")
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	kind := gameactor.IntentRequiredAction
	if event.Type == gameactor.AgentOptionalSpeech {
		kind = gameactor.IntentOptionalSpeech
	}
	intent := gameactor.PlayerIntent{
		RoomID:    event.RoomID,
		PlayerID:  event.PlayerID,
		RequestID: event.ID,
		Kind:      kind,
		ActionID:  decision.ActionID,
		Speech:    strings.TrimSpace(decision.Speech),
		Reason:    strings.TrimSpace(decision.Reason),
		Notes:     decision.Notes,
		CreatedAt: now(),
	}
	r.Memory.Remember("phase=" + event.Phase + " action=" + intent.ActionID + " reason=" + intent.Reason + " speech=" + intent.Speech)
	return intent, nil
}

type Agent struct {
	runner   *Runner
	inbox    chan gameactor.AgentEvent
	onIntent IntentHandler
	onError  ErrorHandler
	cooldown time.Duration
	timeout  time.Duration

	mu     sync.RWMutex
	closed bool
	done   chan struct{}
}

func NewAgent(runner *Runner, buffer int, onIntent IntentHandler) *Agent {
	if buffer <= 0 {
		buffer = 8
	}
	return &Agent{
		runner:   runner,
		inbox:    make(chan gameactor.AgentEvent, buffer),
		onIntent: onIntent,
		timeout:  aiplayer.DecisionTimeout,
		done:     make(chan struct{}),
	}
}

func (a *Agent) SetErrorHandler(handler ErrorHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onError = handler
}

func (a *Agent) SetCooldown(cooldown time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cooldown = cooldown
}

func (a *Agent) SetTimeout(timeout time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.timeout = timeout
}

func (a *Agent) Submit(ctx context.Context, event gameactor.AgentEvent) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return gameactor.ErrActorClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.done:
		return gameactor.ErrActorClosed
	case a.inbox <- event:
		return nil
	}
}

func (a *Agent) Run(ctx context.Context) error {
	defer a.closeDone()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if event.Type == gameactor.AgentShutdown {
				return nil
			}
			a.handle(ctx, event)
		}
	}
}

func (a *Agent) Close() {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	close(a.inbox)
	a.mu.Unlock()
}

func (a *Agent) handle(ctx context.Context, event gameactor.AgentEvent) {
	runner, onIntent, onError, cooldown, timeout := a.snapshot()
	if runner == nil {
		a.reportError(event, errors.New("ai_runner_not_configured"), onError)
		return
	}
	if cooldown > 0 {
		timer := time.NewTimer(cooldown)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	decisionCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		decisionCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	intent, err := runner.Decide(decisionCtx, event)
	if err != nil {
		a.reportError(event, err, onError)
		return
	}
	if onIntent == nil {
		return
	}
	if err := onIntent(ctx, intent); err != nil {
		a.reportError(event, err, onError)
	}
}

func (a *Agent) snapshot() (*Runner, IntentHandler, ErrorHandler, time.Duration, time.Duration) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.runner, a.onIntent, a.onError, a.cooldown, a.timeout
}

func (a *Agent) reportError(event gameactor.AgentEvent, err error, handler ErrorHandler) {
	if handler != nil {
		handler(event, err)
	}
}

func (a *Agent) closeDone() {
	a.mu.Lock()
	if !a.closed {
		a.closed = true
		close(a.inbox)
	}
	select {
	case <-a.done:
	default:
		close(a.done)
	}
	a.mu.Unlock()
}
