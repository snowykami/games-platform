package gameactor

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRoomActorProcessesEventsSeriallyAndPublishes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled := []RoomEventType{}
	published := []RoomEventType{}
	actor := NewRoomActor("room_1", 4, func(_ context.Context, event RoomEvent) error {
		handled = append(handled, event.Type)
		return nil
	})
	actor.Subscribe(func(event RoomEvent) {
		published = append(published, event.Type)
	})

	done := make(chan error, 1)
	go func() {
		done <- actor.Run(ctx)
	}()

	if err := actor.Submit(ctx, RoomEvent{Type: EventPlayerSpeech}); err != nil {
		t.Fatalf("submit speech: %v", err)
	}
	if err := actor.Submit(ctx, RoomEvent{Type: EventAIIntentSubmitted}); err != nil {
		t.Fatalf("submit intent: %v", err)
	}
	if err := actor.Submit(ctx, RoomEvent{Type: EventRoomClosed}); err != nil {
		t.Fatalf("submit close: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("actor did not stop")
	}

	expected := []RoomEventType{EventPlayerSpeech, EventAIIntentSubmitted, EventRoomClosed}
	if !reflect.DeepEqual(handled, expected) {
		t.Fatalf("handled order mismatch: got %v want %v", handled, expected)
	}
	if !reflect.DeepEqual(published, expected) {
		t.Fatalf("published order mismatch: got %v want %v", published, expected)
	}
}

func TestRoomActorRejectsSubmitAfterClose(t *testing.T) {
	actor := NewRoomActor("room_1", 1, nil)
	actor.Close()
	err := actor.Submit(context.Background(), RoomEvent{Type: EventPlayerSpeech})
	if !errors.Is(err, ErrActorClosed) {
		t.Fatalf("expected ErrActorClosed, got %v", err)
	}
}
