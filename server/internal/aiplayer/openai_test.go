package aiplayer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/snowykami/games-platform/server/internal/config"
)

func TestOpenAIProviderRetriesRateLimitThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("play:card"))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(config.AIConfig{LLMAPI: server.URL, LLMModel: "test-model", LLMToken: "token"})
	decision, err := provider.Decide(context.Background(), DecisionInput{
		Game:       "uno",
		Level:      LevelLLM,
		SessionID:  "session_1",
		PlayerName: "北风",
		Actions:    []LegalAction{{ID: "play:card", Label: "出牌"}},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if decision.ActionID != "play:card" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestOpenAIProviderDoesNotRetryBadRequest(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(config.AIConfig{LLMAPI: server.URL, LLMModel: "test-model", LLMToken: "token"})
	_, err := provider.Decide(context.Background(), DecisionInput{
		Game:       "uno",
		Level:      LevelLLM,
		SessionID:  "session_1",
		PlayerName: "北风",
		Actions:    []LegalAction{{ID: "draw", Label: "摸牌"}},
	})
	if err == nil {
		t.Fatal("expected bad request error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func chatCompletionResponse(actionID string) string {
	return fmt.Sprintf(`{
		"choices": [{
			"message": {
				"tool_calls": [{
					"function": {
						"arguments": "{\"actionId\":\"%s\",\"reason\":\"看起来合适\",\"speech\":\"\"}"
					}
				}]
			}
		}]
	}`, actionID)
}
