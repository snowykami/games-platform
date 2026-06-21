package socialdeduction

import (
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

const (
	maxAIDebugTraces        = 24
	maxAIDebugTextRunes     = 600
	maxAIDebugThinkingRunes = 6000
)

func recordAIDebugTrace(room *Room, player *Player, phase Phase, scope socialDecisionScope, actions []aiplayer.LegalAction, decision aiplayer.Decision, decisionErr error, duration time.Duration) {
	if room == nil {
		return
	}
	trace := AIDebugTrace{
		ID:                "ai_trace_" + randomToken(8),
		Phase:             phase,
		Scope:             string(scope),
		ActionID:          decision.ActionID,
		Reason:            trimDebugRunes(decision.Reason, maxAIDebugTextRunes),
		Speech:            trimDebugRunes(decision.Speech, maxAIDebugTextRunes),
		Thinking:          trimDebugRunes(decision.Thinking, maxAIDebugThinkingRunes),
		ThinkingAvailable: strings.TrimSpace(decision.Thinking) != "",
		DurationMs:        duration.Milliseconds(),
		Actions:           debugActions(actions),
		CreatedAt:         time.Now().UTC(),
	}
	if player != nil {
		trace.PlayerID = player.ID
		trace.PlayerName = player.Name
	}
	if decisionErr != nil {
		trace.Error = trimDebugRunes(decisionErr.Error(), maxAIDebugTextRunes)
	}
	room.AIDebugTraces = append(room.AIDebugTraces, trace)
	if len(room.AIDebugTraces) > maxAIDebugTraces {
		room.AIDebugTraces = room.AIDebugTraces[len(room.AIDebugTraces)-maxAIDebugTraces:]
	}
}

func debugActions(actions []aiplayer.LegalAction) []AIDebugAction {
	if len(actions) == 0 {
		return nil
	}
	items := make([]AIDebugAction, 0, len(actions))
	for _, action := range actions {
		items = append(items, AIDebugAction{
			ID:          action.ID,
			Label:       action.Label,
			Description: action.Description,
		})
	}
	return items
}

func trimDebugRunes(value string, limit int) string {
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
