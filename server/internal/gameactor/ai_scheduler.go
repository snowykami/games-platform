package gameactor

import (
	"sync"
	"time"
)

type AIActionResult struct {
	RoomID   string
	Continue bool
}

type AIOptionalSpeechResult struct {
	RoomID  string
	Changed bool
}

type RoomAIScheduler struct {
	mu            sync.Mutex
	actionRunning map[string]struct{}
	speechRunning map[string]struct{}
	actionDelay   time.Duration
	speechDelay   time.Duration
	runAction     func(string) (AIActionResult, error)
	runSpeech     func(string) (AIOptionalSpeechResult, error)
	broadcast     func(string)
}

func NewRoomAIScheduler(actionDelay time.Duration, speechDelay time.Duration, runAction func(string) (AIActionResult, error), runSpeech func(string) (AIOptionalSpeechResult, error), broadcast func(string)) *RoomAIScheduler {
	return &RoomAIScheduler{
		actionRunning: map[string]struct{}{},
		speechRunning: map[string]struct{}{},
		actionDelay:   actionDelay,
		speechDelay:   speechDelay,
		runAction:     runAction,
		runSpeech:     runSpeech,
		broadcast:     broadcast,
	}
}

func (s *RoomAIScheduler) ScheduleAction(roomID string) {
	if s == nil || s.runAction == nil {
		return
	}
	s.mu.Lock()
	if _, ok := s.actionRunning[roomID]; ok {
		s.mu.Unlock()
		return
	}
	s.actionRunning[roomID] = struct{}{}
	s.mu.Unlock()

	go func() {
		speechRoomID := ""
		defer func() {
			s.mu.Lock()
			delete(s.actionRunning, roomID)
			s.mu.Unlock()
			if speechRoomID != "" {
				s.ScheduleSpeech(speechRoomID)
			}
		}()

		for {
			time.Sleep(s.actionDelay)
			result, err := s.runAction(roomID)
			if err != nil {
				return
			}
			if result.RoomID != "" {
				s.broadcastRoom(result.RoomID)
				speechRoomID = result.RoomID
			}
			if !result.Continue {
				return
			}
		}
	}()
}

func (s *RoomAIScheduler) ScheduleSpeech(roomID string) {
	if s == nil || s.runSpeech == nil {
		return
	}
	s.mu.Lock()
	if _, ok := s.actionRunning[roomID]; ok {
		s.mu.Unlock()
		return
	}
	if _, ok := s.speechRunning[roomID]; ok {
		s.mu.Unlock()
		return
	}
	s.speechRunning[roomID] = struct{}{}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.speechRunning, roomID)
			s.mu.Unlock()
		}()

		time.Sleep(s.speechDelay)
		result, err := s.runSpeech(roomID)
		if err != nil || !result.Changed || result.RoomID == "" {
			return
		}
		s.broadcastRoom(result.RoomID)
	}()
}

func (s *RoomAIScheduler) broadcastRoom(roomID string) {
	if s.broadcast != nil {
		s.broadcast(roomID)
	}
}
