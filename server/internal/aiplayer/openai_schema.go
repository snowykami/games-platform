package aiplayer

import (
	"bytes"
	"encoding/json"
	"strings"
)

func chooseActionTool(actions []LegalAction) toolSpec {
	actionIDs := make([]string, 0, len(actions))
	for _, action := range actions {
		actionIDs = append(actionIDs, action.ID)
	}
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
						"enum":        actionIDs,
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Private short reason in Chinese, preferably no more than 60 Chinese characters. Explain the key table read, not hidden chain-of-thought.",
						"maxLength":   100,
					},
					"speech": map[string]any{
						"type":        "string",
						"description": "Optional natural public table talk in Chinese, usually 8-50 Chinese characters. Sound like a real player. Never reveal private words, roles, cards, night actions, votes, or notes. Leave empty if silence is better.",
						"maxLength":   80,
					},
					"notePlayerId": map[string]any{
						"type":        "string",
						"description": "Optional single player id to remember a private note about. Use one id from the visible players list, or leave empty.",
					},
					"noteText": map[string]any{
						"type":        "string",
						"description": "Optional short private note in Chinese for notePlayerId, no more than 40 Chinese characters. Leave empty if no note is needed.",
						"maxLength":   80,
					},
				},
				"required":             []string{"actionId"},
				"additionalProperties": false,
			},
		},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []toolSpec    `json:"tools"`
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

type chatResponse struct {
	Choices []struct {
		Message struct {
			ReasoningContent string `json:"reasoning_content"`
			Thinking         string `json:"thinking"`
			ToolCalls        []struct {
				Function struct {
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

type chooseActionArguments struct {
	ActionID string            `json:"actionId"`
	Reason   string            `json:"reason"`
	Speech   string            `json:"speech"`
	NoteID   string            `json:"notePlayerId"`
	NoteText string            `json:"noteText"`
	Notes    map[string]string `json:"notes"`
}

func (args *chooseActionArguments) UnmarshalJSON(data []byte) error {
	var raw struct {
		ActionID     string          `json:"actionId"`
		Reason       string          `json:"reason"`
		Speech       string          `json:"speech"`
		NotePlayerID string          `json:"notePlayerId"`
		NoteText     string          `json:"noteText"`
		Notes        json.RawMessage `json:"notes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	args.ActionID = raw.ActionID
	args.Reason = trimRunes(raw.Reason, 100)
	args.Speech = trimRunes(raw.Speech, 80)
	args.NoteID = strings.TrimSpace(raw.NotePlayerID)
	args.NoteText = trimRunes(raw.NoteText, 80)
	args.Notes = notesFromFlatFields(args.NoteID, args.NoteText)
	if len(args.Notes) == 0 {
		args.Notes = parseLegacyNotes(raw.Notes)
	}
	return nil
}

func trimRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func notesFromFlatFields(playerID string, note string) map[string]string {
	if playerID == "" || note == "" {
		return nil
	}
	return map[string]string{playerID: note}
}

func parseLegacyNotes(raw json.RawMessage) map[string]string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var notes map[string]string
	if err := json.Unmarshal(raw, &notes); err == nil {
		return trimNotes(notes)
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil
	}
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(encoded), &notes); err == nil {
		return trimNotes(notes)
	}
	return nil
}

func trimNotes(notes map[string]string) map[string]string {
	if len(notes) == 0 {
		return nil
	}
	trimmed := map[string]string{}
	for playerID, note := range notes {
		playerID = strings.TrimSpace(playerID)
		note = trimRunes(note, 40)
		if playerID != "" && note != "" {
			trimmed[playerID] = note
		}
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}
