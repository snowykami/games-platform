package xiangqi

import (
	"errors"
	"strings"
)

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	return publicRoom(room, viewerID), nil
}

func (m *Manager) currentHuman(roomID string, actorID string) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Phase != PhasePlaying {
		return nil, nil, errors.New("game_not_playing")
	}
	if len(room.Players) == 0 || room.CurrentPlayerIndex >= len(room.Players) {
		return nil, nil, errors.New("invalid_turn")
	}

	player := room.Players[room.CurrentPlayerIndex]
	if player.UserID != actorID || player.IsAI {
		return nil, nil, errors.New("not_current_turn")
	}
	return room, player, nil
}

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}
