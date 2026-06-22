package socialdeduction

import (
	"net/http"

	"github.com/snowykami/games-platform/server/internal/httpx"
)

func (h *Handler) updateUndercoverConfig(w http.ResponseWriter, r *http.Request) {
	var request undercoverConfigRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UpdateUndercoverConfig(roomID, userID, request.DomainIDs, request.IncludeBlank)
	})
}

func (h *Handler) undercoverDescribe(w http.ResponseWriter, r *http.Request) {
	var request describeRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UndercoverDescribe(roomID, userID, request.Text)
	})
}

func (h *Handler) undercoverVote(w http.ResponseWriter, r *http.Request) {
	var request targetRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}
	h.mutateRoom(w, r, func(roomID string, userID string) (PublicRoom, error) {
		return h.manager.UndercoverVote(roomID, userID, request.TargetID, request.Confirmed)
	})
}
