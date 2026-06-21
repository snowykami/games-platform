package gameactor

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRoomCommandRegistryRunsCommandsSerially(t *testing.T) {
	registry := NewRoomCommandRegistry(4)
	defer registry.Close()

	order := []int{}
	errCh := make(chan error, 3)
	for index := 0; index < 3; index++ {
		index := index
		go func() {
			errCh <- registry.Do(context.Background(), RoomEvent{
				RoomID: "room_1",
				Type:   EventHumanIntentSubmitted,
				Lane:   LaneRule,
			}, func(context.Context) error {
				order = append(order, index)
				return nil
			})
		}()
	}

	for index := 0; index < 3; index++ {
		if err := <-errCh; err != nil {
			t.Fatalf("command %d: %v", index, err)
		}
	}
	if len(order) != 3 {
		t.Fatalf("expected three commands, got %v", order)
	}
}

func TestRoomCommandRegistryCommandErrorDoesNotCloseActor(t *testing.T) {
	registry := NewRoomCommandRegistry(4)
	defer registry.Close()
	expectedErr := errors.New("bad_move")

	err := registry.Do(context.Background(), RoomEvent{
		RoomID: "room_1",
		Type:   EventHumanIntentSubmitted,
		Lane:   LaneRule,
	}, func(context.Context) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected command error, got %v", err)
	}

	ran := false
	err = registry.Do(context.Background(), RoomEvent{
		RoomID: "room_1",
		Type:   EventPlayerSpeech,
		Lane:   LaneSpeech,
	}, func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("second command: %v", err)
	}
	if !ran {
		t.Fatal("expected second command to run after previous error")
	}
}

func TestRoomCommandRegistryRemovesClosedRoom(t *testing.T) {
	registry := NewRoomCommandRegistry(4)
	defer registry.Close()

	if err := registry.Do(context.Background(), RoomEvent{
		RoomID: "room_1",
		Type:   EventPlayerSpeech,
		Lane:   LaneSpeech,
	}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("command: %v", err)
	}
	if count := registry.Count(); count != 1 {
		t.Fatalf("expected one actor, got %d", count)
	}
	if err := registry.Do(context.Background(), RoomEvent{
		RoomID: "room_1",
		Type:   EventRoomClosed,
		Lane:   LaneRule,
	}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("close command: %v", err)
	}
	if count := registry.Count(); count != 0 {
		t.Fatalf("expected actor cleanup, got %d", count)
	}
}

func TestRoomCommandRegistryRequiresRoomID(t *testing.T) {
	registry := NewRoomCommandRegistry(4)
	defer registry.Close()

	err := registry.Do(context.Background(), RoomEvent{Type: EventPlayerSpeech}, func(context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrRoomIDRequired) {
		t.Fatalf("expected ErrRoomIDRequired, got %v", err)
	}
}

func TestRoomCommandRegistrySubmitHonorsCanceledContext(t *testing.T) {
	registry := NewRoomCommandRegistry(1)
	defer registry.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := registry.Do(ctx, RoomEvent{
		RoomID: "room_1",
		Type:   EventPlayerSpeech,
		Lane:   LaneSpeech,
	}, func(context.Context) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled before submit, got %v", err)
	}
}

func TestRoomCommandRegistryWaitIgnoresCancellationAfterSubmit(t *testing.T) {
	registry := NewRoomCommandRegistry(1)
	defer registry.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- registry.Do(ctx, RoomEvent{
			RoomID: "room_1",
			Type:   EventPlayerSpeech,
			Lane:   LaneSpeech,
		}, func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()

	<-started
	cancel()
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected command completion after cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("command did not finish")
	}
}

func TestRoomCommandRegistrySingleCallerOrder(t *testing.T) {
	registry := NewRoomCommandRegistry(4)
	defer registry.Close()

	order := []string{}
	for _, item := range []string{"a", "b", "c"} {
		item := item
		if err := registry.Do(context.Background(), RoomEvent{
			RoomID: "room_1",
			Type:   EventHumanIntentSubmitted,
			Lane:   LaneRule,
		}, func(context.Context) error {
			order = append(order, item)
			return nil
		}); err != nil {
			t.Fatalf("command %s: %v", item, err)
		}
	}
	if !reflect.DeepEqual(order, []string{"a", "b", "c"}) {
		t.Fatalf("order mismatch: %v", order)
	}
}
