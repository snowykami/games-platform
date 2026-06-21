package gameactor

import (
	"sync"
	"testing"
	"time"
)

func TestRoomAISchedulerRunsActionAndSpeech(t *testing.T) {
	var mu sync.Mutex
	broadcasts := []string{}
	actionCalls := 0
	speechCalls := 0
	done := make(chan struct{})

	scheduler := NewRoomAIScheduler(
		time.Millisecond,
		time.Millisecond,
		func(roomID string) (AIActionResult, error) {
			mu.Lock()
			actionCalls++
			mu.Unlock()
			return AIActionResult{RoomID: roomID, Continue: false}, nil
		},
		func(roomID string) (AIOptionalSpeechResult, error) {
			mu.Lock()
			speechCalls++
			mu.Unlock()
			return AIOptionalSpeechResult{RoomID: roomID, Changed: true}, nil
		},
		func(roomID string) {
			mu.Lock()
			broadcasts = append(broadcasts, roomID)
			if len(broadcasts) >= 2 {
				close(done)
			}
			mu.Unlock()
		},
	)

	scheduler.ScheduleAction("room_1")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not broadcast action and speech")
	}

	mu.Lock()
	defer mu.Unlock()
	if actionCalls != 1 || speechCalls != 1 {
		t.Fatalf("expected one action and one speech call, got action=%d speech=%d", actionCalls, speechCalls)
	}
}

func TestRoomAISchedulerDoesNotRunSpeechWhileActionIsRunning(t *testing.T) {
	actionStarted := make(chan struct{})
	allowActionReturn := make(chan struct{})
	speechCalled := make(chan struct{}, 1)
	scheduler := NewRoomAIScheduler(
		time.Millisecond,
		time.Millisecond,
		func(roomID string) (AIActionResult, error) {
			close(actionStarted)
			<-allowActionReturn
			return AIActionResult{RoomID: roomID, Continue: false}, nil
		},
		func(roomID string) (AIOptionalSpeechResult, error) {
			speechCalled <- struct{}{}
			return AIOptionalSpeechResult{RoomID: roomID, Changed: true}, nil
		},
		func(string) {},
	)

	scheduler.ScheduleAction("room_1")
	<-actionStarted
	scheduler.ScheduleSpeech("room_1")

	select {
	case <-speechCalled:
		t.Fatal("speech ran while action was still running")
	case <-time.After(20 * time.Millisecond):
	}

	close(allowActionReturn)
	select {
	case <-speechCalled:
	case <-time.After(time.Second):
		t.Fatal("speech did not run after action finished")
	}
}
