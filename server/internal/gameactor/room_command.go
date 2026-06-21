package gameactor

import (
	"context"
	"sync"
	"time"
)

const DefaultRoomCommandTimeout = 35 * time.Second

type RoomCommandRegistry struct {
	buffer int

	mu     sync.Mutex
	actors map[string]*RoomActor
}

type roomCommandPayload struct {
	run  func(context.Context) error
	done chan error
}

func NewRoomCommandRegistry(buffer int) *RoomCommandRegistry {
	if buffer <= 0 {
		buffer = 32
	}
	return &RoomCommandRegistry{
		buffer: buffer,
		actors: map[string]*RoomActor{},
	}
}

func (r *RoomCommandRegistry) Do(ctx context.Context, event RoomEvent, run func(context.Context) error) error {
	if r == nil {
		if run == nil {
			return nil
		}
		return run(ctx)
	}
	if event.RoomID == "" {
		return ErrRoomIDRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil {
		run = func(context.Context) error { return nil }
	}
	actor := r.ensure(event.RoomID)
	done := make(chan error, 1)
	event.Payload = roomCommandPayload{run: run, done: done}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if err := actor.Submit(ctx, event); err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), DefaultRoomCommandTimeout)
	defer cancel()
	select {
	case <-waitCtx.Done():
		return waitCtx.Err()
	case err := <-done:
		if event.Type == EventRoomClosed {
			r.RemoveRoom(event.RoomID)
		}
		return err
	}
}

func (r *RoomCommandRegistry) RemoveRoom(roomID string) {
	r.mu.Lock()
	actor := r.actors[roomID]
	delete(r.actors, roomID)
	r.mu.Unlock()
	if actor != nil {
		actor.Close()
	}
}

func (r *RoomCommandRegistry) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	actors := make([]*RoomActor, 0, len(r.actors))
	for roomID, actor := range r.actors {
		actors = append(actors, actor)
		delete(r.actors, roomID)
	}
	r.mu.Unlock()
	for _, actor := range actors {
		actor.Close()
	}
}

func (r *RoomCommandRegistry) Count() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.actors)
}

func (r *RoomCommandRegistry) ensure(roomID string) *RoomActor {
	r.mu.Lock()
	defer r.mu.Unlock()
	if actor := r.actors[roomID]; actor != nil {
		return actor
	}
	actor := NewRoomActor(roomID, r.buffer, handleRoomCommand)
	r.actors[roomID] = actor
	go func() {
		_ = actor.Run(context.Background())
	}()
	return actor
}

func handleRoomCommand(ctx context.Context, event RoomEvent) error {
	payload, ok := event.Payload.(roomCommandPayload)
	if !ok {
		return nil
	}
	if payload.run == nil {
		payload.done <- nil
		return nil
	}
	payload.done <- payload.run(ctx)
	return nil
}
