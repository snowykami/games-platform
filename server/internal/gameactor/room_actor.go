package gameactor

import (
	"context"
	"errors"
	"sync"
)

var ErrActorClosed = errors.New("room_actor_closed")
var ErrRoomIDRequired = errors.New("room_id_required")

type EventHandler func(context.Context, RoomEvent) error
type EventSubscriber func(RoomEvent)

type RoomActor struct {
	roomID string
	inbox  chan RoomEvent

	handler EventHandler

	mu          sync.RWMutex
	subscribers []EventSubscriber
	closed      bool
	done        chan struct{}
}

func NewRoomActor(roomID string, buffer int, handler EventHandler) *RoomActor {
	if buffer <= 0 {
		buffer = 32
	}
	return &RoomActor{
		roomID:  roomID,
		inbox:   make(chan RoomEvent, buffer),
		handler: handler,
		done:    make(chan struct{}),
	}
}

func (a *RoomActor) RoomID() string {
	return a.roomID
}

func (a *RoomActor) Subscribe(subscriber EventSubscriber) {
	if subscriber == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return
	}
	a.subscribers = append(a.subscribers, subscriber)
}

func (a *RoomActor) Submit(ctx context.Context, event RoomEvent) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return ErrActorClosed
	}
	if event.RoomID == "" {
		event.RoomID = a.roomID
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.done:
		return ErrActorClosed
	case a.inbox <- event:
		return nil
	}
}

func (a *RoomActor) Run(ctx context.Context) error {
	defer a.closeDone()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if a.handler != nil {
				if err := a.handler(ctx, event); err != nil {
					return err
				}
			}
			a.publish(event)
			if event.Type == EventRoomClosed {
				return nil
			}
		}
	}
}

func (a *RoomActor) Close() {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	close(a.inbox)
	a.mu.Unlock()
}

func (a *RoomActor) publish(event RoomEvent) {
	a.mu.RLock()
	subscribers := append([]EventSubscriber{}, a.subscribers...)
	a.mu.RUnlock()
	for _, subscriber := range subscribers {
		subscriber(event)
	}
}

func (a *RoomActor) closeDone() {
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
