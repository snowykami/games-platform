package xiangqi

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

const websocketWriteTimeout = 2 * time.Second

type Subscriber struct {
	roomID string
	userID string
	conn   *websocket.Conn
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
		560*time.Millisecond,
		900*time.Millisecond,
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
			return gameactor.AIOptionalSpeechResult{RoomID: room.ID, Changed: changed}, err
		},
		hub.Broadcast,
	)
	return hub
}

func (h *Hub) Subscribe(ctx context.Context, roomID string, userID string, conn *websocket.Conn) {
	sub := &Subscriber{roomID: roomID, userID: userID, conn: conn}

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
		room, err := h.manager.Public(roomID, sub.userID)
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
		targetUserID := ""
		if current, err := h.manager.Public(roomID, userID); err == nil {
			targetUserID = playerUserID(current.Players, request.PlayerID)
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.RemovePlayer(roomID, userID, request.PlayerID)
			if err == nil {
				h.CloseUser(roomID, targetUserID)
			}
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
	case "room.rename":
		var request nameRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_name_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneView, func() error {
			_, err := h.manager.RenamePlayer(roomID, userID, request.Name)
			return err
		})
	case "room.start":
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.Start(roomID, userID)
			return err
		})
	case "room.move":
		var request moveRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_move_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.Move(roomID, userID, request.PieceID, Position{X: request.X, Y: request.Y})
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
