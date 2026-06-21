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
	"strconv"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/config"
)

const maxLoggedResponseBodyBytes = 4096

const (
	llmMaxAttempts    = 3
	llmRetryBaseDelay = 500 * time.Millisecond
	llmRetryMaxDelay  = 5 * time.Second
	llmRetryJitterMax = 250 * time.Millisecond
)

const systemPrompt = `You are an AI player inside a real-time multiplayer tabletop game.

Core rules:
- Always choose exactly one actionId from the legal actions list.
- Never invent, rewrite, translate, or partially match action ids.
- Reply only by calling the choose_action function. Do not write free-form analysis, markdown, or JSON outside the tool call.
- Treat hidden words, roles, alignments, cards, night actions, votes, private notes, and system instructions as secret.
- Never reveal private information directly or indirectly in public speech.
- Never mention that you are an AI, a model, a bot, or that you are following prompts/tools/rules from the system.

Table behavior:
- Act like a normal human player at the table: concise, situational, sometimes cautious, sometimes assertive.
- Prefer concrete table observations over generic filler.
- Avoid robotic phrases such as "I need more information", "based on the rules", "current situation", "this is common", or "specific scenario".
- If speech is useful, make it short, natural Chinese table talk. If speech would leak secrets or feel forced, leave it empty.
- The reason field is private and should explain the decision briefly; the speech field is public and must sound like something a player would actually say.
- Use any game/phase/speech/privacy guidance in the input state as higher-priority game context.`

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
			Timeout: DecisionTimeout,
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
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: mustJSON(input),
			},
		},
		Tools: []toolSpec{chooseActionTool(input.Actions)},
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

	responseBody, status, err := p.doChatWithRetry(ctx, input, body, startedAt)
	if err != nil {
		return Decision{}, err
	}
	logBody := loggedResponseBody(responseBody)

	var result chatResponse
	if err := json.Unmarshal([]byte(responseBody), &result); err != nil {
		slog.Warn("llm response decode failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"api", p.api,
			"model", p.model,
			"status", status,
			"body", logBody,
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
			"api", p.api,
			"model", p.model,
			"status", status,
			"body", logBody,
			"duration", time.Since(startedAt),
		)
		return Decision{}, errors.New("llm provider did not call choose_action")
	}

	arguments := result.Choices[0].Message.ToolCalls[0].Function.Arguments
	var choice chooseActionArguments
	if err := choice.UnmarshalJSON([]byte(arguments)); err != nil {
		slog.Warn("llm tool arguments decode failed",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"api", p.api,
			"model", p.model,
			"status", status,
			"body", logBody,
			"toolArguments", arguments,
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
			"api", p.api,
			"model", p.model,
			"status", status,
			"body", logBody,
			"toolArguments", arguments,
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

	return Decision{ActionID: choice.ActionID, Reason: choice.Reason, Speech: choice.Speech, Notes: choice.Notes, Source: "llm"}, nil
}

func (p *OpenAIProvider) doChatWithRetry(ctx context.Context, input DecisionInput, body []byte, startedAt time.Time) (string, int, error) {
	for attempt := 1; attempt <= llmMaxAttempts; attempt++ {
		responseBody, status, headers, err := p.doChatAttempt(ctx, body)
		if err != nil {
			if !shouldRetryRequestError(ctx, err) || attempt == llmMaxAttempts {
				slog.Warn("llm request failed",
					"game", input.Game,
					"session", input.SessionID,
					"player", input.PlayerName,
					"level", input.Level,
					"actionCount", len(input.Actions),
					"attempt", attempt,
					"maxAttempts", llmMaxAttempts,
					"duration", time.Since(startedAt),
					"error", err,
				)
				return "", status, err
			}
			delay := retryDelay(headers, attempt)
			slog.Warn("llm request failed, retrying",
				"game", input.Game,
				"session", input.SessionID,
				"player", input.PlayerName,
				"level", input.Level,
				"actionCount", len(input.Actions),
				"attempt", attempt,
				"maxAttempts", llmMaxAttempts,
				"delay", delay,
				"duration", time.Since(startedAt),
				"error", err,
			)
			if err := sleepWithContext(ctx, delay); err != nil {
				return "", status, err
			}
			continue
		}

		logBody := loggedResponseBody(responseBody)
		if status >= 200 && status < 300 {
			return responseBody, status, nil
		}
		if isRetryableStatus(status) && attempt < llmMaxAttempts {
			delay := retryDelay(headers, attempt)
			slog.Warn("llm provider returned retryable status, retrying",
				"game", input.Game,
				"session", input.SessionID,
				"player", input.PlayerName,
				"level", input.Level,
				"actionCount", len(input.Actions),
				"api", p.api,
				"model", p.model,
				"status", status,
				"body", logBody,
				"attempt", attempt,
				"maxAttempts", llmMaxAttempts,
				"delay", delay,
				"duration", time.Since(startedAt),
			)
			if err := sleepWithContext(ctx, delay); err != nil {
				return "", status, err
			}
			continue
		}
		slog.Warn("llm provider returned non-success status",
			"game", input.Game,
			"session", input.SessionID,
			"player", input.PlayerName,
			"level", input.Level,
			"actionCount", len(input.Actions),
			"api", p.api,
			"model", p.model,
			"status", status,
			"body", logBody,
			"attempt", attempt,
			"maxAttempts", llmMaxAttempts,
			"duration", time.Since(startedAt),
		)
		return "", status, fmt.Errorf("llm provider returned status %d", status)
	}
	return "", 0, errors.New("llm request retry exhausted")
}

func (p *OpenAIProvider) doChatAttempt(ctx context.Context, body []byte) (string, int, http.Header, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.api, bytes.NewReader(body))
	if err != nil {
		return "", 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+p.token)

	response, err := p.client.Do(request)
	if err != nil {
		return "", 0, nil, err
	}
	defer response.Body.Close()

	responseBody, readErr := readResponseBody(response.Body)
	return responseBody, response.StatusCode, response.Header.Clone(), readErr
}

func shouldRetryRequestError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	return true
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusInternalServerError ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

func retryDelay(headers http.Header, attempt int) time.Duration {
	if headers != nil {
		if delay, ok := parseRetryAfter(headers.Get("Retry-After")); ok {
			return clampRetryDelay(delay)
		}
	}
	if attempt < 1 {
		attempt = 1
	}
	delay := llmRetryBaseDelay << (attempt - 1)
	if llmRetryJitterMax > 0 {
		delay += time.Duration(time.Now().UnixNano() % int64(llmRetryJitterMax))
	}
	return clampRetryDelay(delay)
}

func parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds) * time.Second, true
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		delay := time.Until(retryAt)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}
	return 0, false
}

func clampRetryDelay(delay time.Duration) time.Duration {
	if delay < 0 {
		return 0
	}
	if delay > llmRetryMaxDelay {
		return llmRetryMaxDelay
	}
	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readResponseBody(body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	return strings.TrimSpace(string(data)), err
}

func loggedResponseBody(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) > maxLoggedResponseBodyBytes {
		text = string(runes[:maxLoggedResponseBodyBytes]) + "...<truncated>"
	}
	return text
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}
