package xiangqi

import "fmt"

func formatMoveMessage(player *Player, move Move) string {
	captureText := ""
	if move.Captured != nil {
		captureText = fmt.Sprintf("，吃掉%s", formatPiece(*move.Captured))
	}
	checkText := ""
	if move.Check {
		checkText = "，将军"
	}
	return fmt.Sprintf("%s %s 从 %s 到 %s%s%s。", player.Name, formatPiece(Piece{Side: move.Side, Type: move.PieceType}), formatPosition(move.From), formatPosition(move.To), captureText, checkText)
}

func formatPiece(piece Piece) string {
	labels := map[Side]map[PieceType]string{
		SideRed:   {PieceGeneral: "帥", PieceAdvisor: "仕", PieceElephant: "相", PieceHorse: "傌", PieceRook: "俥", PieceCannon: "炮", PieceSoldier: "兵"},
		SideBlack: {PieceGeneral: "將", PieceAdvisor: "士", PieceElephant: "象", PieceHorse: "馬", PieceRook: "車", PieceCannon: "砲", PieceSoldier: "卒"},
	}
	return labels[piece.Side][piece.Type]
}

func formatPosition(position Position) string {
	return fmt.Sprintf("%d路%d线", position.X+1, position.Y+1)
}
