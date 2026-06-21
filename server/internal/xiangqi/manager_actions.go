package xiangqi

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

	resetGame(room)
	room.Phase = PhasePlaying
	room.CurrentPlayerIndex = 0
	room.Players[0].Side = SideRed
	room.Players[1].Side = SideBlack
	room.Log = append(room.Log, createLog("象棋开始，红方先行。"))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Move(roomID string, actorID string, pieceID string, to Position) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !insideBoard(to) {
		return PublicRoom{}, errors.New("invalid_position")
	}

	piece, ok := findPiece(room.Pieces, pieceID)
	if !ok {
		return PublicRoom{}, errors.New("piece_not_found")
	}
	if piece.Side != player.Side {
		return PublicRoom{}, errors.New("piece_wrong_side")
	}
	if !containsPosition(legalMoves(room.Pieces, piece), to) {
		return PublicRoom{}, errors.New("xiangqi_illegal_move")
	}

	applyPlayerMove(room, player, piece, to)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}
