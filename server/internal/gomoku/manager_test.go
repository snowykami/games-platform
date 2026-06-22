package gomoku

import "testing"

func TestPlaceWinsWithFiveInRow(t *testing.T) {
	manager := NewManager(nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "黑棋", Kind: "guest"})
	if _, err := manager.JoinRoom(room.ID, UserView{ID: "u2", DisplayName: "白棋", Kind: "guest"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(room.ID, "u1"); err != nil {
		t.Fatal(err)
	}

	moves := []struct {
		userID string
		x      int
		y      int
	}{
		{"u1", 0, 0},
		{"u2", 0, 1},
		{"u1", 1, 0},
		{"u2", 1, 1},
		{"u1", 2, 0},
		{"u2", 2, 1},
		{"u1", 3, 0},
		{"u2", 3, 1},
		{"u1", 4, 0},
	}

	var next PublicRoom
	for _, move := range moves {
		updated, err := manager.Place(room.ID, move.userID, move.x, move.y)
		if err != nil {
			t.Fatal(err)
		}
		next = updated
	}

	if next.Phase != PhaseFinished {
		t.Fatalf("expected finished phase, got %s", next.Phase)
	}
	if next.WinnerID == "" {
		t.Fatal("expected winner")
	}
	if len(next.WinningLine) != 5 {
		t.Fatalf("expected five winning points, got %d", len(next.WinningLine))
	}
}

func TestRejectsOccupiedPoint(t *testing.T) {
	manager := NewManager(nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "黑棋", Kind: "guest"})
	if _, err := manager.JoinRoom(room.ID, UserView{ID: "u2", DisplayName: "白棋", Kind: "guest"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(room.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Place(room.ID, "u1", 7, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Place(room.ID, "u2", 7, 7); err == nil {
		t.Fatal("expected occupied point error")
	}
}

func TestAIChoosesImmediateWinningMove(t *testing.T) {
	room := &Room{}
	for x := 3; x <= 6; x++ {
		room.Board[7][x] = StoneBlack
		room.Moves = append(room.Moves, Move{X: x, Y: 7, Stone: StoneBlack})
	}

	x, y := chooseAIMove(room, StoneBlack, "master")
	if y != 7 || (x != 2 && x != 7) {
		t.Fatalf("expected AI to finish open four, got %s", formatPoint(x, y))
	}
}

func TestAIBlocksImmediateOpponentWin(t *testing.T) {
	room := &Room{}
	for x := 3; x <= 6; x++ {
		room.Board[7][x] = StoneWhite
		room.Moves = append(room.Moves, Move{X: x, Y: 7, Stone: StoneWhite})
	}
	room.Board[8][8] = StoneBlack
	room.Moves = append(room.Moves, Move{X: 8, Y: 8, Stone: StoneBlack})

	x, y := chooseAIMove(room, StoneBlack, "master")
	if y != 7 || (x != 2 && x != 7) {
		t.Fatalf("expected AI to block opponent open four, got %s", formatPoint(x, y))
	}
}
