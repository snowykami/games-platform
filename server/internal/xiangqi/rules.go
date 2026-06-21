package xiangqi

func legalMoves(pieces []Piece, piece Piece) []Position {
	moves := pseudoMoves(pieces, piece)
	legal := []Position{}
	for _, move := range moves {
		target, occupied := pieceAt(pieces, move)
		if occupied && target.Side == piece.Side {
			continue
		}
		if !sideInCheck(applyMove(pieces, piece, move), piece.Side) {
			legal = append(legal, move)
		}
	}
	return legal
}

func allLegalMoves(pieces []Piece, side Side) []struct {
	piece Piece
	to    Position
} {
	moves := []struct {
		piece Piece
		to    Position
	}{}
	for _, piece := range pieces {
		if piece.Side != side {
			continue
		}
		for _, to := range legalMoves(pieces, piece) {
			moves = append(moves, struct {
				piece Piece
				to    Position
			}{piece: piece, to: to})
		}
	}
	return moves
}

func pseudoMoves(pieces []Piece, piece Piece) []Position {
	switch piece.Type {
	case PieceGeneral:
		return generalMoves(pieces, piece)
	case PieceAdvisor:
		return advisorMoves(piece)
	case PieceElephant:
		return elephantMoves(pieces, piece)
	case PieceHorse:
		return horseMoves(pieces, piece)
	case PieceRook:
		return slidingMoves(pieces, piece, false)
	case PieceCannon:
		return slidingMoves(pieces, piece, true)
	default:
		return soldierMoves(piece)
	}
}

func generalMoves(pieces []Piece, piece Piece) []Position {
	moves := filterPositions([]Position{
		{X: piece.X + 1, Y: piece.Y},
		{X: piece.X - 1, Y: piece.Y},
		{X: piece.X, Y: piece.Y + 1},
		{X: piece.X, Y: piece.Y - 1},
	}, func(position Position) bool {
		return insidePalace(position, piece.Side)
	})

	for _, other := range pieces {
		if other.Side != piece.Side && other.Type == PieceGeneral && other.X == piece.X && piecesBetween(pieces, Position{X: piece.X, Y: piece.Y}, Position{X: other.X, Y: other.Y}) == 0 {
			moves = append(moves, Position{X: other.X, Y: other.Y})
		}
	}
	return moves
}

func advisorMoves(piece Piece) []Position {
	return filterPositions([]Position{
		{X: piece.X + 1, Y: piece.Y + 1},
		{X: piece.X + 1, Y: piece.Y - 1},
		{X: piece.X - 1, Y: piece.Y + 1},
		{X: piece.X - 1, Y: piece.Y - 1},
	}, func(position Position) bool {
		return insidePalace(position, piece.Side)
	})
}

func elephantMoves(pieces []Piece, piece Piece) []Position {
	return filterPositions([]Position{
		{X: piece.X + 2, Y: piece.Y + 2},
		{X: piece.X + 2, Y: piece.Y - 2},
		{X: piece.X - 2, Y: piece.Y + 2},
		{X: piece.X - 2, Y: piece.Y - 2},
	}, func(position Position) bool {
		eye := Position{X: piece.X + (position.X-piece.X)/2, Y: piece.Y + (position.Y-piece.Y)/2}
		_, blocked := pieceAt(pieces, eye)
		return insideBoard(position) && ownRiverSide(position, piece.Side) && !blocked
	})
}

func horseMoves(pieces []Piece, piece Piece) []Position {
	offsets := []Position{{X: 1, Y: 2}, {X: 2, Y: 1}, {X: -1, Y: 2}, {X: -2, Y: 1}, {X: 1, Y: -2}, {X: 2, Y: -1}, {X: -1, Y: -2}, {X: -2, Y: -1}}
	moves := []Position{}
	for _, offset := range offsets {
		leg := Position{X: piece.X, Y: piece.Y + offset.Y/2}
		if abs(offset.X) == 2 {
			leg = Position{X: piece.X + offset.X/2, Y: piece.Y}
		}
		position := Position{X: piece.X + offset.X, Y: piece.Y + offset.Y}
		if _, blocked := pieceAt(pieces, leg); insideBoard(position) && !blocked {
			moves = append(moves, position)
		}
	}
	return moves
}

func slidingMoves(pieces []Piece, piece Piece, cannon bool) []Position {
	moves := []Position{}
	for _, direction := range slidingDirections {
		position := Position{X: piece.X + direction.X, Y: piece.Y + direction.Y}
		screenFound := false
		for insideBoard(position) {
			_, occupied := pieceAt(pieces, position)
			if !cannon {
				moves = append(moves, position)
				if occupied {
					break
				}
			} else if !screenFound {
				if occupied {
					screenFound = true
				} else {
					moves = append(moves, position)
				}
			} else if occupied {
				moves = append(moves, position)
				break
			}
			position = Position{X: position.X + direction.X, Y: position.Y + direction.Y}
		}
	}
	return moves
}

func soldierMoves(piece Piece) []Position {
	forward := 1
	if piece.Side == SideRed {
		forward = -1
	}
	moves := []Position{{X: piece.X, Y: piece.Y + forward}}
	if crossedRiver(piece) {
		moves = append(moves, Position{X: piece.X + 1, Y: piece.Y}, Position{X: piece.X - 1, Y: piece.Y})
	}
	return filterPositions(moves, insideBoard)
}

func sideInCheck(pieces []Piece, side Side) bool {
	general, ok := findGeneral(pieces, side)
	if !ok {
		return true
	}
	generalPosition := Position{X: general.X, Y: general.Y}
	for _, piece := range pieces {
		if piece.Side == side {
			continue
		}
		if containsPosition(pseudoMoves(pieces, piece), generalPosition) {
			return true
		}
	}
	return false
}

func hasAnyLegalMove(pieces []Piece, side Side) bool {
	return len(allLegalMoves(pieces, side)) > 0
}

func applyMove(pieces []Piece, piece Piece, to Position) []Piece {
	next := []Piece{}
	for _, item := range pieces {
		if item.ID == piece.ID || samePosition(Position{X: item.X, Y: item.Y}, to) {
			continue
		}
		next = append(next, item)
	}
	piece.X = to.X
	piece.Y = to.Y
	next = append(next, piece)
	return next
}

func pieceAt(pieces []Piece, position Position) (Piece, bool) {
	for _, piece := range pieces {
		if piece.X == position.X && piece.Y == position.Y {
			return piece, true
		}
	}
	return Piece{}, false
}

func findPiece(pieces []Piece, id string) (Piece, bool) {
	for _, piece := range pieces {
		if piece.ID == id {
			return piece, true
		}
	}
	return Piece{}, false
}

func findGeneral(pieces []Piece, side Side) (Piece, bool) {
	for _, piece := range pieces {
		if piece.Side == side && piece.Type == PieceGeneral {
			return piece, true
		}
	}
	return Piece{}, false
}

func filterPositions(positions []Position, keep func(Position) bool) []Position {
	filtered := []Position{}
	for _, position := range positions {
		if keep(position) {
			filtered = append(filtered, position)
		}
	}
	return filtered
}

func piecesBetween(pieces []Piece, from Position, to Position) int {
	stepX := sign(to.X - from.X)
	stepY := sign(to.Y - from.Y)
	position := Position{X: from.X + stepX, Y: from.Y + stepY}
	count := 0
	for !samePosition(position, to) {
		if _, ok := pieceAt(pieces, position); ok {
			count++
		}
		position = Position{X: position.X + stepX, Y: position.Y + stepY}
	}
	return count
}

func nextPlayerIndex(room *Room, side Side) int {
	nextSide := oppositeSide(side)
	for index, player := range room.Players {
		if player.Side == nextSide {
			return index
		}
	}
	return 0
}

func opponentPlayer(room *Room, side Side) *Player {
	nextSide := oppositeSide(side)
	for _, player := range room.Players {
		if player.Side == nextSide {
			return player
		}
	}
	return room.Players[0]
}

func containsPosition(positions []Position, target Position) bool {
	for _, position := range positions {
		if samePosition(position, target) {
			return true
		}
	}
	return false
}

func insideBoard(position Position) bool {
	return position.X >= 0 && position.X < BoardWidth && position.Y >= 0 && position.Y < BoardHeight
}

func insidePalace(position Position, side Side) bool {
	return position.X >= 3 && position.X <= 5 && ((side == SideRed && position.Y >= 7 && position.Y <= 9) || (side == SideBlack && position.Y >= 0 && position.Y <= 2))
}

func ownRiverSide(position Position, side Side) bool {
	return side == SideRed && position.Y >= 5 || side == SideBlack && position.Y <= 4
}

func crossedRiver(piece Piece) bool {
	return piece.Side == SideRed && piece.Y <= 4 || piece.Side == SideBlack && piece.Y >= 5
}

func samePosition(first Position, second Position) bool {
	return first.X == second.X && first.Y == second.Y
}

func oppositeSide(side Side) Side {
	if side == SideRed {
		return SideBlack
	}
	return SideRed
}

func pieceValue(pieceType PieceType) float64 {
	values := map[PieceType]float64{
		PieceAdvisor:  2,
		PieceCannon:   4.5,
		PieceElephant: 2,
		PieceGeneral:  1000,
		PieceHorse:    4,
		PieceRook:     9,
		PieceSoldier:  1.5,
	}
	return values[pieceType]
}

func sign(value int) int {
	if value > 0 {
		return 1
	}
	if value < 0 {
		return -1
	}
	return 0
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
