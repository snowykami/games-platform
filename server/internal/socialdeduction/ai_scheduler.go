package socialdeduction

import "strings"

// Legacy AI scheduler. Keep behavior stable while games migrate to RoomActor one by one.
// Do not add new AI capabilities here; new work should target gameactor/aiagent adapters.
func (m *Manager) RunAIOptionalSpeech(roomID string) (PublicRoom, bool, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return PublicRoom{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || room.Phase == PhaseFinished || len(room.Speeches) == 0 {
		return m.publicRoom(room, ""), false, nil
	}
	if hasPendingAIRequiredAction(room) {
		return m.publicRoom(room, ""), false, nil
	}
	lastSpeech := room.Speeches[len(room.Speeches)-1]
	if lastSpeech.ID == room.LastAISpeechSourceID {
		return m.publicRoom(room, ""), false, nil
	}
	player := nextAISpeechPlayer(room, lastSpeech.PlayerID)
	if player == nil {
		room.LastAISpeechSourceID = lastSpeech.ID
		return m.publicRoom(room, ""), false, nil
	}
	room.LastAISpeechSourceID = lastSpeech.ID
	state := m.aiSpeechState(room, player)
	decision, err := m.socialSpeechDecision(room, player, state, speechActions())
	if err != nil {
		return PublicRoom{}, false, err
	}
	if decision.ActionID != "speak" || strings.TrimSpace(decision.Speech) == "" {
		return PublicRoom{}, false, nil
	}
	player = findPlayerByID(room, player.ID)
	if player == nil || !player.IsAI || !player.Alive {
		return m.publicRoom(room, ""), false, nil
	}
	if !recordSpeech(room, player, decision.Speech) {
		return m.publicRoom(room, ""), false, nil
	}
	touchSpeech(room)
	return m.publicRoom(room, ""), true, nil
}

func (m *Manager) RunAIAction(roomID string) (PublicRoom, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || room.Phase == PhaseFinished {
		return m.publicRoom(room, ""), false, nil
	}

	acted := false
	for _, player := range room.Players {
		if !player.IsAI || !player.Alive {
			continue
		}
		if m.runAIAction(room, player) {
			acted = true
			break
		}
	}
	if acted {
		touchRule(room)
		return m.publicRoom(room, ""), room.Phase != PhaseFinished, nil
	}
	return m.publicRoom(room, ""), false, nil
}

func hasPendingAIRequiredAction(room *Room) bool {
	switch room.Phase {
	case PhaseWerewolfNight:
		for _, player := range room.Players {
			if player.IsAI && player.Alive && canActAtNight(player) {
				if _, ok := room.Werewolf.NightActions[player.ID]; !ok {
					return true
				}
			}
		}
	case PhaseWerewolfVote:
		for _, player := range room.Players {
			if player.IsAI && player.Alive && !room.Werewolf.RevealedIdiots[player.ID] {
				if vote, ok := room.Werewolf.Votes[player.ID]; !ok || !vote.Confirmed {
					return true
				}
			}
		}
	case PhaseWerewolfHunter:
		player := findPlayerByID(room, room.Werewolf.HunterPendingID)
		return player != nil && player.IsAI && player.Alive
	case PhaseAvalonTeam:
		player := findPlayerByID(room, room.Avalon.LeaderID)
		return player != nil && player.IsAI && player.Alive && len(room.Avalon.Team) == 0
	case PhaseAvalonVote:
		for _, player := range room.Players {
			if player.IsAI && player.Alive {
				if _, ok := room.Avalon.TeamVotes[player.ID]; !ok {
					return true
				}
			}
		}
	case PhaseAvalonQuest:
		for _, playerID := range room.Avalon.Team {
			player := findPlayerByID(room, playerID)
			if player != nil && player.IsAI && player.Alive {
				if _, ok := room.Avalon.QuestCards[player.ID]; !ok {
					return true
				}
			}
		}
	case PhaseAssassination:
		for _, player := range room.Players {
			if player.IsAI && player.Alive && player.Role == RoleAssassin {
				return true
			}
		}
	case PhaseUndercoverDescribe:
		player := findPlayerByID(room, room.Undercover.CurrentSpeakerID)
		return player != nil && player.IsAI && player.Alive && !room.Undercover.Described[player.ID]
	case PhaseUndercoverVote:
		for _, player := range room.Players {
			if player.IsAI && player.Alive {
				if vote, ok := room.Undercover.Votes[player.ID]; !ok || !vote.Confirmed {
					return true
				}
			}
		}
	}
	return false
}
