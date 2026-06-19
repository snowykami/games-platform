package uno

import "testing"

func TestIsPlayableRejectsDifferentNumberValue(t *testing.T) {
	room := &Room{
		ActiveColor: ColorGreen,
		DiscardPile: []Card{
			numberCard("green-1", ColorGreen, 1),
		},
	}

	if isPlayable(numberCard("blue-5", ColorBlue, 5), room) {
		t.Fatal("blue 5 must not be playable on green 1")
	}
}

func TestIsPlayableAllowsMatchingColorNumberValueAndActionKind(t *testing.T) {
	room := &Room{
		ActiveColor: ColorGreen,
		DiscardPile: []Card{
			numberCard("green-1", ColorGreen, 1),
		},
	}

	if !isPlayable(numberCard("green-5", ColorGreen, 5), room) {
		t.Fatal("matching active color should be playable")
	}
	if !isPlayable(numberCard("blue-1", ColorBlue, 1), room) {
		t.Fatal("matching number value should be playable")
	}

	room.DiscardPile = []Card{{ID: "green-skip", Color: ColorGreen, Kind: KindSkip}}
	if !isPlayable(Card{ID: "blue-skip", Color: ColorBlue, Kind: KindSkip}, room) {
		t.Fatal("matching action kind should be playable")
	}
}

func TestPublicRoomReturnsPlayableCardIDsForCurrentViewerOnly(t *testing.T) {
	viewer := &Player{
		ID:     "player-1",
		UserID: "user-1",
		Name:   "玩家",
		Hand: []Card{
			numberCard("green-5", ColorGreen, 5),
			numberCard("blue-1", ColorBlue, 1),
			numberCard("blue-5", ColorBlue, 5),
		},
	}
	room := &Room{
		Phase:              PhasePlaying,
		Players:            []*Player{viewer, &Player{ID: "player-2", UserID: "user-2", Name: "对手"}},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{numberCard("green-1", ColorGreen, 1)},
	}

	got := publicRoom(room, "user-1")
	want := map[string]bool{"green-5": true, "blue-1": true}
	if len(got.PlayableCardIDs) != len(want) {
		t.Fatalf("playable card ids = %v, want two playable cards", got.PlayableCardIDs)
	}
	for _, id := range got.PlayableCardIDs {
		if !want[id] {
			t.Fatalf("unexpected playable card id %q from %v", id, got.PlayableCardIDs)
		}
	}

	otherViewer := publicRoom(room, "user-2")
	if len(otherViewer.PlayableCardIDs) != 0 {
		t.Fatalf("non-current viewer playable card ids = %v, want empty", otherViewer.PlayableCardIDs)
	}
}

func numberCard(id string, color Color, value int) Card {
	return Card{ID: id, Color: color, Kind: KindNumber, Value: &value}
}
