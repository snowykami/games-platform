package socialdeduction

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func (m *Manager) chooseAvalonTeam(room *Room, leader *Player) ([]string, string) {
	actions := avalonTeamActions(room, leader)
	if m.canUseLLM(leader) && len(actions) > 0 {
		llmActions, actionMap := avalonTeamActionsForLLM(room, actions)
		decision, err := m.socialDecision(room, leader, avalonAIState(room, leader, "team"), llmActions)
		if err == nil {
			actionID := actionMap[decision.ActionID]
			if actionID == "" {
				actionID = decision.ActionID
			}
			team := strings.Split(strings.TrimPrefix(actionID, "team:"), ",")
			if len(team) == room.Avalon.RequiredTeam {
				return team, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("avalon llm team failed", "room", room.ID, "player", leader.ID, "playerName", leader.Name, "error", err)
		}
	}
	return nil, ""
}

func (m *Manager) chooseAvalonTeamVote(room *Room, player *Player) (bool, string, bool) {
	actions := []aiplayer.LegalAction{
		{ID: "vote:approve", Label: "同意这支队伍"},
		{ID: "vote:reject", Label: "反对这支队伍"},
	}
	if m.canUseLLM(player) {
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "team_vote"), actions)
		if err == nil {
			return decision.ActionID == "vote:approve", strings.TrimSpace(decision.Speech), true
		}
		slog.Warn("avalon llm team vote failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
	}
	return false, "", false
}

func (m *Manager) chooseAvalonQuestCard(room *Room, player *Player) (string, string) {
	actions := []aiplayer.LegalAction{{ID: "quest:success", Label: "提交成功牌"}}
	if player.Alignment == AlignmentEvil {
		actions = append(actions, aiplayer.LegalAction{ID: "quest:fail", Label: "提交失败牌"})
	}
	if m.canUseLLM(player) {
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "quest"), actions)
		if err == nil && decision.ActionID == "quest:fail" && player.Alignment == AlignmentEvil {
			return "fail", strings.TrimSpace(decision.Speech)
		}
		if err == nil {
			return "success", strings.TrimSpace(decision.Speech)
		}
		slog.Warn("avalon llm quest failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
	}
	return "", ""
}

func (m *Manager) chooseAvalonAssassination(room *Room, player *Player) (*Player, string) {
	actions := []aiplayer.LegalAction{}
	for _, target := range goodPlayers(room) {
		actions = append(actions, aiplayer.LegalAction{ID: "assassinate:" + target.ID, Label: fmt.Sprintf("刺杀 %s", target.Name)})
	}
	if m.canUseLLM(player) && len(actions) > 0 {
		llmActions, actionMap := playerTargetActionsForLLM(room, actions, []string{"assassinate:"})
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "assassination"), llmActions)
		if err == nil {
			actionID := actionMap[decision.ActionID]
			if actionID == "" {
				actionID = decision.ActionID
			}
			if target := playerFromAction(room, actionID, "assassinate:"); target != nil {
				return target, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("avalon llm assassination failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return nil, ""
}
