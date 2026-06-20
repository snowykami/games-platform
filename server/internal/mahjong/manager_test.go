package mahjong

import "testing"

func TestStartDealsFourPlayersAndHidesOpponentHands(t *testing.T) {
	manager := NewManager(nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "东家", Kind: "guest"})

	for _, user := range []UserView{
		{ID: "u2", DisplayName: "南家", Kind: "guest"},
		{ID: "u3", DisplayName: "西家", Kind: "guest"},
		{ID: "u4", DisplayName: "北家", Kind: "guest"},
	} {
		if _, err := manager.JoinRoom(room.ID, user); err != nil {
			t.Fatalf("join room: %v", err)
		}
	}

	started, err := manager.Start(room.ID, "u1")
	if err != nil {
		t.Fatalf("start room: %v", err)
	}

	if started.Phase != PhasePlaying {
		t.Fatalf("phase = %s, want %s", started.Phase, PhasePlaying)
	}
	if len(started.Players[0].Hand) != 14 {
		t.Fatalf("host hand = %d, want 14", len(started.Players[0].Hand))
	}
	if started.Players[1].HandCount != 13 || len(started.Players[1].Hand) != 0 {
		t.Fatalf("opponent hand visibility mismatch: count=%d visible=%d", started.Players[1].HandCount, len(started.Players[1].Hand))
	}
	if started.WallCount != 69 {
		t.Fatalf("wall count = %d, want 69", started.WallCount)
	}

	viewForSouth, err := manager.Public(room.ID, "u2")
	if err != nil {
		t.Fatalf("public room: %v", err)
	}
	if len(viewForSouth.Players[0].Hand) != 0 || len(viewForSouth.Players[1].Hand) != 13 {
		t.Fatalf("viewer hand filtering failed")
	}
}

func TestDiscardAdvancesToNextDraw(t *testing.T) {
	manager := NewManager(nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "东家", Kind: "guest"})

	for _, user := range []UserView{
		{ID: "u2", DisplayName: "南家", Kind: "guest"},
		{ID: "u3", DisplayName: "西家", Kind: "guest"},
		{ID: "u4", DisplayName: "北家", Kind: "guest"},
	} {
		if _, err := manager.JoinRoom(room.ID, user); err != nil {
			t.Fatalf("join room: %v", err)
		}
	}
	started, err := manager.Start(room.ID, "u1")
	if err != nil {
		t.Fatalf("start room: %v", err)
	}
	tileID := started.Players[0].Hand[0].ID

	afterDiscard, err := manager.Discard(room.ID, "u1", tileID)
	if err != nil {
		t.Fatalf("discard: %v", err)
	}
	if afterDiscard.Phase == PhaseClaiming {
		skipped := map[string]bool{}
		for _, option := range manager.rooms[room.ID].ClaimOptions {
			player := findPlayerByID(manager.rooms[room.ID], option.PlayerID)
			if player != nil && !skipped[player.UserID] {
				afterDiscard, err = manager.SkipClaims(room.ID, player.UserID)
				if err != nil {
					t.Fatalf("skip claims: %v", err)
				}
				skipped[player.UserID] = true
			}
		}
	}
	if afterDiscard.CurrentPlayerID != afterDiscard.Players[1].ID {
		t.Fatalf("current player = %s, want %s", afterDiscard.CurrentPlayerID, afterDiscard.Players[1].ID)
	}
	if afterDiscard.HasDrawn {
		t.Fatalf("next player should need to draw")
	}

	afterDraw, err := manager.Draw(room.ID, "u2")
	if err != nil {
		t.Fatalf("draw: %v", err)
	}
	if !afterDraw.HasDrawn || afterDraw.Players[1].HandCount != 14 {
		t.Fatalf("draw state mismatch: hasDrawn=%v handCount=%d", afterDraw.HasDrawn, afterDraw.Players[1].HandCount)
	}
}

func TestNeedsFourPlayersToStart(t *testing.T) {
	manager := NewManager(nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "东家", Kind: "guest"})

	if _, err := manager.Start(room.ID, "u1"); err == nil {
		t.Fatalf("start with one player succeeded")
	}
}
