package xiangqi

import "fmt"

func initialPieces() []Piece {
	pieces := []Piece{}
	pieces = appendBackRank(pieces, SideBlack, 0)
	pieces = appendBackRank(pieces, SideRed, 9)
	pieces = appendPieces(pieces, SideBlack, PieceCannon, 2, []int{1, 7})
	pieces = appendPieces(pieces, SideRed, PieceCannon, 7, []int{1, 7})
	pieces = appendPieces(pieces, SideBlack, PieceSoldier, 3, []int{0, 2, 4, 6, 8})
	pieces = appendPieces(pieces, SideRed, PieceSoldier, 6, []int{0, 2, 4, 6, 8})
	return pieces
}

func appendBackRank(pieces []Piece, side Side, y int) []Piece {
	order := []PieceType{PieceRook, PieceHorse, PieceElephant, PieceAdvisor, PieceGeneral, PieceAdvisor, PieceElephant, PieceHorse, PieceRook}
	for x, pieceType := range order {
		pieces = append(pieces, createPiece(side, pieceType, x, y))
	}
	return pieces
}

func appendPieces(pieces []Piece, side Side, pieceType PieceType, y int, files []int) []Piece {
	for _, x := range files {
		pieces = append(pieces, createPiece(side, pieceType, x, y))
	}
	return pieces
}

func createPiece(side Side, pieceType PieceType, x int, y int) Piece {
	return Piece{ID: fmt.Sprintf("%s-%s-%d-%d", side, pieceType, x, y), Side: side, Type: pieceType, X: x, Y: y}
}
