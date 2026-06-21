package uno

import (
	"errors"
	"strings"
	"time"
)

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	return publicRoom(room, viewerID), nil
}

func (m *Manager) CurrentRoomForUser(userID string) (PublicRoom, bool) {
	m.mu.RLock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, room := range m.rooms {
		rooms = append(rooms, room)
	}
	m.mu.RUnlock()

	var current PublicRoom
	var currentUpdatedAt time.Time
	var found bool
	for _, room := range rooms {
		room.mu.Lock()
		if room.Phase != PhaseFinished && roomHasHumanUser(room, userID) && (!found || room.UpdatedAt.After(currentUpdatedAt)) {
			current = publicRoom(room, userID)
			currentUpdatedAt = room.UpdatedAt
			found = true
		}
		room.mu.Unlock()
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

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	m.mu.RLock()
	defer m.mu.RUnlock()
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}

func (m *Manager) lockRoom(roomID string) (*Room, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, err
	}
	room.mu.Lock()
	return room, nil
}
