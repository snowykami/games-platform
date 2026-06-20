package aiplayer

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/snowykami/games-platform/server/internal/httpx"
)

type Handler struct {
	provider Provider
}

type capabilitiesResponse struct {
	LLMEnabled bool         `json:"llmEnabled"`
	Levels     []Level      `json:"levels"`
	Model      string       `json:"model,omitempty"`
	Profiles   []BotProfile `json:"profiles"`
}

type decideRequest struct {
	Game        string        `json:"game"`
	Level       string        `json:"level"`
	SessionID   string        `json:"sessionId"`
	PlayerName  string        `json:"playerName"`
	Personality string        `json:"personality"`
	SpeechStyle string        `json:"speechStyle"`
	State       any           `json:"state"`
	Actions     []LegalAction `json:"actions"`
}

type decideResponse struct {
	Decision Decision `json:"decision"`
}

func NewHandler(provider Provider) *Handler {
	return &Handler{provider: provider}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()
	router.Get("/capabilities", h.capabilities)
	router.Post("/decide", h.decide)
	return router
}

func (h *Handler) capabilities(w http.ResponseWriter, _ *http.Request) {
	levels := []Level{LevelBeginner, LevelNormal, LevelMaster}
	llmEnabled := h.provider != nil && h.provider.Enabled()
	if llmEnabled {
		levels = append(levels, LevelLLM)
	}
	model := ""
	if modelProvider, ok := h.provider.(interface{ ModelName() string }); ok {
		model = modelProvider.ModelName()
	}

	httpx.WriteJSON(w, http.StatusOK, capabilitiesResponse{LLMEnabled: llmEnabled, Levels: levels, Model: model, Profiles: Profiles()})
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	var request decideRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	llmEnabled := h.provider != nil && h.provider.Enabled()
	level := NormalizeLevel(request.Level, llmEnabled)
	input := DecisionInput{
		Game:        request.Game,
		Level:       level,
		SessionID:   request.SessionID,
		PlayerName:  request.PlayerName,
		Personality: request.Personality,
		SpeechStyle: request.SpeechStyle,
		State:       request.State,
		Actions:     request.Actions,
	}

	decision, err := FirstAction(input.Actions)
	if level == LevelLLM && llmEnabled {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if nextDecision, nextErr := h.provider.Decide(ctx, input); nextErr == nil {
			decision = nextDecision
		} else {
			slog.Warn("generic ai llm decision failed, falling back",
				"game", request.Game,
				"session", request.SessionID,
				"player", request.PlayerName,
				"level", level,
				"actionCount", len(request.Actions),
				"error", nextErr,
			)
		}
	}
	if err != nil && decision.ActionID == "" {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, decideResponse{Decision: decision})
}
