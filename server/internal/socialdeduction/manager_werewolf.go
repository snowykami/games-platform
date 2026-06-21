package socialdeduction

import (
	"errors"
	"fmt"
	"strings"
)

func (m *Manager) UpdateWerewolfRoles(roomID string, actorID string, config WerewolfRoleConfig) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf {
		return PublicRoom{}, errors.New("not_werewolf_room")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_update_roles")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("roles_only_lobby")
	}

	nextConfig, err := normalizeWerewolfConfig(config, len(room.Players))
	if err != nil {
		return PublicRoom{}, err
	}
	room.Werewolf.RoleConfig = nextConfig
	room.Log = append(room.Log, createLog(fmt.Sprintf("房主将角色组调整为：%s。", nextConfig.Name)))
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) NightAction(roomID string, actorID string, actionID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireWerewolfActor(roomID, actorID, PhaseWerewolfNight)
	if err != nil {
		return PublicRoom{}, err
	}
	if !canActAtNight(player) {
		return PublicRoom{}, errors.New("role_has_no_night_action")
	}
	if actionID == "" {
		return PublicRoom{}, errors.New("invalid_target")
	}
	if !strings.Contains(actionID, ":") {
		actionID = "target:" + actionID
	}
	target, err := applyWerewolfNightAction(room, player, actionID)
	if err != nil {
		return PublicRoom{}, err
	}
	targetID := ""
	if target != nil {
		targetID = target.ID
	}
	recordAction(room, PublicAction{Type: "night_action", ActorID: player.ID, ActorName: player.Name, TargetID: targetID, Message: fmt.Sprintf("%s 完成了夜晚行动。", player.Name)})
	m.advanceWerewolfNight(room)
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) HunterShot(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfHunter {
		return PublicRoom{}, errors.New("invalid_phase")
	}
	hunter := findPlayerByID(room, room.Werewolf.HunterPendingID)
	player := findPlayerByUserID(room, actorID)
	if hunter == nil || player == nil || hunter.ID != player.ID || player.IsAI {
		return PublicRoom{}, errors.New("not_active_human_player")
	}
	if err := resolveHunterShot(room, targetID); err != nil {
		return PublicRoom{}, err
	}
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) AdvanceDay(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay {
		return PublicRoom{}, errors.New("invalid_phase")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_advance")
	}
	room.Phase = PhaseWerewolfVote
	room.Werewolf.Votes = map[string]WerewolfVoteIntent{}
	room.Log = append(room.Log, createLog("白天讨论结束，开始放逐投票。"))
	recordAction(room, PublicAction{Type: "vote_started", Message: "开始放逐投票。"})
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) WerewolfVote(roomID string, actorID string, targetID string, confirmed bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireWerewolfActor(roomID, actorID, PhaseWerewolfVote)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Werewolf.RevealedIdiots[player.ID] {
		return PublicRoom{}, errors.New("idiot_cannot_vote_after_reveal")
	}
	target := findPlayerByID(room, targetID)
	if target == nil || !target.Alive {
		return PublicRoom{}, errors.New("invalid_target")
	}
	previous := room.Werewolf.Votes[player.ID]
	room.Werewolf.Votes[player.ID] = WerewolfVoteIntent{TargetID: target.ID, Confirmed: confirmed}
	if confirmed {
		recordAction(room, PublicAction{Type: "vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已确认投票。", player.Name)})
		m.resolveWerewolfVote(room)
	} else if previous.TargetID != target.ID || previous.Confirmed {
		recordAction(room, PublicAction{Type: "vote_select", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 选择了投票目标。", player.Name)})
	}
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}
