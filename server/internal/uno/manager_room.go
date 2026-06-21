package uno

import (
	"errors"
	"strings"
)

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	return publicRoom(room, viewerID), nil
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
