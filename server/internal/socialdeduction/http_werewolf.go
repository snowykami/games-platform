package socialdeduction

import (
	"net/http"

	"github.com/snowykami/games-platform/server/internal/gameactor"
	"github.com/snowykami/games-platform/server/internal/httpx"
)

func (h *Handler) updateWerewolfRoles(w http.ResponseWriter, r *http.Request) {
	var request werewolfRoleConfigRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UpdateWerewolfRoles(roomID, userID, request.Config)
	})
}

func (h *Handler) nightAction(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		actionID := request.ActionID
		if actionID == "" {
			actionID = request.TargetID
		}
		return h.manager.NightAction(roomID, userID, actionID)
	})
}

func (h *Handler) wolfSpeech(w http.ResponseWriter, r *http.Request) {
	var request speechRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoomWithEvent(w, r, gameactor.EventPlayerSpeech, gameactor.LaneSpeech, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.WerewolfWolfSpeech(roomID, userID, request.Text)
	})
}

func (h *Handler) hunterShot(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.HunterShot(roomID, userID, request.TargetID)
	})
}

func (h *Handler) advanceDay(w http.ResponseWriter, r *http.Request) {
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.AdvanceDay(roomID, userID)
	})
}

func (h *Handler) werewolfVote(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.WerewolfVote(roomID, userID, request.TargetID, request.Confirmed)
	})
}
