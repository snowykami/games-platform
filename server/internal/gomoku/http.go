package gomoku

import (
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/snowykami/games-platform/server/internal/auth"
	"github.com/snowykami/games-platform/server/internal/gameactor"
	"github.com/snowykami/games-platform/server/internal/httpx"
)

type Handler struct {
	manager *Manager
	hub     *Hub
}

type placeRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
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

type nameRequest struct {
	Name string `json:"name"`
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager, hub: NewHub(manager)}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()
	router.Post("/rooms", h.createRoom)
	router.Get("/rooms/current", h.currentRoom)
	router.Get("/rooms/{roomID}", h.getRoom)
	router.Post("/rooms/{roomID}/join", h.joinRoom)
	router.Post("/rooms/{roomID}/ai", h.addAI)
	router.Patch("/rooms/{roomID}/ai/{playerID}", h.updateAI)
	router.Delete("/rooms/{roomID}/players/{playerID}", h.removePlayer)
	router.Post("/rooms/{roomID}/speech", h.speech)
	router.Patch("/rooms/{roomID}/name", h.renamePlayer)
	router.Post("/rooms/{roomID}/start", h.start)
	router.Post("/rooms/{roomID}/place", h.place)
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
	var room PublicRoom
	err := h.manager.RunRoomCommand(r.Context(), roomID, gameactor.EventPlayerConnected, gameactor.LanePresence, func() error {
		var err error
		room, err = h.manager.JoinRoom(roomID, toUserView(user))
		return err
	})
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	h.hub.Subscribe(r.Context(), room.ID, user.ID, conn)
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room := h.manager.CreateRoom(toUserView(user))
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) currentRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, ok := h.manager.CurrentRoomForUser(user.ID)
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, map[string]*PublicRoom{"room": nil})
		return
	}
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
	roomID := chi.URLParam(r, "roomID")
	var room PublicRoom
	err := h.manager.RunRoomCommand(r.Context(), roomID, gameactor.EventPlayerConnected, gameactor.LanePresence, func() error {
		var err error
		room, err = h.manager.JoinRoom(roomID, toUserView(user))
		return err
	})
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.Broadcast(room.ID)
	h.hub.ScheduleAIAction(room.ID)
	h.hub.ScheduleAIOptionalSpeech(room.ID)
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

func (h *Handler) removePlayer(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	roomID := chi.URLParam(r, "roomID")
	playerID := chi.URLParam(r, "playerID")
	targetUserID := ""
	if current, err := h.manager.Public(roomID, user.ID); err == nil {
		targetUserID = playerUserID(current.Players, playerID)
	}
	var room PublicRoom
	err := h.manager.RunRoomCommand(r.Context(), roomID, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, func() error {
		var err error
		room, err = h.manager.RemovePlayer(roomID, user.ID, playerID)
		return err
	})
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.CloseUser(room.ID, targetUserID)
	h.hub.Broadcast(room.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) speech(w http.ResponseWriter, r *http.Request) {
	var request speechRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoomWithEvent(w, r, gameactor.EventPlayerSpeech, gameactor.LaneSpeech, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Say(roomID, userID, request.Text)
	})
}

func (h *Handler) renamePlayer(w http.ResponseWriter, r *http.Request) {
	var request nameRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.RenamePlayer(roomID, userID, request.Name)
	})
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Start(roomID, userID)
	})
}

func (h *Handler) place(w http.ResponseWriter, r *http.Request) {
	var request placeRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Place(roomID, userID, request.X, request.Y)
	})
}

func (h *Handler) mutateRoom(w http.ResponseWriter, r *http.Request, mutate func(roomID string, userID string) (PublicRoom, error)) {
	h.mutateRoomWithEvent(w, r, gameactor.EventHumanIntentSubmitted, gameactor.LaneRule, mutate)
}

func (h *Handler) mutateRoomWithEvent(w http.ResponseWriter, r *http.Request, eventType gameactor.RoomEventType, lane gameactor.EventLane, mutate func(roomID string, userID string) (PublicRoom, error)) {
	user := mustUser(r)
	roomID := chi.URLParam(r, "roomID")
	var room PublicRoom
	err := h.manager.RunRoomCommand(r.Context(), roomID, eventType, lane, func() error {
		var err error
		room, err = mutate(roomID, user.ID)
		return err
	})
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.hub.Broadcast(room.ID)
	h.hub.ScheduleAIAction(room.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func mustUser(r *http.Request) *auth.User {
	user, _ := auth.UserFromContext(r.Context())
	return user
}

func toUserView(user *auth.User) UserView {
	return UserView{ID: user.ID, DisplayName: user.DisplayName, Role: string(user.Role), Kind: string(user.Kind)}
}

func playerUserID(players []Player, playerID string) string {
	for _, player := range players {
		if player.ID == playerID {
			return player.UserID
		}
	}
	return ""
}
