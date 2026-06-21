package uno

import (
	"context"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

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

func TestStackingAccumulatesDrawPenalty(t *testing.T) {
	room := &Room{
		Rules:              RuleSet{Stacking: true},
		Phase:              PhasePlaying,
		Players:            []*Player{{ID: "player-1", UserID: "user-1", Name: "一号", Hand: []Card{{ID: "green-draw-two", Color: ColorGreen, Kind: KindDrawTwo}, numberCard("red-1", ColorRed, 1)}}, {ID: "player-2", UserID: "user-2", Name: "二号"}},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{{ID: "green-2", Color: ColorGreen, Kind: KindDrawTwo}},
	}

	playCard(room, room.Players[0], 0, ColorGreen)

	if room.PendingDrawCount != 2 {
		t.Fatalf("pending draw count = %d, want 2", room.PendingDrawCount)
	}
	if room.CurrentPlayerIndex != 1 {
		t.Fatalf("current player index = %d, want next player to answer stack", room.CurrentPlayerIndex)
	}
}

func TestSevenZeroSwapsHandsOnSeven(t *testing.T) {
	currentHand := []Card{numberCard("green-7", ColorGreen, 7), numberCard("red-1", ColorRed, 1)}
	targetHand := []Card{numberCard("blue-3", ColorBlue, 3)}
	room := &Room{
		Rules:              RuleSet{SevenZero: true},
		Phase:              PhasePlaying,
		Players:            []*Player{{ID: "player-1", UserID: "user-1", Name: "一号", Hand: currentHand}, {ID: "player-2", UserID: "user-2", Name: "二号", Hand: targetHand}},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{numberCard("green-1", ColorGreen, 1)},
	}

	playCard(room, room.Players[0], 0, ColorGreen)

	if room.Players[0].Hand[0].ID != "blue-3" {
		t.Fatalf("current player hand = %v, want swapped target hand", room.Players[0].Hand)
	}
	if room.Players[1].Hand[0].ID != "red-1" {
		t.Fatalf("target player hand = %v, want remaining current hand", room.Players[1].Hand)
	}
}

func TestJumpInOnlyAllowsExactSameFace(t *testing.T) {
	player := &Player{
		ID:     "player-2",
		UserID: "user-2",
		Name:   "二号",
		Hand: []Card{
			numberCard("green-1-copy", ColorGreen, 1),
			numberCard("green-2", ColorGreen, 2),
		},
	}
	room := &Room{
		Rules:              RuleSet{JumpIn: true},
		Phase:              PhasePlaying,
		Players:            []*Player{{ID: "player-1", UserID: "user-1", Name: "一号"}, player},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{numberCard("green-1", ColorGreen, 1)},
	}

	if !canJumpIn(player.Hand[0], player, room) {
		t.Fatal("exact same face should be allowed to jump in")
	}
	if canJumpIn(player.Hand[1], player, room) {
		t.Fatal("matching color with different value must not jump in")
	}
}

func TestTickAutoActsAfterTurnTimeout(t *testing.T) {
	now := time.Now().UTC()
	deadline := now.Add(-time.Second)
	manager := NewManager(nil)
	manager.rooms["ROOMT"] = &Room{
		ID:    "ROOMT",
		Phase: PhasePlaying,
		Players: []*Player{
			{
				ID:        "player-1",
				UserID:    "user-1",
				Name:      "一号",
				Connected: true,
				Hand: []Card{
					numberCard("green-5", ColorGreen, 5),
					numberCard("red-1", ColorRed, 1),
				},
			},
			{ID: "player-2", UserID: "user-2", Name: "二号", Connected: true},
		},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{numberCard("green-1", ColorGreen, 1)},
		TurnDeadline:       &deadline,
	}

	result := manager.Tick(now)

	if len(result.BroadcastRoomIDs) == 0 {
		t.Fatal("tick should request a room broadcast after timeout")
	}
	room := manager.rooms["ROOMT"]
	if room.CurrentPlayerIndex != 1 {
		t.Fatalf("current player index = %d, want timeout to advance to player 2", room.CurrentPlayerIndex)
	}
	if len(room.Players[0].Hand) != 1 {
		t.Fatalf("timed out player hand size = %d, want 1 after auto play", len(room.Players[0].Hand))
	}
	if got := room.DiscardPile[len(room.DiscardPile)-1].ID; got != "green-5" {
		t.Fatalf("top discard = %q, want first legal card green-5", got)
	}
	if room.TurnDeadline == nil || !room.TurnDeadline.After(now) {
		t.Fatal("turn deadline should be refreshed after timeout action")
	}
}

func TestTickDestroysRoomAfterAllHumansOffline(t *testing.T) {
	now := time.Now().UTC()
	offlineSince := now.Add(-offlineRoomTTL - time.Second)
	manager := NewManager(nil)
	manager.rooms["ROOMX"] = &Room{
		ID:    "ROOMX",
		Phase: PhaseLobby,
		Players: []*Player{
			{ID: "player-1", UserID: "user-1", Name: "一号", Connected: false},
			{ID: "ai-1", UserID: "ai-user-1", Name: "AI", IsAI: true, Connected: true},
		},
		AllHumansOfflineSince: &offlineSince,
	}

	result := manager.Tick(now)

	if len(result.DestroyedRoomIDs) != 1 || result.DestroyedRoomIDs[0] != "ROOMX" {
		t.Fatalf("destroyed rooms = %v, want ROOMX", result.DestroyedRoomIDs)
	}
	if _, err := manager.Public("ROOMX", "user-1"); err == nil {
		t.Fatal("destroyed room should no longer be public")
	}
}

func TestLLMAITurnDoesNotBlockOtherRooms(t *testing.T) {
	provider := &blockingProvider{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	manager := NewManager(provider)
	manager.rooms["ROOMA"] = &Room{
		ID:    "ROOMA",
		Phase: PhasePlaying,
		Players: []*Player{
			{
				ID:     "ai-1",
				UserID: "ai-user-1",
				Name:   "AI 一号",
				IsAI:   true,
				AI:     &AIProfile{Level: string(aiplayer.LevelLLM)},
				Hand:   []Card{numberCard("ai-green-1", ColorGreen, 1)},
			},
			{ID: "human-1", UserID: "human-user-1", Name: "玩家一号"},
		},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorGreen,
		DiscardPile:        []Card{numberCard("top-green-2", ColorGreen, 2)},
		DrawPile:           []Card{numberCard("draw-red-1", ColorRed, 1)},
	}
	manager.rooms["ROOMB"] = &Room{
		ID:    "ROOMB",
		Phase: PhasePlaying,
		Players: []*Player{
			{ID: "human-2", UserID: "human-user-2", Name: "玩家二号"},
			{ID: "human-3", UserID: "human-user-3", Name: "玩家三号"},
		},
		CurrentPlayerIndex: 0,
		Direction:          1,
		ActiveColor:        ColorBlue,
		DiscardPile:        []Card{numberCard("top-blue-3", ColorBlue, 3)},
	}

	done := make(chan error, 1)
	go func() {
		_, _, err := manager.RunAIAction("ROOMA")
		done <- err
	}()

	select {
	case <-provider.started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("llm provider was not called")
	}

	publicDone := make(chan error, 1)
	go func() {
		_, err := manager.Public("ROOMB", "human-user-2")
		publicDone <- err
	}()

	select {
	case err := <-publicDone:
		if err != nil {
			t.Fatalf("public room: %v", err)
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("room B public view was blocked by room A llm turn")
	}

	close(provider.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run ai: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ai turn did not finish")
	}
}

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p *blockingProvider) Enabled() bool {
	return true
}

func (p *blockingProvider) Decide(ctx context.Context, input aiplayer.DecisionInput) (aiplayer.Decision, error) {
	p.started <- struct{}{}
	select {
	case <-p.release:
		return aiplayer.Decision{ActionID: input.Actions[0].ID, Source: "test"}, nil
	case <-ctx.Done():
		return aiplayer.Decision{}, ctx.Err()
	}
}

func numberCard(id string, color Color, value int) Card {
	return Card{ID: id, Color: color, Kind: KindNumber, Value: &value}
}
