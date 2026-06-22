package socialdeduction

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

type noteRequest struct {
	Note string `json:"note"`
}

type werewolfRoleConfigRequest struct {
	Config WerewolfRoleConfig `json:"config"`
}

type targetRequest struct {
	TargetID  string `json:"targetId"`
	ActionID  string `json:"actionId,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
}

type teamRequest struct {
	Team []string `json:"team"`
}

type teamVoteRequest struct {
	Approve bool `json:"approve"`
}

type questRequest struct {
	Card string `json:"card"`
}

type undercoverConfigRequest struct {
	PresetID     string   `json:"presetId"`
	DomainIDs    []string `json:"domainIds,omitempty"`
	IncludeBlank bool     `json:"includeBlank"`
}

type describeRequest struct {
	Text string `json:"text"`
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
	router.Patch("/rooms/{roomID}/notes/{playerID}", h.updatePlayerNote)
	router.Post("/rooms/{roomID}/start", h.start)
	router.Post("/rooms/{roomID}/werewolf-roles", h.updateWerewolfRoles)
	router.Post("/rooms/{roomID}/night-action", h.nightAction)
	router.Post("/rooms/{roomID}/wolf-speech", h.wolfSpeech)
	router.Post("/rooms/{roomID}/hunter-shot", h.hunterShot)
	router.Post("/rooms/{roomID}/advance-day", h.advanceDay)
	router.Post("/rooms/{roomID}/werewolf-vote", h.werewolfVote)
	router.Post("/rooms/{roomID}/team", h.proposeTeam)
	router.Post("/rooms/{roomID}/team-vote", h.teamVote)
	router.Post("/rooms/{roomID}/quest", h.quest)
	router.Post("/rooms/{roomID}/assassinate", h.assassinate)
	router.Patch("/rooms/{roomID}/undercover-config", h.updateUndercoverConfig)
	router.Post("/rooms/{roomID}/describe", h.undercoverDescribe)
	router.Post("/rooms/{roomID}/undercover-vote", h.undercoverVote)
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
	publicOptions := publicRoomOptionsForRequest(r)
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
	h.hub.Subscribe(r.Context(), room.ID, user.ID, publicOptions.GodViewAvailable, publicOptions.GodView, conn)
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room := h.manager.CreateRoom(toUserView(user))
	if view, err := h.manager.PublicWithOptions(room.ID, user.ID, publicRoomOptionsForRequest(r)); err == nil {
		room = view
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) currentRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, ok := h.manager.CurrentRoomForUser(user.ID, publicRoomOptionsForRequest(r))
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, map[string]*PublicRoom{"room": nil})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) getRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	room, err := h.manager.PublicWithOptions(chi.URLParam(r, "roomID"), user.ID, publicRoomOptionsForRequest(r))
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusNotFound, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func (h *Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	roomID := chi.URLParam(r, "roomID")
	publicOptions := publicRoomOptionsForRequest(r)
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
	if publicOptions.GodView {
		if view, err := h.manager.PublicWithOptions(room.ID, user.ID, publicOptions); err == nil {
			room = view
		}
	}
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

func (h *Handler) updatePlayerNote(w http.ResponseWriter, r *http.Request) {
	var request noteRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UpdatePlayerNote(roomID, userID, chi.URLParam(r, "playerID"), request.Note)
	})
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Start(roomID, userID)
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
	if options := publicRoomOptionsForRequest(r); options.GodView {
		if view, err := h.manager.PublicWithOptions(room.ID, user.ID, options); err == nil {
			room = view
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]PublicRoom{"room": room})
}

func mustUser(r *http.Request) *auth.User {
	user, _ := auth.UserFromContext(r.Context())
	return user
}

func toUserView(user *auth.User) UserView {
	return UserView{ID: user.ID, DisplayName: user.DisplayName, Role: string(user.Role), Kind: string(user.Kind)}
}

func publicRoomOptionsForRequest(r *http.Request) PublicRoomOptions {
	user := mustUser(r)
	isAdmin := user != nil && user.Role == auth.RoleAdmin
	return PublicRoomOptions{
		GodViewAvailable: isAdmin,
		GodView:          isAdmin && r.URL.Query().Get("godView") == "1",
	}
}

func playerUserID(players []PublicPlayer, playerID string) string {
	for _, player := range players {
		if player.ID == playerID {
			return player.UserID
		}
	}
	return ""
}
