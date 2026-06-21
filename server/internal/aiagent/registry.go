package aiagent

import (
	"context"
	"sync"

	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type AgentFactory func(roomID string, playerID string) *Agent

type Registry struct {
	factory AgentFactory

	mu     sync.Mutex
	agents map[string]*Agent
}

func NewRegistry(factory AgentFactory) *Registry {
	return &Registry{
		factory: factory,
		agents:  map[string]*Agent{},
	}
}

func (r *Registry) Ensure(ctx context.Context, roomID string, playerID string) (*Agent, error) {
	return r.EnsureWith(ctx, roomID, playerID, r.factory)
}

func (r *Registry) EnsureWith(ctx context.Context, roomID string, playerID string, factory AgentFactory) (*Agent, error) {
	key := agentKey(roomID, playerID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if agent := r.agents[key]; agent != nil {
		return agent, nil
	}
	if factory == nil {
		return nil, ErrAgentFactoryMissing
	}
	agent := factory(roomID, playerID)
	if agent == nil {
		return nil, ErrAgentFactoryMissing
	}
	r.agents[key] = agent
	go func() {
		_ = agent.Run(ctx)
	}()
	return agent, nil
}

func (r *Registry) Submit(ctx context.Context, event gameactor.AgentEvent) error {
	agent, err := r.Ensure(ctx, event.RoomID, event.PlayerID)
	if err != nil {
		return err
	}
	return agent.Submit(ctx, event)
}

func (r *Registry) Remove(roomID string, playerID string) {
	key := agentKey(roomID, playerID)
	r.mu.Lock()
	agent := r.agents[key]
	delete(r.agents, key)
	r.mu.Unlock()
	if agent != nil {
		agent.Close()
	}
}

func (r *Registry) RemoveRoom(roomID string) {
	r.mu.Lock()
	agents := []*Agent{}
	for key, agent := range r.agents {
		if roomIDFromAgentKey(key) == roomID {
			agents = append(agents, agent)
			delete(r.agents, key)
		}
	}
	r.mu.Unlock()
	for _, agent := range agents {
		agent.Close()
	}
}

func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.agents)
}

func agentKey(roomID string, playerID string) string {
	return roomID + "\x00" + playerID
}

func roomIDFromAgentKey(key string) string {
	for index, char := range key {
		if char == '\x00' {
			return key[:index]
		}
	}
	return key
}
