package gomoku

import (
	"errors"
	"time"
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
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need_two_players")
	}

	resetBoard(room)
	room.Phase = PhasePlaying
	room.CurrentPlayerIndex = 0
	room.Players[0].Stone = StoneBlack
	room.Players[1].Stone = StoneWhite
	room.Log = append(room.Log, createLog("五子棋开始，黑棋先行。"))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Place(roomID string, actorID string, x int, y int) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !inBounds(x, y) {
		return PublicRoom{}, errors.New("invalid_position")
	}
	if room.Board[y][x] != "" {
		return PublicRoom{}, errors.New("position_occupied")
	}

	placeStone(room, player, x, y)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}
