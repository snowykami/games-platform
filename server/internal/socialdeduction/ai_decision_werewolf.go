package socialdeduction

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func (m *Manager) chooseWerewolfNightAction(room *Room, actor *Player) (string, string) {
	actions := werewolfNightActions(room, actor)
	if len(actions) == 0 {
		return "", ""
	}
	if m.canUseLLM(actor) {
		llmActions, actionMap := werewolfActionsForLLM(room, actions)
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), llmActions)
		if err == nil {
			if actionID, ok := actionMap[decision.ActionID]; ok && aiplayer.ValidateAction(actionID, actions) {
				return actionID, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("werewolf llm night action failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
			var staleErr staleAIDecisionError
			if errors.As(err, &staleErr) {
				return "", ""
			}
		} else {
			slog.Warn("werewolf llm night action invalid", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "actionID", decision.ActionID)
		}
	}
	return fallbackWerewolfNightAction(actor, actions), ""
}

func (m *Manager) chooseWerewolfVote(room *Room, actor *Player) (*Player, string) {
	actions := werewolfVoteActions(room, actor)
	if len(actions) == 0 {
		return nil, ""
	}
	if m.canUseLLM(actor) {
		llmActions, actionMap := werewolfActionsForLLM(room, actions)
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), llmActions)
		if err == nil {
			if actionID, ok := actionMap[decision.ActionID]; ok && aiplayer.ValidateAction(actionID, actions) {
				if target := playerFromAction(room, actionID, "vote:"); target != nil && target.Alive {
					return target, strings.TrimSpace(decision.Speech)
				}
			}
		}
		if err != nil {
			slog.Warn("werewolf llm vote failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
			var staleErr staleAIDecisionError
			if errors.As(err, &staleErr) {
				return nil, ""
			}
		} else {
			slog.Warn("werewolf llm vote invalid", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "actionID", decision.ActionID)
		}
	}
	target := firstPlayerTarget(room, actions, "vote:")
	return target, ""
}

func (m *Manager) chooseHunterShot(room *Room, actor *Player) (*Player, string, bool) {
	actions := hunterShotActions(room, actor)
	if len(actions) == 0 {
		return nil, "", false
	}
	if m.canUseLLM(actor) {
		llmActions, actionMap := werewolfActionsForLLM(room, actions)
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), llmActions)
		if err == nil {
			actionID, ok := actionMap[decision.ActionID]
			if ok && aiplayer.ValidateAction(actionID, actions) {
				if actionID == "shoot:skip" {
					return nil, strings.TrimSpace(decision.Speech), true
				}
				target := playerFromAction(room, actionID, "shoot:")
				if target != nil && target.Alive {
					return target, strings.TrimSpace(decision.Speech), true
				}
			}
		}
		if err != nil {
			slog.Warn("werewolf llm hunter shot failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
			var staleErr staleAIDecisionError
			if errors.As(err, &staleErr) {
				return nil, "", false
			}
		}
	}
	return nil, "", true
}

func fallbackWerewolfNightAction(actor *Player, actions []aiplayer.LegalAction) string {
	if actor.Role == RoleWitch {
		for _, action := range actions {
			if action.ID == "skip:witch" {
				return action.ID
			}
		}
	}
	if len(actions) == 0 {
		return ""
	}
	return actions[0].ID
}

func firstPlayerTarget(room *Room, actions []aiplayer.LegalAction, prefix string) *Player {
	for _, action := range actions {
		target := playerFromAction(room, action.ID, prefix)
		if target != nil && target.Alive {
			return target
		}
	}
	return nil
}
