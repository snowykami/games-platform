package mahjong

import (
	"errors"
	"strings"
	"time"
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

func (m *Manager) CurrentRoomForUser(userID string) (PublicRoom, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var current PublicRoom
	var currentUpdatedAt time.Time
	var found bool
	for _, room := range m.rooms {
		if room.Phase == PhaseFinished || !roomHasHumanUser(room, userID) {
			continue
		}
		if !found || room.UpdatedAt.After(currentUpdatedAt) {
			current = publicRoom(room, userID)
			currentUpdatedAt = room.UpdatedAt
			found = true
		}
	}
	return current, found
}

func roomHasHumanUser(room *Room, userID string) bool {
	for _, player := range room.Players {
		if player.UserID == userID && !player.IsAI {
			return true
		}
	}
	return false
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
