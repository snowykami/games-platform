package gomoku

import (
	"fmt"
	"time"
)

func resetBoard(room *Room) {
	room.Board = [BoardSize][BoardSize]Stone{}
	room.Moves = nil
	room.WinnerID = ""
	room.WinningLine = nil
	room.IsDraw = false
	room.RecentActions = nil
}

func placeStone(room *Room, player *Player, x int, y int) {
	room.Board[y][x] = player.Stone
	move := Move{
		X:          x,
		Y:          y,
		Stone:      player.Stone,
		PlayerID:   player.ID,
		PlayerName: player.Name,
		PlacedAt:   time.Now().UTC(),
	}
	room.Moves = append(room.Moves, move)

	message := fmt.Sprintf("%s 落子 %s。", player.Name, formatPoint(x, y))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{
		Type:      ActionPlace,
		ActorID:   player.ID,
		ActorName: player.Name,
		X:         x,
		Y:         y,
		Stone:     player.Stone,
		Message:   message,
	})

	if line := winningLine(room, x, y, player.Stone); len(line) >= 5 {
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		room.WinningLine = line
		winMessage := fmt.Sprintf("%s 五连获胜。", player.Name)
		room.Log = append(room.Log, createLog(winMessage))
		recordAction(room, PublicAction{
			Type:      ActionWin,
			ActorID:   player.ID,
			ActorName: player.Name,
			Stone:     player.Stone,
			Message:   winMessage,
		})
		return
	}

	if len(room.Moves) == BoardSize*BoardSize {
		room.Phase = PhaseFinished
		room.IsDraw = true
		drawMessage := "棋盘已满，平局。"
		room.Log = append(room.Log, createLog(drawMessage))
		recordAction(room, PublicAction{
			Type:      ActionDraw,
			ActorID:   player.ID,
			ActorName: player.Name,
			Message:   drawMessage,
		})
		return
	}

	room.CurrentPlayerIndex = (room.CurrentPlayerIndex + 1) % len(room.Players)
}

func winningLine(room *Room, x int, y int, stone Stone) []Point {
	for _, direction := range directions {
		line := []Point{{X: x, Y: y}}
		line = append(collectLine(room, x, y, direction.X, direction.Y, stone), line...)
		line = append(line, collectLine(room, x, y, -direction.X, -direction.Y, stone)...)
		if len(line) >= 5 {
			return line
		}
	}
	return nil
}

func collectLine(room *Room, x int, y int, dx int, dy int, stone Stone) []Point {
	line := []Point{}
	for step := 1; step < 5; step++ {
		nextX := x + dx*step
		nextY := y + dy*step
		if !inBounds(nextX, nextY) || room.Board[nextY][nextX] != stone {
			break
		}
		line = append(line, Point{X: nextX, Y: nextY})
	}
	return line
}

func inBounds(x int, y int) bool {
	return x >= 0 && x < BoardSize && y >= 0 && y < BoardSize
}

func formatPoint(x int, y int) string {
	return fmt.Sprintf("%c%d", 'A'+rune(x), y+1)
}
