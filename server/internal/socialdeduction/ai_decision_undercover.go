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

func (m *Manager) chooseUndercoverVote(room *Room, player *Player) *Player {
	actions := undercoverVoteActions(room, player)
	if len(actions) == 0 {
		return nil
	}
	if m.canUseLLM(player) {
		llmActions, actionMap := playerTargetActionsForLLM(room, actions, []string{"vote:"})
		decision, err := m.socialDecision(room, player, undercoverAIState(room, player, "vote"), llmActions)
		if err == nil && strings.HasPrefix(decision.ActionID, "vote:") {
			actionID := actionMap[decision.ActionID]
			if actionID == "" {
				actionID = decision.ActionID
			}
			return findPlayerByID(room, strings.TrimPrefix(actionID, "vote:"))
		}
		if err != nil {
			slog.Warn("undercover llm vote failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return nil
}
