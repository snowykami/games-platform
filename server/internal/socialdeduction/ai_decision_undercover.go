package socialdeduction

import (
	"log/slog"
	"strings"
)

func (m *Manager) chooseUndercoverDescription(room *Room, player *Player) string {
	actions := undercoverDescriptionActions(room, player)
	if len(actions) == 0 {
		return ""
	}
	if m.canUseLLM(player) {
		decision, err := m.socialDecision(room, player, undercoverAIState(room, player, "describe"), actions)
		if err == nil {
			if text, ok := validUndercoverDescription(decision.Speech, undercoverWordForPlayer(room, player)); ok {
				return text
			}
			slog.Warn("undercover llm describe speech rejected",
				"room", room.ID,
				"player", player.ID,
				"playerName", player.Name,
				"actionID", decision.ActionID,
				"speech", strings.TrimSpace(decision.Speech),
			)
			return fallbackUndercoverDescription(decision.ActionID)
		}
		slog.Warn("undercover llm describe failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
	}
	return ""
}

func (m *Manager) chooseUndercoverVote(room *Room, player *Player) (*Player, string) {
	actions := undercoverVoteActions(room, player)
	if len(actions) == 0 {
		return nil, ""
	}
	if m.canUseLLM(player) {
		llmActions, actionMap := playerTargetActionsForLLM(room, actions, []string{"vote:"})
		decision, err := m.socialDecision(room, player, undercoverAIState(room, player, "vote"), llmActions)
		if err == nil && strings.HasPrefix(decision.ActionID, "vote:") {
			actionID := actionMap[decision.ActionID]
			if actionID == "" {
				actionID = decision.ActionID
			}
			return findPlayerByID(room, strings.TrimPrefix(actionID, "vote:")), validUndercoverVoteSpeech(decision.Speech)
		}
		if err != nil {
			slog.Warn("undercover llm vote failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return nil, ""
}

func validUndercoverVoteSpeech(speech string) string {
	speech = strings.TrimSpace(speech)
	if speech == "" || isGenericUndercoverVoteSpeech(speech) {
		return ""
	}
	runes := []rune(speech)
	if len(runes) > 60 {
		return ""
	}
	return speech
}

func isGenericUndercoverVoteSpeech(speech string) bool {
	normalized := strings.TrimSpace(strings.Trim(speech, "。.!！?？~～ "))
	if normalized == "" {
		return true
	}
	generic := map[string]bool{
		"我先投这里":   true,
		"我投这里":    true,
		"先投这里":    true,
		"我先票这里":   true,
		"我票这里":    true,
		"先票这里":    true,
		"我先票这个位置": true,
		"我先投这个位置": true,
		"先票这个位置":  true,
		"先投这个位置":  true,
		"我先票这个":   true,
		"我投这个":    true,
	}
	if generic[normalized] {
		return true
	}
	return strings.Contains(normalized, "这个位置") || strings.Contains(normalized, "这个人")
}
