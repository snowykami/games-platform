package socialdeduction

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const (
	websocketWriteTimeout           = 2 * time.Second
	socialAIActionDelay             = 620 * time.Millisecond
	socialAIOptionalSpeechDelay     = 650 * time.Millisecond
	maxSocialAIOptionalSpeechStreak = 2
)

type Subscriber struct {
	roomID           string
	userID           string
	godViewAvailable bool
	godView          bool
	conn             *websocket.Conn
}

type Hub struct {
	manager     *Manager
	mu          sync.Mutex
	subscribers map[*Subscriber]struct{}
	aiScheduler *gameactor.RoomAIScheduler
}

func NewHub(manager *Manager) *Hub {
	hub := &Hub{
		manager:     manager,
		subscribers: map[*Subscriber]struct{}{},
	}
	hub.aiScheduler = gameactor.NewRoomAIScheduler(
		socialAIActionDelay,
		socialAIOptionalSpeechDelay,
		func(roomID string) (gameactor.AIActionResult, error) {
			var room PublicRoom
			shouldContinue := false
			err := manager.RunRoomCommand(context.Background(), roomID, gameactor.EventAIIntentSubmitted, gameactor.LaneRule, func() error {
				var err error
				room, shouldContinue, err = manager.RunAIAction(roomID)
				return err
			})
			return gameactor.AIActionResult{RoomID: room.ID, Continue: shouldContinue}, err
		},
		func(roomID string) (gameactor.AIOptionalSpeechResult, error) {
			var room PublicRoom
			changed := false
			err := manager.RunRoomCommand(context.Background(), roomID, gameactor.EventPlayerSpeech, gameactor.LaneSpeech, func() error {
				var err error
				room, changed, err = manager.RunAIOptionalSpeech(roomID)
				return err
			})
			if changed && socialPhaseHasRequiredAIAction(room) {
				go hub.ScheduleAIAction(room.ID)
			}
			return gameactor.AIOptionalSpeechResult{
				RoomID:   room.ID,
				Changed:  changed,
				Continue: shouldContinueSocialAIOptionalSpeech(room),
			}, err
		},
		hub.Broadcast,
	)
	return hub
}

func (h *Hub) Subscribe(ctx context.Context, roomID string, userID string, godViewAvailable bool, godView bool, conn *websocket.Conn) {
	sub := &Subscriber{roomID: roomID, userID: userID, godViewAvailable: godViewAvailable, godView: godView, conn: conn}

	h.mu.Lock()
	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()

	h.Broadcast(roomID)
	defer func() {
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
		_ = h.manager.RunRoomCommand(context.Background(), roomID, gameactor.EventPlayerDisconnected, gameactor.LanePresence, func() error {
			h.manager.Leave(roomID, userID)
			return nil
		})
		h.Broadcast(roomID)
		conn.Close(websocket.StatusNormalClosure, "bye")
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var message wsMessage
		if err := json.Unmarshal(data, &message); err != nil {
			writeWSError(ctx, conn, "invalid message")
			continue
		}
		if err := h.handleMessage(message, roomID, userID); err != nil {
			writeWSError(ctx, conn, err.Error())
			continue
		}
		h.Broadcast(roomID)
		h.ScheduleAIAction(roomID)
		h.ScheduleAIOptionalSpeech(roomID)
	}
}

func (h *Hub) Broadcast(roomID string) {
	h.mu.Lock()
	subscribers := make([]*Subscriber, 0, len(h.subscribers))
	for sub := range h.subscribers {
		if sub.roomID == roomID {
			subscribers = append(subscribers, sub)
		}
	}
	h.mu.Unlock()

	for _, sub := range subscribers {
		room, err := h.manager.PublicWithOptions(roomID, sub.userID, PublicRoomOptions{
			GodViewAvailable: sub.godViewAvailable,
			GodView:          sub.godView,
		})
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), websocketWriteTimeout)
		err = sub.conn.Write(ctx, websocket.MessageText, mustMarshal(map[string]any{
			"type": "room.state",
			"room": room,
		}))
		cancel()
		if err != nil {
			h.dropSubscriber(sub)
		}
	}
}

func (h *Hub) dropSubscriber(sub *Subscriber) {
	h.mu.Lock()
	delete(h.subscribers, sub)
	h.mu.Unlock()
	_ = sub.conn.Close(websocket.StatusPolicyViolation, "write failed")
}

func (h *Hub) CloseUser(roomID string, userID string) {
	if userID == "" {
		return
	}
	h.mu.Lock()
	subscribers := make([]*Subscriber, 0, len(h.subscribers))
	for sub := range h.subscribers {
		if sub.roomID == roomID && sub.userID == userID {
			subscribers = append(subscribers, sub)
			delete(h.subscribers, sub)
		}
	}
	h.mu.Unlock()
	for _, sub := range subscribers {
		_ = sub.conn.Close(websocket.StatusNormalClosure, "removed from room")
	}
}

func (h *Hub) handleMessage(message wsMessage, roomID string, userID string) error {
	switch message.Type {
	case "room.add_ai":
		var request addAIRequest
		if len(message.Payload) > 0 {
			if err := json.Unmarshal(message.Payload, &request); err != nil {
				return errors.New("invalid_ai_payload")
			}
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.AddAI(roomID, userID, AIOptions{Level: request.Level})
			return err
		})
	case "room.update_ai":
		var request updateAIRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_ai_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.UpdateAI(roomID, userID, request.PlayerID, AIOptions{Level: request.Level})
			return err
		})
	case "room.remove_player":
		var request updateAIRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_player_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.RemovePlayer(roomID, userID, request.PlayerID)
			return err
		})
	case "room.speech":
		var request speechRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_speech_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventPlayerSpeech, gameactor.LaneSpeech, func() error {
			_, err := h.manager.Say(roomID, userID, request.Text)
			return err
		})
	case "room.werewolf_roles":
		var request werewolfRoleConfigRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_role_config_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.UpdateWerewolfRoles(roomID, userID, request.Config)
			return err
		})
	default:
		return errors.New("unknown_message_type")
	}
}

func (h *Hub) runMessageCommand(roomID string, eventType gameactor.RoomEventType, lane gameactor.EventLane, run func() error) error {
	return h.manager.RunRoomCommand(context.Background(), roomID, eventType, lane, run)
}

func (h *Hub) ScheduleAIAction(roomID string) {
	h.aiScheduler.ScheduleAction(roomID)
}

func (h *Hub) ScheduleAIOptionalSpeech(roomID string) {
	h.aiScheduler.ScheduleSpeech(roomID)
}

func shouldContinueSocialAIOptionalSpeech(room PublicRoom) bool {
	if len(room.Speeches) == 0 {
		return werewolfDayHasPendingAISpeaker(room)
	}
	if werewolfDayHasPendingAISpeaker(room) {
		return true
	}
	aiPlayers := map[string]struct{}{}
	for _, player := range room.Players {
		if player.IsAI && player.Alive {
			aiPlayers[player.ID] = struct{}{}
		}
	}
	streak := 0
	for i := len(room.Speeches) - 1; i >= 0; i-- {
		if _, ok := aiPlayers[room.Speeches[i].PlayerID]; !ok {
			break
		}
		streak++
	}
	return streak > 0 && streak < maxSocialAIOptionalSpeechStreak
}

func socialPhaseHasRequiredAIAction(room PublicRoom) bool {
	switch room.Phase {
	case PhaseWerewolfNight, PhaseWerewolfVote, PhaseWerewolfHunter, PhaseAvalonTeam, PhaseAvalonVote, PhaseAvalonQuest, PhaseAssassination, PhaseUndercoverDescribe, PhaseUndercoverVote:
		return true
	default:
		return false
	}
}

func werewolfDayHasPendingAISpeaker(room PublicRoom) bool {
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay {
		return false
	}
	for _, player := range room.Players {
		if player.IsAI && player.Alive && !room.Werewolf.RevealedIdiots[player.ID] && !room.Werewolf.DaySpeakers[player.ID] {
			return true
		}
	}
	return false
}

func writeWSError(ctx context.Context, conn *websocket.Conn, message string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = conn.Write(ctx, websocket.MessageText, mustMarshal(map[string]string{
		"type":  "error",
		"error": message,
	}))
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
