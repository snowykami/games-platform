package gameactor

import (
	"context"
	"testing"
)

func TestRoomRuntimeRunsCommand(t *testing.T) {
	runtime := NewRoomRuntime(4)
	defer runtime.Close()

	ran := false
	err := runtime.RunRoomCommand(context.Background(), "room_1", EventHumanIntentSubmitted, LaneRule, func() error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("run room command: %v", err)
	}
	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestNilRoomRuntimeRunsInline(t *testing.T) {
	var runtime *RoomRuntime
	ran := false
	err := runtime.RunRoomCommand(context.Background(), "room_1", EventHumanIntentSubmitted, LaneRule, func() error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("run inline command: %v", err)
	}
	if !ran {
		t.Fatal("expected inline command to run")
	}
}
