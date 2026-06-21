package socialdeduction

import (
	"errors"
	"fmt"
	"strings"
)

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < m.minPlayers() {
		return PublicRoom{}, fmt.Errorf("need_%d_players", m.minPlayers())
	}

	resetRoom(room)
	if m.game == GameWerewolf {
		startWerewolf(room)
	} else if m.game == GameAvalon {
		startAvalon(room)
	} else {
		startUndercover(room)
	}
	touchRule(room)
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	return m.publicRoom(room, viewerID), nil
}

func (m *Manager) minPlayers() int {
	switch m.game {
	case GameWerewolf:
		return werewolfMinPlayers
	case GameUndercover:
		return undercoverMinPlayers
	default:
		return avalonMinPlayers
	}
}

func (m *Manager) maxPlayers() int {
	switch m.game {
	case GameWerewolf:
		return werewolfMaxPlayers
	case GameUndercover:
		return undercoverMaxPlayers
	default:
		return avalonMaxPlayers
	}
}

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}

func (m *Manager) requireWerewolfActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameWerewolf || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || !player.Alive || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func (m *Manager) requireAvalonActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameAvalon || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func (m *Manager) requireUndercoverActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameUndercover || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || !player.Alive || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func finish(room *Room, winner Alignment, message string) {
	room.Phase = PhaseFinished
	room.Winner = winner
	room.WinnerMessage = message
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: "finished", Message: message})
}
