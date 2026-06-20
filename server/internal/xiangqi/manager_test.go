package xiangqi

import "testing"

func TestManagerStartsRoomAndMovesPiece(t *testing.T) {
	manager := NewManager(nil)
	host := UserView{ID: "u1", DisplayName: "红方", Kind: "guest"}
	guest := UserView{ID: "u2", DisplayName: "黑方", Kind: "guest"}
	room := manager.CreateRoom(host)

	if _, err := manager.JoinRoom(room.ID, guest); err != nil {
		t.Fatalf("join room: %v", err)
	}

	started, err := manager.Start(room.ID, host.ID)
	if err != nil {
		t.Fatalf("start room: %v", err)
	}
	if started.Phase != PhasePlaying {
		t.Fatalf("phase = %s, want playing", started.Phase)
	}
	if len(started.Pieces) != 32 {
		t.Fatalf("pieces = %d, want 32", len(started.Pieces))
	}

	next, err := manager.Move(room.ID, host.ID, "red-soldier-0-6", Position{X: 0, Y: 5})
	if err != nil {
		t.Fatalf("move soldier: %v", err)
	}
	if next.CurrentPlayerID == started.CurrentPlayerID {
		t.Fatalf("turn did not advance")
	}
}

func TestManagerRejectsIllegalMove(t *testing.T) {
	manager := NewManager(nil)
	host := UserView{ID: "u1", DisplayName: "红方", Kind: "guest"}
	guest := UserView{ID: "u2", DisplayName: "黑方", Kind: "guest"}
	room := manager.CreateRoom(host)
	_, _ = manager.JoinRoom(room.ID, guest)
	_, _ = manager.Start(room.ID, host.ID)

	if _, err := manager.Move(room.ID, host.ID, "red-soldier-0-6", Position{X: 0, Y: 7}); err == nil {
		t.Fatal("illegal backward soldier move succeeded")
	}
}

func TestManagerRunsAI(t *testing.T) {
	manager := NewManager(nil)
	host := UserView{ID: "u1", DisplayName: "红方", Kind: "guest"}
	room := manager.CreateRoom(host)
	if _, err := manager.AddAI(room.ID, host.ID, AIOptions{}); err != nil {
		t.Fatalf("add ai: %v", err)
	}
	if _, err := manager.Start(room.ID, host.ID); err != nil {
		t.Fatalf("start room: %v", err)
	}
	if _, err := manager.Move(room.ID, host.ID, "red-soldier-0-6", Position{X: 0, Y: 5}); err != nil {
		t.Fatalf("human move: %v", err)
	}

	next, _, err := manager.RunNextAI(room.ID)
	if err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if len(next.Moves) != 2 {
		t.Fatalf("moves = %d, want 2", len(next.Moves))
	}
}
