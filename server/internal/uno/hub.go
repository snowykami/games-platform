package uno

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/snowykami/games-platform/server/internal/gameactor"
	"github.com/snowykami/games-platform/server/internal/wsx"
)

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const websocketWriteTimeout = 2 * time.Second

type Subscriber struct {
	roomID string
	userID string
	client *wsx.Client
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
		720*time.Millisecond,
		900*time.Millisecond,
		func(roomID string) (gameactor.AIActionResult, error) {
			room, shouldContinue, err := manager.RunAIAction(roomID)
			return gameactor.AIActionResult{RoomID: room.ID, Continue: shouldContinue}, err
		},
		func(roomID string) (gameactor.AIOptionalSpeechResult, error) {
			room, changed, err := manager.RunAIOptionalSpeech(roomID)
			return gameactor.AIOptionalSpeechResult{RoomID: room.ID, Changed: changed}, err
		},
		hub.Broadcast,
	)
	go hub.runScheduler()
	return hub
}

func (h *Hub) Subscribe(ctx context.Context, roomID string, userID string, conn *websocket.Conn) {
	client := wsx.NewClient(ctx, conn, websocketWriteTimeout, 32)
	sub := &Subscriber{roomID: roomID, userID: userID, client: client}

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
		sub.client.Close(websocket.StatusNormalClosure, "bye")
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var message wsMessage
		if err := json.Unmarshal(data, &message); err != nil {
			writeWSError(sub.client, "invalid message")
			continue
		}
		if message.Type == "ping" {
			writeWSPong(sub.client, message.Payload)
			continue
		}

		if err := h.handleMessage(message, roomID, userID); err != nil {
			writeWSError(sub.client, err.Error())
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
		if !sub.client.SendJSON(map[string]any{
			"type": "room.state",
			"room": room,
		}) {
			h.dropSubscriber(sub)
		}
	}
}

func (h *Hub) dropSubscriber(sub *Subscriber) {
	h.mu.Lock()
	delete(h.subscribers, sub)
	h.mu.Unlock()
	sub.client.Close(websocket.StatusPolicyViolation, "write failed")
}

func (h *Hub) CloseRoom(roomID string) {
	h.mu.Lock()
	subscribers := make([]*Subscriber, 0, len(h.subscribers))
	for sub := range h.subscribers {
		if sub.roomID == roomID {
			subscribers = append(subscribers, sub)
			delete(h.subscribers, sub)
		}
	}
	h.mu.Unlock()

	for _, sub := range subscribers {
		sub.client.Close(websocket.StatusNormalClosure, "room closed")
	}
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
		sub.client.Close(websocket.StatusNormalClosure, "removed from room")
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
	case "room.draw":
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.Draw(roomID, userID)
			return err
		})
	case "room.play":
		var request playRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_play_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.Play(roomID, userID, request.CardID, request.Color)
			return err
		})
	case "room.call_uno":
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.CallUNO(roomID, userID)
			return err
		})
	case "room.catch_uno":
		var request catchUNORequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_catch_uno_payload")
		}
		return h.runMessageCommand(roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
			_, err := h.manager.CatchUNO(roomID, userID, request.TargetID)
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

func (h *Hub) runScheduler() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for now := range ticker.C {
		result := h.manager.Tick(now)
		for _, roomID := range result.DestroyedRoomIDs {
			h.CloseRoom(roomID)
		}
		for _, roomID := range uniqueRoomIDs(result.BroadcastRoomIDs) {
			h.Broadcast(roomID)
		}
		for _, roomID := range uniqueRoomIDs(result.ScheduleAIRoomIDs) {
			h.ScheduleAIAction(roomID)
		}
	}
}

func uniqueRoomIDs(roomIDs []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		if _, ok := seen[roomID]; ok {
			continue
		}
		seen[roomID] = struct{}{}
		unique = append(unique, roomID)
	}
	return unique
}

func writeWSError(client *wsx.Client, message string) {
	client.SendError(message)
}

func writeWSPong(client *wsx.Client, payload json.RawMessage) {
	client.SendPong(payload)
}
