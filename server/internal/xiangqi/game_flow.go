package xiangqi

import (
	"fmt"
	"time"
)

func resetGame(room *Room) {
	room.Pieces = initialPieces()
	room.Moves = nil
	room.WinnerID = ""
	room.CheckSide = ""
	room.RecentActions = nil
	for _, player := range room.Players {
		player.Side = ""
	}
}

func applyPlayerMove(room *Room, player *Player, piece Piece, to Position) {
	from := Position{X: piece.X, Y: piece.Y}
	captured, hasCaptured := pieceAt(room.Pieces, to)
	nextPieces := applyMove(room.Pieces, piece, to)
	opponent := oppositeSide(piece.Side)
	check := sideInCheck(nextPieces, opponent)
	checkmate := !hasAnyLegalMove(nextPieces, opponent)
	move := Move{
		ID:         "mov_" + randomToken(8),
		PieceID:    piece.ID,
		PieceType:  piece.Type,
		Side:       piece.Side,
		From:       from,
		To:         to,
		Check:      check,
		Checkmate:  checkmate,
		PlayerID:   player.ID,
		PlayerName: player.Name,
		PlayedAt:   time.Now().UTC(),
	}
	if hasCaptured {
		move.Captured = &captured
	}

	room.Pieces = nextPieces
	room.Moves = append(room.Moves, move)
	room.CheckSide = ""
	if check {
		room.CheckSide = opponent
	}

	message := formatMoveMessage(player, move)
	room.Log = append(room.Log, createLog(message))
	actionType := ActionMove
	if hasCaptured {
		actionType = ActionCapture
	}
	if check {
		actionType = ActionCheck
	}
	if hasCaptured && captured.Type == PieceGeneral || checkmate {
		actionType = ActionCheckmate
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		message = fmt.Sprintf("%s 获胜。", player.Name)
		room.Log = append(room.Log, createLog(message))
	}
	recordAction(room, PublicAction{
		Type:      actionType,
		ActorID:   player.ID,
		ActorName: player.Name,
		Move:      &move,
		Message:   message,
	})

	if room.Phase == PhasePlaying {
		room.CurrentPlayerIndex = nextPlayerIndex(room, player.Side)
	}
}
