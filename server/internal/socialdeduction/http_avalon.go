package socialdeduction

import (
	"net/http"

	"github.com/snowykami/games-platform/server/internal/httpx"
)

func (h *Handler) proposeTeam(w http.ResponseWriter, r *http.Request) {
	var request teamRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.ProposeTeam(roomID, userID, request.Team)
	})
}

func (h *Handler) teamVote(w http.ResponseWriter, r *http.Request) {
	var request teamVoteRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.TeamVote(roomID, userID, request.Approve)
	})
}

func (h *Handler) quest(w http.ResponseWriter, r *http.Request) {
	var request questRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.QuestCard(roomID, userID, request.Card)
	})
}

func (h *Handler) assassinate(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.Assassinate(roomID, userID, request.TargetID)
	})
}
