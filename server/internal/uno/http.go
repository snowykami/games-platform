package uno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/snowykami/games-platform/server/internal/auth"
	"github.com/snowykami/games-platform/server/internal/httpx"
)

type Handler struct {
	manager *Manager
	hub     *Hub
}

type playRequest struct {
	CardID string `json:"cardId"`
	Color  Color  `json:"color"`
}

type createRoomRequest struct {
	VariantKey string `json:"variantKey"`
	ThemeKey   string `json:"themeKey"`
}

type addAIRequest struct {
	Level string `json:"level"`
}

type updateAIRequest struct {
	Level    string `json:"level"`
	PlayerID string `json:"playerId,omitempty"`
}

type speechRequest struct {
	Text string `json:"text"`
}

type catchUNORequest struct {
	TargetID string `json:"targetId"`
}

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager, hub: NewHub(manager)}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()
	router.Post("/rooms", h.createRoom)
	router.Get("/rooms/{roomID}", h.getRoom)
	router.Post("/rooms/{roomID}/join", h.joinRoom)
	router.Post("/rooms/{roomID}/ai", h.addAI)
	router.Patch("/rooms/{roomID}/ai/{playerID}", h.updateAI)
	router.Post("/rooms/{roomID}/speech", h.speech)
	router.Post("/rooms/{roomID}/start", h.start)
	router.Post("/rooms/{roomID}/play", h.play)
	router.Post("/rooms/{roomID}/draw", h.draw)
	router.Post("/rooms/{roomID}/uno", h.callUNO)
	router.Post("/rooms/{roomID}/catch-uno", h.catchUNO)
	return router
}

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.WriteErrorKey(w, r, http.StatusUnauthorized, "login_required")
		return
	}
	if user.Banned {
		httpx.WriteErrorKey(w, r, http.StatusForbidden, "user_banned")
		return
	}

	roomID := r.URL.Query().Get("room")
	if _, err := h.manager.JoinRoom(roomID, toUserView(user)); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	h.hub.Subscribe(r.Context(), roomID, user.ID, conn)
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	var request createRoomRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httpx.DecodeJSON(r, &request); err != nil {
			httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
			return
		}
	}

	room := h.manager.CreateRoom(toUserView(user), RoomOptions{
		VariantKey: request.VariantKey,
		ThemeKey:   request.ThemeKey,
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) getRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, err := h.manager.Public(chi.URLParam(r, "roomID"), user.ID)
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusNotFound, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, err := h.manager.JoinRoom(chi.URLParam(r, "roomID"), toUserView(user))
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.Broadcast(room.ID)
	h.hub.ScheduleAI(room.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) addAI(w http.ResponseWriter, r *http.Request) {
	var request addAIRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httpx.DecodeJSON(r, &request); err != nil {
			httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
			return
		}
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.AddAI(roomID, userID, AIOptions{Level: request.Level})
	})
}

func (h *Handler) updateAI(w http.ResponseWriter, r *http.Request) {
	var request updateAIRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UpdateAI(roomID, userID, chi.URLParam(r, "playerID"), AIOptions{Level: request.Level})
	})
}

func (h *Handler) speech(w http.ResponseWriter, r *http.Request) {
	var request speechRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Say(roomID, userID, request.Text)
	})
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Start(roomID, userID)
	})
}

func (h *Handler) draw(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Draw(roomID, userID)
	})
}

func (h *Handler) play(w http.ResponseWriter, r *http.Request) {
	var request playRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Play(roomID, userID, request.CardID, request.Color)
	})
}

func (h *Handler) callUNO(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.CallUNO(roomID, userID)
	})
}

func (h *Handler) catchUNO(w http.ResponseWriter, r *http.Request) {
	var request catchUNORequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.CatchUNO(roomID, userID, request.TargetID)
	})
}

func (h *Handler) mutateRoom(w http.ResponseWriter, r *http.Request, mutate func(roomID string, userID string) (PublicRoom, error)) {
	user := mustUser(r)
	room, err := mutate(chi.URLParam(r, "roomID"), user.ID)
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.Broadcast(room.ID)
	h.hub.ScheduleAI(room.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func mustUser(r *http.Request) *auth.User {
	user, _ := auth.UserFromContext(r.Context())
	return user
}

func toUserView(user *auth.User) UserView {
	return UserView{ID: user.ID, DisplayName: user.DisplayName, Role: string(user.Role), Kind: string(user.Kind)}
}

type Subscriber struct {
	roomID string
	userID string
	conn   *websocket.Conn
}

type Hub struct {
	manager     *Manager
	mu          sync.Mutex
	subscribers map[*Subscriber]struct{}
	aiRunning   map[string]struct{}
}

func NewHub(manager *Manager) *Hub {
	return &Hub{
		manager:     manager,
		subscribers: map[*Subscriber]struct{}{},
		aiRunning:   map[string]struct{}{},
	}
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
		h.manager.Leave(roomID, userID)
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
		h.ScheduleAI(roomID)
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
		_ = sub.conn.Write(context.Background(), websocket.MessageText, mustMarshal(map[string]any{
			"type": "room.state",
			"room": room,
		}))
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
		_, err := h.manager.AddAI(roomID, userID, AIOptions{Level: request.Level})
		return err
	case "room.update_ai":
		var request updateAIRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_ai_payload")
		}
		_, err := h.manager.UpdateAI(roomID, userID, request.PlayerID, AIOptions{Level: request.Level})
		return err
	case "room.speech":
		var request speechRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_speech_payload")
		}
		_, err := h.manager.Say(roomID, userID, request.Text)
		return err
	case "room.start":
		_, err := h.manager.Start(roomID, userID)
		return err
	case "room.draw":
		_, err := h.manager.Draw(roomID, userID)
		return err
	case "room.play":
		var request playRequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_play_payload")
		}
		_, err := h.manager.Play(roomID, userID, request.CardID, request.Color)
		return err
	case "room.call_uno":
		_, err := h.manager.CallUNO(roomID, userID)
		return err
	case "room.catch_uno":
		var request catchUNORequest
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			return errors.New("invalid_catch_uno_payload")
		}
		_, err := h.manager.CatchUNO(roomID, userID, request.TargetID)
		return err
	default:
		return errors.New("unknown_message_type")
	}
}

func (h *Hub) ScheduleAI(roomID string) {
	h.mu.Lock()
	if _, ok := h.aiRunning[roomID]; ok {
		h.mu.Unlock()
		return
	}
	h.aiRunning[roomID] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.aiRunning, roomID)
			h.mu.Unlock()
		}()

		for {
			time.Sleep(720 * time.Millisecond)
			room, shouldContinue, err := h.manager.RunNextAI(roomID)
			if err != nil {
				return
			}
			if room.ID != "" {
				h.Broadcast(room.ID)
			}
			if !shouldContinue {
				return
			}
		}
	}()
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
