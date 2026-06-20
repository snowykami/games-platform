package socialdeduction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func TestPublicRoomHidesWerewolfRoles(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := &Room{
		ID:         "WWFTEST",
		Game:       GameWerewolf,
		HostUserID: "u1",
		Phase:      PhaseWerewolfNight,
		Players: []*Player{
			testPlayer("p1", "u1", "Host", RoleVillager, AlignmentGood),
			testPlayer("p2", "u2", "Wolf One", RoleWerewolf, AlignmentEvil),
			testPlayer("p3", "u3", "Wolf Two", RoleWerewolf, AlignmentEvil),
		},
		Werewolf: WerewolfState{Day: 1, NightActions: map[string]string{}, Votes: map[string]string{}},
	}

	hostView := manager.publicRoom(room, "u1")
	if hostView.Players[0].Role != RoleVillager {
		t.Fatalf("expected self role to be visible, got %q", hostView.Players[0].Role)
	}
	if hostView.Players[1].Role != "" {
		t.Fatalf("expected villager view to hide werewolf role, got %q", hostView.Players[1].Role)
	}

	wolfView := manager.publicRoom(room, "u2")
	if wolfView.Players[2].Role != RoleWerewolf {
		t.Fatalf("expected werewolves to see each other, got %q", wolfView.Players[2].Role)
	}
}

func TestAvalonGoodPlayerCannotSubmitFailCard(t *testing.T) {
	manager := NewManager(GameAvalon, nil)
	room := &Room{
		ID:         "AVLTEST",
		Game:       GameAvalon,
		HostUserID: "u1",
		Phase:      PhaseAvalonQuest,
		Players: []*Player{
			testPlayer("p1", "u1", "Merlin", RoleMerlin, AlignmentGood),
			testPlayer("p2", "u2", "Assassin", RoleAssassin, AlignmentEvil),
			testPlayer("p3", "u3", "Loyal", RoleLoyal, AlignmentGood),
			testPlayer("p4", "u4", "Loyal Two", RoleLoyal, AlignmentGood),
			testPlayer("p5", "u5", "Loyal Three", RoleLoyal, AlignmentGood),
		},
		Avalon: AvalonState{
			Round:         1,
			LeaderID:      "p1",
			Team:          []string{"p1", "p3"},
			TeamVotes:     map[string]bool{},
			QuestCards:    map[string]string{},
			RequiredTeam:  2,
			RequiredFails: 1,
		},
	}
	manager.rooms[room.ID] = room

	if _, err := manager.QuestCard(room.ID, "u1", "fail"); err == nil || err.Error() != "good_player_cannot_fail" {
		t.Fatalf("expected good_player_cannot_fail, got %v", err)
	}
}

func TestWerewolfCustomRoleConfigIsUsedOnStart(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "Host", Kind: "guest"})
	for index := 2; index <= 6; index++ {
		if _, err := manager.JoinRoom(room.ID, UserView{ID: fmt.Sprintf("u%d", index), DisplayName: fmt.Sprintf("Player %d", index), Kind: "guest"}); err != nil {
			t.Fatalf("join player %d: %v", index, err)
		}
	}

	custom := WerewolfRoleConfig{
		Mode: "custom",
		Counts: WerewolfRoleCounts{
			Villager: 4,
			Werewolf: 1,
			Seer:     1,
		},
	}
	if _, err := manager.UpdateWerewolfRoles(room.ID, "u1", custom); err != nil {
		t.Fatalf("update custom roles: %v", err)
	}

	if _, err := manager.Start(room.ID, "u1"); err != nil {
		t.Fatalf("start room: %v", err)
	}
	started, err := manager.room(room.ID)
	if err != nil {
		t.Fatalf("load started room: %v", err)
	}
	counts := WerewolfRoleCounts{}
	for _, player := range started.Players {
		switch player.Role {
		case RoleVillager:
			counts.Villager++
		case RoleWerewolf:
			counts.Werewolf++
		case RoleSeer:
			counts.Seer++
		case RoleGuard:
			counts.Guard++
		}
	}
	if counts != custom.Counts {
		t.Fatalf("expected custom counts %+v, got %+v", custom.Counts, counts)
	}
}

func TestWerewolfRejectsMismatchedRoleConfig(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "Host", Kind: "guest"})
	badConfig := WerewolfRoleConfig{
		Mode: "custom",
		Counts: WerewolfRoleCounts{
			Villager: 6,
			Werewolf: 1,
		},
	}

	if _, err := manager.UpdateWerewolfRoles(room.ID, "u1", badConfig); err == nil || err.Error() != "role_count_mismatch" {
		t.Fatalf("expected role_count_mismatch, got %v", err)
	}
}

func TestRenamePlayerKeepsStableIDs(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "Host", Kind: "guest"})
	playerID := room.Players[0].ID
	userID := room.Players[0].UserID

	public, err := manager.RenamePlayer(room.ID, "u1", "  Night   Reader  ")
	if err != nil {
		t.Fatalf("rename player: %v", err)
	}
	if public.Players[0].Name != "Night Reader" {
		t.Fatalf("expected normalized display name, got %q", public.Players[0].Name)
	}
	if public.Players[0].ID != playerID || public.Players[0].UserID != userID {
		t.Fatalf("expected ids to stay stable, got player=%q user=%q", public.Players[0].ID, public.Players[0].UserID)
	}
}

func TestPlayerNotesArePrivatePerViewer(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := manager.CreateRoom(UserView{ID: "u1", DisplayName: "Host", Kind: "guest"})
	if _, err := manager.JoinRoom(room.ID, UserView{ID: "u2", DisplayName: "Guest", Kind: "guest"}); err != nil {
		t.Fatalf("join room: %v", err)
	}
	internalRoom, err := manager.room(room.ID)
	if err != nil {
		t.Fatalf("load room: %v", err)
	}
	targetID := internalRoom.Players[1].ID

	hostView, err := manager.UpdatePlayerNote(room.ID, "u1", targetID, "  2号   可疑  ")
	if err != nil {
		t.Fatalf("update note: %v", err)
	}
	if hostView.Players[1].Note != "2号 可疑" {
		t.Fatalf("expected host note to be visible to host, got %q", hostView.Players[1].Note)
	}

	guestView := manager.publicRoom(internalRoom, "u2")
	if guestView.Players[1].Note != "" || guestView.Players[0].Note != "" {
		t.Fatalf("expected notes to be hidden from other viewer, got %+v", guestView.Players)
	}

	clearedView, err := manager.UpdatePlayerNote(room.ID, "u1", targetID, "")
	if err != nil {
		t.Fatalf("clear note: %v", err)
	}
	if clearedView.Players[1].Note != "" {
		t.Fatalf("expected note to be cleared, got %q", clearedView.Players[1].Note)
	}
}

func TestWerewolfNightUsesLLMDecision(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "target:p2",
			Speech:   "我先看二号。",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := &Room{
		ID:    "WWFLLM1",
		Game:  GameWerewolf,
		Phase: PhaseWerewolfNight,
		Players: []*Player{
			testAIPlayer("p1", "Seer Bot", RoleSeer, AlignmentGood),
			testPlayer("p2", "u2", "Villager", RoleVillager, AlignmentGood),
			testPlayer("p3", "u3", "Wolf", RoleWerewolf, AlignmentEvil),
		},
		Werewolf: WerewolfState{
			Day:          1,
			NightActions: map[string]string{},
			SeerChecks:   map[string]Alignment{},
			Votes:        map[string]string{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunNextAI(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if provider.input.Game != "werewolf" {
		t.Fatalf("expected werewolf decision input, got %q", provider.input.Game)
	}
	if !aiplayer.ValidateAction("target:p2", provider.input.Actions) {
		t.Fatalf("expected LLM actions to include target:p2, got %+v", provider.input.Actions)
	}
	if room.Werewolf.NightActions["p1"] != "p2" {
		t.Fatalf("expected seer to target p2, got %q", room.Werewolf.NightActions["p1"])
	}
	if room.Werewolf.SeerChecks["p2"] != AlignmentGood {
		t.Fatalf("expected seer check to store p2 alignment, got %q", room.Werewolf.SeerChecks["p2"])
	}
	if len(room.Speeches) != 1 || room.Speeches[0].Text != "我先看二号。" {
		t.Fatalf("expected LLM speech to be recorded, got %+v", room.Speeches)
	}
}

func TestWerewolfVoteUsesLLMDecision(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:p2",
			Notes: map[string]string{
				"p2":      "发言像狼",
				"missing": "不会保存",
			},
			Speech: "二号像狼。",
			Source: "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := &Room{
		ID:    "WWFLLM2",
		Game:  GameWerewolf,
		Phase: PhaseWerewolfVote,
		Players: []*Player{
			testAIPlayer("p1", "Vote Bot", RoleVillager, AlignmentGood),
			testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
			testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
		},
		Werewolf: WerewolfState{
			Day:          1,
			NightActions: map[string]string{},
			SeerChecks:   map[string]Alignment{},
			Votes:        map[string]string{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunNextAI(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if provider.input.Game != "werewolf" {
		t.Fatalf("expected werewolf decision input, got %q", provider.input.Game)
	}
	if !aiplayer.ValidateAction("vote:p2", provider.input.Actions) {
		t.Fatalf("expected LLM actions to include vote:p2, got %+v", provider.input.Actions)
	}
	if room.Werewolf.Votes["p1"] != "p2" {
		t.Fatalf("expected vote p2, got %q", room.Werewolf.Votes["p1"])
	}
	if room.PlayerNotes["p1"]["p2"] != "发言像狼" {
		t.Fatalf("expected AI note to be stored, got %+v", room.PlayerNotes)
	}
	if _, ok := room.PlayerNotes["p1"]["missing"]; ok {
		t.Fatalf("expected AI note for unknown player to be ignored")
	}
	if len(room.Speeches) != 1 || room.Speeches[0].Text != "二号像狼。" {
		t.Fatalf("expected LLM speech to be recorded, got %+v", room.Speeches)
	}
}

func TestWerewolfWitchCanSaveNightVictim(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWITCH1", PhaseWerewolfNight, []*Player{
		testPlayer("p1", "u1", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p2", "u2", "Witch", RoleWitch, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	room.Werewolf.NightActions["p1"] = "p3"
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u2", "save:p3"); err != nil {
		t.Fatalf("witch save: %v", err)
	}
	if !room.Players[2].Alive {
		t.Fatalf("expected villager to survive witch antidote")
	}
	if !room.Werewolf.WitchAntidoteUsed {
		t.Fatalf("expected witch antidote to be consumed")
	}
	if room.Phase != PhaseWerewolfDay {
		t.Fatalf("expected day phase, got %q", room.Phase)
	}
}

func TestWerewolfWitchCanPoison(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWITCH2", PhaseWerewolfNight, []*Player{
		testPlayer("p1", "u1", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p2", "u2", "Witch", RoleWitch, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
		testPlayer("p4", "u4", "Villager Two", RoleVillager, AlignmentGood),
	})
	room.Werewolf.NightActions["p1"] = "p3"
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u2", "poison:p1"); err != nil {
		t.Fatalf("witch poison: %v", err)
	}
	if room.Players[0].Alive {
		t.Fatalf("expected poisoned wolf to die")
	}
	if room.Players[2].Alive {
		t.Fatalf("expected wolf victim to die without antidote")
	}
	if !room.Werewolf.WitchPoisonUsed {
		t.Fatalf("expected witch poison to be consumed")
	}
	if room.Phase != PhaseFinished || room.Winner != AlignmentGood {
		t.Fatalf("expected good win after poisoned last wolf, phase=%q winner=%q", room.Phase, room.Winner)
	}
}

func TestWerewolfHunterCanShootAfterExile(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFHUNT", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Hunter", RoleHunter, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfVote(room.ID, "u2", "p1"); err != nil {
		t.Fatalf("wolf vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1"); err != nil {
		t.Fatalf("villager vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u1", "p2"); err != nil {
		t.Fatalf("hunter vote: %v", err)
	}
	if room.Phase != PhaseWerewolfHunter || room.Werewolf.HunterPendingID != "p1" {
		t.Fatalf("expected hunter phase, phase=%q pending=%q", room.Phase, room.Werewolf.HunterPendingID)
	}

	if _, err := manager.HunterShot(room.ID, "u1", "p2"); err != nil {
		t.Fatalf("hunter shot: %v", err)
	}
	if room.Players[1].Alive {
		t.Fatalf("expected hunter shot target to die")
	}
	if room.Phase != PhaseFinished || room.Winner != AlignmentGood {
		t.Fatalf("expected good win after hunter shot, phase=%q winner=%q", room.Phase, room.Winner)
	}
}

func TestWerewolfIdiotSurvivesFirstExile(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFIDIOT", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Idiot", RoleIdiot, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfVote(room.ID, "u1", "p2"); err != nil {
		t.Fatalf("idiot vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u2", "p1"); err != nil {
		t.Fatalf("wolf vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1"); err != nil {
		t.Fatalf("villager vote: %v", err)
	}
	if !room.Players[0].Alive {
		t.Fatalf("expected idiot to survive first exile")
	}
	if !room.Werewolf.RevealedIdiots["p1"] {
		t.Fatalf("expected idiot to be revealed")
	}
	if room.Phase != PhaseWerewolfNight {
		t.Fatalf("expected next night after idiot reveal, got %q", room.Phase)
	}
}

func testPlayer(id string, userID string, name string, role Role, alignment Alignment) *Player {
	return &Player{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Kind:      "guest",
		Connected: true,
		Alive:     true,
		Role:      role,
		Alignment: alignment,
		JoinedAt:  time.Now().UTC(),
	}
}

func testWerewolfRoom(id string, phase Phase, players []*Player) *Room {
	return &Room{
		ID:      id,
		Game:    GameWerewolf,
		Phase:   phase,
		Players: players,
		Werewolf: WerewolfState{
			Day:            1,
			NightActions:   map[string]string{},
			SeerChecks:     map[string]Alignment{},
			Votes:          map[string]string{},
			RevealedIdiots: map[string]bool{},
		},
	}
}

func testAIPlayer(id string, name string, role Role, alignment Alignment) *Player {
	player := testPlayer(id, "ai_"+id, name, role, alignment)
	player.Kind = "ai"
	player.IsAI = true
	player.AI = &AIProfile{
		Name:        name,
		Personality: "谨慎分析",
		SpeechStyle: "短句",
		Level:       string(aiplayer.LevelLLM),
	}
	return player
}

type fakeDecisionProvider struct {
	enabled  bool
	decision aiplayer.Decision
	input    aiplayer.DecisionInput
}

func (p *fakeDecisionProvider) Enabled() bool {
	return p.enabled
}

func (p *fakeDecisionProvider) Decide(_ context.Context, input aiplayer.DecisionInput) (aiplayer.Decision, error) {
	p.input = input
	return p.decision, nil
}
