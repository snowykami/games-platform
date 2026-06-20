package aiplayer

import (
	"context"
	"errors"
	"strings"
)

type Level string

const (
	LevelBeginner Level = "beginner"
	LevelNormal   Level = "normal"
	LevelMaster   Level = "master"
	LevelLLM      Level = "ai"
)

type LegalAction struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type DecisionInput struct {
	Game        string        `json:"game"`
	Level       Level         `json:"level"`
	SessionID   string        `json:"sessionId"`
	PlayerName  string        `json:"playerName"`
	Personality string        `json:"personality"`
	SpeechStyle string        `json:"speechStyle,omitempty"`
	State       any           `json:"state"`
	Actions     []LegalAction `json:"actions"`
}

type Decision struct {
	ActionID string            `json:"actionId"`
	Reason   string            `json:"reason,omitempty"`
	Speech   string            `json:"speech,omitempty"`
	Notes    map[string]string `json:"notes,omitempty"`
	Source   string            `json:"source"`
}

type Provider interface {
	Enabled() bool
	Decide(ctx context.Context, input DecisionInput) (Decision, error)
}

func NormalizeLevel(value string, llmEnabled bool) Level {
	switch Level(strings.TrimSpace(strings.ToLower(value))) {
	case LevelBeginner:
		return LevelBeginner
	case LevelMaster:
		return LevelMaster
	case LevelLLM:
		if llmEnabled {
			return LevelLLM
		}
		return LevelNormal
	default:
		return LevelNormal
	}
}

func ValidateAction(actionID string, actions []LegalAction) bool {
	for _, action := range actions {
		if action.ID == actionID {
			return true
		}
	}
	return false
}

func FirstAction(actions []LegalAction) (Decision, error) {
	if len(actions) == 0 {
		return Decision{}, errors.New("no legal actions")
	}
	return Decision{ActionID: actions[0].ID, Source: "fallback"}, nil
}
