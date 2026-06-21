package socialdeduction

import (
	"errors"
	"fmt"
)

func (m *Manager) UpdateUndercoverConfig(roomID string, actorID string, presetID string, includeBlank bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameUndercover {
		return PublicRoom{}, errors.New("not_undercover_room")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_update_undercover")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("undercover_config_only_lobby")
	}
	if !undercoverPresetExists(presetID) {
		return PublicRoom{}, errors.New("invalid_undercover_preset")
	}
	room.Undercover.PresetID = presetID
	room.Undercover.IncludeBlank = includeBlank
	room.Undercover.Presets = undercoverPresets()
	room.Log = append(room.Log, createLog(fmt.Sprintf("房主选择了题库：%s。", undercoverPresetName(presetID))))
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UndercoverDescribe(roomID string, actorID string, text string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireUndercoverActor(roomID, actorID, PhaseUndercoverDescribe)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Undercover.CurrentSpeakerID != player.ID {
		return PublicRoom{}, errors.New("not_current_speaker")
	}
	if !recordSpeech(room, player, text) {
		return PublicRoom{}, errors.New("invalid_speech")
	}
	room.Undercover.Described[player.ID] = true
	recordAction(room, PublicAction{Type: "undercover_describe", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 完成了描述。", player.Name)})
	advanceUndercoverSpeaker(room)
	touchRuleAndSpeech(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UndercoverVote(roomID string, actorID string, targetID string, confirmed bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireUndercoverActor(roomID, actorID, PhaseUndercoverVote)
	if err != nil {
		return PublicRoom{}, err
	}
	target := findPlayerByID(room, targetID)
	if target == nil || !target.Alive || target.ID == player.ID {
		return PublicRoom{}, errors.New("invalid_target")
	}
	previous := room.Undercover.Votes[player.ID]
	room.Undercover.Votes[player.ID] = UndercoverVoteIntent{TargetID: target.ID, Confirmed: confirmed}
	if confirmed {
		recordAction(room, PublicAction{Type: "undercover_vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已确认投票。", player.Name)})
		resolveUndercoverVote(room)
	} else if previous.TargetID != target.ID || previous.Confirmed {
		recordAction(room, PublicAction{Type: "undercover_vote_select", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 选择了投票目标。", player.Name)})
	}
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}
