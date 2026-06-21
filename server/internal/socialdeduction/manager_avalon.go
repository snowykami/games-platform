package socialdeduction

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func (m *Manager) ProposeTeam(roomID string, actorID string, team []string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonTeam)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Avalon.LeaderID != player.ID {
		return PublicRoom{}, errors.New("only_leader_propose")
	}
	if len(team) != room.Avalon.RequiredTeam {
		return PublicRoom{}, errors.New("invalid_team_size")
	}
	if hasDuplicate(team) {
		return PublicRoom{}, errors.New("duplicate_team_member")
	}
	for _, id := range team {
		target := findPlayerByID(room, id)
		if target == nil || !target.Alive {
			return PublicRoom{}, errors.New("invalid_team_member")
		}
	}

	room.Avalon.Team = append([]string{}, team...)
	room.Avalon.TeamVotes = map[string]bool{}
	room.Phase = PhaseAvalonVote
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 提名了任务队伍。", player.Name)))
	recordAction(room, PublicAction{Type: "team_proposed", ActorID: player.ID, ActorName: player.Name, Message: "任务队伍已提名。"})
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) TeamVote(roomID string, actorID string, approve bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonVote)
	if err != nil {
		return PublicRoom{}, err
	}
	room.Avalon.TeamVotes[player.ID] = approve
	recordAction(room, PublicAction{Type: "team_vote", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 已投票。", player.Name)})
	m.resolveAvalonTeamVote(room)
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) QuestCard(roomID string, actorID string, card string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonQuest)
	if err != nil {
		return PublicRoom{}, err
	}
	if !slices.Contains(room.Avalon.Team, player.ID) {
		return PublicRoom{}, errors.New("not_on_quest_team")
	}
	card = strings.TrimSpace(card)
	if card != "success" && card != "fail" {
		return PublicRoom{}, errors.New("invalid_quest_card")
	}
	if card == "fail" && player.Alignment != AlignmentEvil {
		return PublicRoom{}, errors.New("good_player_cannot_fail")
	}
	room.Avalon.QuestCards[player.ID] = card
	recordAction(room, PublicAction{Type: "quest_card", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 已提交任务牌。", player.Name)})
	m.resolveAvalonQuest(room)
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) Assassinate(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAssassination)
	if err != nil {
		return PublicRoom{}, err
	}
	if player.Role != RoleAssassin {
		return PublicRoom{}, errors.New("only_assassin")
	}
	target := findPlayerByID(room, targetID)
	if target == nil || target.Alignment != AlignmentGood {
		return PublicRoom{}, errors.New("invalid_target")
	}
	if target.Role == RoleMerlin {
		finish(room, AlignmentEvil, fmt.Sprintf("%s 刺中了梅林，邪恶阵营逆转获胜。", player.Name))
	} else {
		finish(room, AlignmentGood, fmt.Sprintf("%s 没有找到梅林，正义阵营获胜。", player.Name))
	}
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}
