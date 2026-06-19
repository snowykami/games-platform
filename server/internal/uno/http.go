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
	router.Post("/rooms/{roomID}/start", h.start)
	router.Post("/rooms/{roomID}/play", h.play)
	router.Post("/rooms/{roomID}/draw", h.draw)
	return router
}

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "login required")
		return
	}
	if user.Banned {
		httpx.WriteError(w, http.StatusForbidden, "user is banned")
		return
	}

	roomID := r.URL.Query().Get("room")
	if _, err := h.manager.JoinRoom(roomID, toUserView(user)); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
			httpx.WriteError(w, http.StatusBadRequest, "invalid json body")
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
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, err := h.manager.JoinRoom(chi.URLParam(r, "roomID"), toUserView(user))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.Broadcast(room.ID)
	h.hub.ScheduleAI(room.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) addAI(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.AddAI(roomID, userID)
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
		httpx.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Play(roomID, userID, request.CardID, request.Color)
	})
}

func (h *Handler) mutateRoom(w http.ResponseWriter, r *http.Request, mutate func(roomID string, userID string) (PublicRoom, error)) {
	user := mustUser(r)
	room, err := mutate(chi.URLParam(r, "roomID"), user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
		_, err := h.manager.AddAI(roomID, userID)
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
			return errors.New("invalid play payload")
		}
		_, err := h.manager.Play(roomID, userID, request.CardID, request.Color)
		return err
	default:
		return errors.New("unknown message type")
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
