package aiplayer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/config"
)

type OpenAIProvider struct {
	api    string
	model  string
	token  string
	client *http.Client
}

func NewOpenAIProvider(cfg config.AIConfig) *OpenAIProvider {
	return &OpenAIProvider{
		api:   strings.TrimSpace(cfg.LLMAPI),
		model: strings.TrimSpace(cfg.LLMModel),
		token: strings.TrimSpace(cfg.LLMToken),
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (p *OpenAIProvider) Enabled() bool {
	return p.api != "" && p.model != "" && p.token != ""
}

func (p *OpenAIProvider) Decide(ctx context.Context, input DecisionInput) (Decision, error) {
	if !p.Enabled() {
		return Decision{}, errors.New("llm provider is not configured")
	}
	if len(input.Actions) == 0 {
		return Decision{}, errors.New("no legal actions")
	}

	payload := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You are a game AI. You must choose exactly one action from the provided legal actions. Do not invent actions. Reply only by calling choose_action.",
			},
			{
				Role:    "user",
				Content: mustJSON(input),
			},
		},
		Tools: []toolSpec{chooseActionTool()},
	}

	startedAt := time.Now()
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("llm request payload encode failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"error", err,
		)
		return Decision{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.api, bytes.NewReader(body))
	if err != nil {
		slog.Warn("llm request creation failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"error", err,
		)
		return Decision{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+p.token)

	response, err := p.client.Do(request)
	if err != nil {
		slog.Warn("llm request failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"duration", time.Since(startedAt),
			"error", err,
		)
		return Decision{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		slog.Warn("llm provider returned non-success status",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"status", response.StatusCode,
			"body", strings.TrimSpace(string(responseBody)),
			"duration", time.Since(startedAt),
		)
		return Decision{}, fmt.Errorf("llm provider returned status %d", response.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		slog.Warn("llm response decode failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"duration", time.Since(startedAt),
			"error", err,
		)
		return Decision{}, err
	}
	if len(result.Choices) == 0 || len(result.Choices[0].Message.ToolCalls) == 0 {
		slog.Warn("llm response missing tool call",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"duration", time.Since(startedAt),
		)
		return Decision{}, errors.New("llm provider did not call choose_action")
	}

	arguments := result.Choices[0].Message.ToolCalls[0].Function.Arguments
	var choice chooseActionArguments
	if err := json.Unmarshal([]byte(arguments), &choice); err != nil {
		slog.Warn("llm tool arguments decode failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"duration", time.Since(startedAt),
			"error", err,
		)
		return Decision{}, err
	}
	if !ValidateAction(choice.ActionID, input.Actions) {
		slog.Warn("llm selected illegal action",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"actionID", choice.ActionID,
			"duration", time.Since(startedAt),
		)
		return Decision{}, fmt.Errorf("llm selected illegal action %q", choice.ActionID)
	}

	slog.Info("llm decision succeeded",
		"game", input.Game,
		"session", input.SessionID,
		"player", input.PlayerName,
		"level", input.Level,
		"actionCount", len(input.Actions),
		"actionID", choice.ActionID,
		"reasonLength", len(choice.Reason),
		"speechLength", len(choice.Speech),
		"duration", time.Since(startedAt),
	)

	return Decision{ActionID: choice.ActionID, Reason: choice.Reason, Speech: choice.Speech, Source: "llm"}, nil
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func chooseActionTool() toolSpec {
	return toolSpec{
		Type: "function",
		Function: functionSpec{
			Name:        "choose_action",
			Description: "Choose one legal action for the current game turn.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"actionId": map[string]any{
						"type":        "string",
						"description": "The exact id of one action from the legal actions list.",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Short reason in Chinese.",
					},
					"speech": map[string]any{
						"type":        "string",
						"description": "Optional short in-character table talk. Leave empty if the AI should stay silent.",
					},
				},
				"required":             []string{"actionId"},
				"additionalProperties": false,
			},
		},
	}
}

type chatRequest struct {
	Model      string        `json:"model"`
	Messages   []chatMessage `json:"messages"`
	Tools      []toolSpec    `json:"tools"`
	ToolChoice *toolChoice   `json:"tool_choice,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolSpec struct {
	Type     string       `json:"type"`
	Function functionSpec `json:"function"`
}

type functionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type toolChoice struct {
	Type     string             `json:"type"`
	Function toolChoiceFunction `json:"function"`
}

type toolChoiceFunction struct {
	Name string `json:"name"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			ToolCalls []struct {
				Function struct {
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

type chooseActionArguments struct {
	ActionID string `json:"actionId"`
	Reason   string `json:"reason"`
	Speech   string `json:"speech"`
}
