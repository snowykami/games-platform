package socialdeduction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		Werewolf: WerewolfState{Day: 1, NightActions: map[string]string{}, Votes: map[string]WerewolfVoteIntent{}},
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

func TestPublicRoomGodViewRevealsAllRoles(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := &Room{
		ID:         "WWFGOD",
		Game:       GameWerewolf,
		HostUserID: "u1",
		Phase:      PhaseWerewolfNight,
		Players: []*Player{
			testPlayer("p1", "u1", "Host", RoleVillager, AlignmentGood),
			testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
			testPlayer("p3", "u3", "Seer", RoleSeer, AlignmentGood),
		},
		Werewolf: WerewolfState{Day: 1, NightActions: map[string]string{}, Votes: map[string]WerewolfVoteIntent{}},
	}

	normalView := manager.publicRoom(room, "u1")
	if normalView.Players[1].Role != "" || normalView.Players[1].VisibleToYou {
		t.Fatalf("expected normal villager view to hide wolf role, got %+v", normalView.Players[1])
	}

	godView := manager.publicRoomWithOptions(room, "u1", PublicRoomOptions{GodViewAvailable: true, GodView: true})
	if !godView.GodViewAvailable || !godView.GodViewEnabled {
		t.Fatalf("expected god view flags to be enabled, got available=%v enabled=%v", godView.GodViewAvailable, godView.GodViewEnabled)
	}
	for _, player := range godView.Players {
		if player.Role == "" || player.Alignment == "" || !player.VisibleToYou {
			t.Fatalf("expected god view to reveal player identity, got %+v", player)
		}
	}
}

func TestPublicRoomDoesNotSerializeStableUserIDs(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := &Room{
		ID:         "WWFTEST",
		Game:       GameWerewolf,
		HostUserID: "u1",
		Phase:      PhaseWerewolfNight,
		Players: []*Player{
			testPlayer("p1", "u1", "Host", RoleVillager, AlignmentGood),
			testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		},
		Werewolf: WerewolfState{Day: 1, NightActions: map[string]string{}, Votes: map[string]WerewolfVoteIntent{}},
	}

	view := manager.publicRoom(room, "u1")
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal public room: %v", err)
	}
	serialized := string(payload)
	if strings.Contains(serialized, "userId") || strings.Contains(serialized, "hostUserId") {
		t.Fatalf("public room leaked stable user IDs: %s", serialized)
	}
	if view.HostPlayerID != "p1" || view.YouPlayerID != "p1" {
		t.Fatalf("expected per-room player IDs, got host=%q you=%q", view.HostPlayerID, view.YouPlayerID)
	}
}

func TestAvalonTeamVotesHiddenUntilVoteCompletes(t *testing.T) {
	manager := NewManager(GameAvalon, nil)
	room := &Room{
		ID:         "AVLTEST",
		Game:       GameAvalon,
		HostUserID: "u1",
		Phase:      PhaseAvalonVote,
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
			TeamVotes:     map[string]bool{"p1": true, "p2": false},
			QuestCards:    map[string]string{},
			RequiredTeam:  2,
			RequiredFails: 1,
		},
	}

	voteView := manager.publicRoom(room, "u3")
	if len(voteView.Avalon.TeamVotes) != 0 {
		t.Fatalf("expected hidden votes during team vote, got %#v", voteView.Avalon.TeamVotes)
	}
	if voteView.Avalon.TeamVoteCount != 2 {
		t.Fatalf("expected submitted vote count 2, got %d", voteView.Avalon.TeamVoteCount)
	}

	room.Phase = PhaseAvalonQuest
	revealedView := manager.publicRoom(room, "u3")
	if len(revealedView.Avalon.TeamVotes) != 2 {
		t.Fatalf("expected votes after team vote phase, got %#v", revealedView.Avalon.TeamVotes)
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
			ActionID: "target:seat_2",
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
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if provider.input.Game != "werewolf" {
		t.Fatalf("expected werewolf decision input, got %q", provider.input.Game)
	}
	if !aiplayer.ValidateAction("target:seat_2", provider.input.Actions) {
		t.Fatalf("expected LLM actions to include target:seat_2, got %+v", provider.input.Actions)
	}
	if room.Werewolf.NightActions["p1"] != "p2" {
		t.Fatalf("expected seer to target p2, got %q", room.Werewolf.NightActions["p1"])
	}
	if room.Werewolf.SeerChecks["p2"] != AlignmentGood {
		t.Fatalf("expected seer check to store p2 alignment, got %q", room.Werewolf.SeerChecks["p2"])
	}
	if len(room.Speeches) != 0 {
		t.Fatalf("expected werewolf night speech to stay private, got %+v", room.Speeches)
	}
}

func TestWerewolfLLMInputDoesNotExposeAIOrHumanIDPrefixes(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "target:seat_2",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := &Room{
		ID:    "WWFLLMIDS",
		Game:  GameWerewolf,
		Phase: PhaseWerewolfNight,
		Players: []*Player{
			testAIPlayer("ai_wolf", "北风", RoleWerewolf, AlignmentEvil),
			testPlayer("plr_human", "u1", "snowykami", RoleVillager, AlignmentGood),
			testAIPlayer("ai_seer", "南星", RoleSeer, AlignmentGood),
		},
		Werewolf: WerewolfState{
			Day:          1,
			NightActions: map[string]string{},
			SeerChecks:   map[string]Alignment{},
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	payload, err := json.Marshal(provider.input)
	if err != nil {
		t.Fatalf("marshal llm input: %v", err)
	}
	serialized := string(payload)
	if strings.Contains(serialized, "ai_") || strings.Contains(serialized, "plr_") {
		t.Fatalf("expected LLM input to hide internal player ID prefixes, got %s", serialized)
	}
	if strings.Contains(serialized, "击杀 snowykami") || strings.Contains(serialized, "击杀 北风") {
		t.Fatalf("expected LLM kill actions to use seat labels, got %s", serialized)
	}
	if room.Werewolf.NightActions["ai_wolf"] != "plr_human" {
		t.Fatalf("expected aliased action to resolve to real target, got %q", room.Werewolf.NightActions["ai_wolf"])
	}
}

func TestWerewolfNightWithOnlyHumanIdiotAdvancesByAI(t *testing.T) {
	provider := &fakeDecisionProvider{enabled: true}
	provider.onDecide = func(input aiplayer.DecisionInput) {
		if len(input.Actions) == 0 {
			return
		}
		provider.decision = aiplayer.Decision{ActionID: input.Actions[0].ID, Source: "llm"}
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFIDIO", PhaseWerewolfNight, []*Player{
		testPlayer("human_idiot", "u1", "snowykami", RoleIdiot, AlignmentGood),
		testAIPlayer("wolf_1", "北风", RoleWerewolf, AlignmentEvil),
		testAIPlayer("wolf_2", "南星", RoleWerewolf, AlignmentEvil),
		testAIPlayer("seer_1", "阿澈", RoleSeer, AlignmentGood),
		testAIPlayer("witch_1", "白川", RoleWitch, AlignmentGood),
		testAIPlayer("villager_1", "星野", RoleVillager, AlignmentGood),
		testAIPlayer("villager_2", "青灯", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	for i := 0; i < 8 && room.Phase == PhaseWerewolfNight; i++ {
		if _, _, err := manager.RunAIAction(room.ID); err != nil {
			t.Fatalf("run ai action %d: %v", i, err)
		}
	}
	if room.Phase != PhaseWerewolfDay {
		t.Fatalf("expected AI night actions to advance to day, phase=%s pending=%v actions=%v", room.Phase, pendingRequiredActions(room), room.Werewolf.NightActions)
	}
}

func TestSocialLLMInputIncludesGameAndSpeechGuidance(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "target:seat_2", Source: "llm"},
	}
	manager := NewManager(GameWerewolf, provider)
	room := &Room{
		ID:    "WWFGUIDE",
		Game:  GameWerewolf,
		Phase: PhaseWerewolfNight,
		Players: []*Player{
			testAIPlayer("ai_seer", "北风", RoleSeer, AlignmentGood),
			testPlayer("human_villager", "u1", "snowykami", RoleVillager, AlignmentGood),
			testAIPlayer("ai_wolf", "南星", RoleWerewolf, AlignmentEvil),
		},
		Werewolf: WerewolfState{
			Day:          1,
			NightActions: map[string]string{},
			SeerChecks:   map[string]Alignment{},
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	state, ok := provider.input.State.(map[string]any)
	if !ok {
		t.Fatalf("expected map state, got %#v", provider.input.State)
	}
	serialized := fmt.Sprint(state)
	for _, expected := range []string{"真实玩家", "狼人杀目标", "夜晚行动", "避免模板话", "不能直接或间接泄露隐藏身份"} {
		if !strings.Contains(serialized, expected) {
			t.Fatalf("expected social guidance %q in LLM state, got %s", expected, serialized)
		}
	}
}

func TestUndercoverDescriptionActionsDoNotRevealSecretWord(t *testing.T) {
	room := &Room{
		ID:    "UNDCLUE1",
		Game:  GameUndercover,
		Phase: PhaseUndercoverDescribe,
		Players: []*Player{
			testAIPlayer("p1", "Clue Bot", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			WordPair: UndercoverWordPair{CivilianWord: "苹果", UndercoverWord: "梨"},
		},
	}

	actions := undercoverDescriptionActions(room, room.Players[0])
	if len(actions) == 0 {
		t.Fatal("expected clue actions")
	}
	for _, action := range actions {
		if strings.Contains(action.Label, "苹果") || strings.Contains(action.Description, "苹果") {
			t.Fatalf("expected action to avoid secret word, got %+v", action)
		}
	}
	state := undercoverAIState(room, room.Players[0], "describe")
	forbidden, ok := state["forbiddenPublicSpeech"].([]string)
	if !ok || len(forbidden) != 1 || forbidden[0] != "苹果" {
		t.Fatalf("expected forbidden public speech to include secret word, got %#v", state["forbiddenPublicSpeech"])
	}
}

func TestUndercoverLLMDescriptionRejectsUnsafeSpeech(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "say:use",
			Speech:   "我的词是苹果",
			Source:   "llm",
		},
	}
	manager := NewManager(GameUndercover, provider)
	room := &Room{
		ID:    "UNDCLUE2",
		Game:  GameUndercover,
		Phase: PhaseUndercoverDescribe,
		Players: []*Player{
			testAIPlayer("p1", "Clue Bot", RoleCivilian, AlignmentGood),
			testPlayer("p2", "u2", "Guest", RoleUndercover, AlignmentEvil),
			testPlayer("p3", "u3", "Guest Two", RoleCivilian, AlignmentGood),
			testPlayer("p4", "u4", "Guest Three", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			Round:            1,
			WordPair:         UndercoverWordPair{CivilianWord: "苹果", UndercoverWord: "梨"},
			CurrentSpeakerID: "p1",
			Described:        map[string]bool{},
			Votes:            map[string]UndercoverVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if len(room.Speeches) != 1 {
		t.Fatalf("expected one public clue, got %+v", room.Speeches)
	}
	if strings.Contains(room.Speeches[0].Text, "苹果") {
		t.Fatalf("expected public clue to hide secret word, got %q", room.Speeches[0].Text)
	}
	if room.Speeches[0].Text == "我的词是苹果" || strings.Contains(room.Speeches[0].Text, "生活里") || strings.Contains(room.Speeches[0].Text, "具体场景") {
		t.Fatalf("expected safe non-template fallback, got %q", room.Speeches[0].Text)
	}
}

func TestWerewolfVoteUsesLLMDecision(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:seat_2",
			Notes: map[string]string{
				"seat_2":  "发言像狼",
				"seat_99": "不会保存",
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
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if provider.input.Game != "werewolf" {
		t.Fatalf("expected werewolf decision input, got %q", provider.input.Game)
	}
	if !aiplayer.ValidateAction("vote:seat_2", provider.input.Actions) {
		t.Fatalf("expected LLM actions to include vote:seat_2, got %+v", provider.input.Actions)
	}
	if vote := room.Werewolf.Votes["p1"]; vote.TargetID != "p2" || !vote.Confirmed {
		t.Fatalf("expected confirmed vote p2, got %+v", vote)
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

func TestWerewolfVoteFallbackSpeechIsSpecific(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:seat_2",
			Speech:   "",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := &Room{
		ID:    "WWFLLMFALLBACK",
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
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if len(room.Speeches) != 1 {
		t.Fatalf("expected fallback vote speech, got %+v", room.Speeches)
	}
	if room.Speeches[0].Text == "我投这里。" || !strings.Contains(room.Speeches[0].Text, "2号") {
		t.Fatalf("expected specific fallback speech with target seat, got %q", room.Speeches[0].Text)
	}
}

func TestWerewolfCanTargetSelfAtNight(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFSELF", PhaseWerewolfNight, []*Player{
		testPlayer("p1", "u1", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p2", "u2", "Witch", RoleWitch, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u1", "target:p1"); err != nil {
		t.Fatalf("werewolf self target should be legal: %v", err)
	}
	if room.Werewolf.NightActions["p1"] != "p1" {
		t.Fatalf("expected wolf self target to be recorded, got %q", room.Werewolf.NightActions["p1"])
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

	if _, err := manager.WerewolfVote(room.ID, "u2", "p1", true); err != nil {
		t.Fatalf("wolf vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1", true); err != nil {
		t.Fatalf("villager vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u1", "p2", true); err != nil {
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

func TestWerewolfAIHunterActsAfterNightDeath(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFAIHUNT", PhaseWerewolfNight, []*Player{
		testAIPlayer("p1", "Wolf", RoleWerewolf, AlignmentEvil),
		testAIPlayer("p2", "Hunter", RoleHunter, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	room.Werewolf.NightActions["p1"] = "p2"
	manager.rooms[room.ID] = room

	manager.advanceWerewolfNight(room)
	if room.Phase != PhaseWerewolfHunter || room.Werewolf.HunterPendingID != "p2" || room.Players[1].Alive {
		t.Fatalf("expected dead AI hunter to be pending, phase=%q pending=%q alive=%v", room.Phase, room.Werewolf.HunterPendingID, room.Players[1].Alive)
	}
	if !hasPendingAIRequiredAction(room) {
		t.Fatal("expected dead AI hunter shot to be treated as pending")
	}

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai hunter shot: %v", err)
	}
	if room.Phase == PhaseWerewolfHunter || room.Werewolf.HunterPendingID != "" {
		t.Fatalf("expected AI hunter phase to resolve, phase=%q pending=%q", room.Phase, room.Werewolf.HunterPendingID)
	}
}

func TestWerewolfLLMHunterDecisionAllowsDeadPendingHunter(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "shoot:skip",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFLLMHUNT", PhaseWerewolfHunter, []*Player{
		testAIPlayer("p1", "Wolf", RoleWerewolf, AlignmentEvil),
		testAIPlayer("p2", "Hunter", RoleHunter, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	room.Players[1].Alive = false
	room.Werewolf.HunterPendingID = "p2"
	room.Werewolf.HunterAfterPhase = PhaseWerewolfDay
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run llm hunter shot: %v", err)
	}
	if len(provider.input.Actions) == 0 {
		t.Fatal("expected LLM hunter action to be requested")
	}
	if room.Phase == PhaseWerewolfHunter || room.Werewolf.HunterPendingID != "" {
		t.Fatalf("expected dead LLM hunter to resolve hunter phase, phase=%q pending=%q", room.Phase, room.Werewolf.HunterPendingID)
	}
}

func TestWerewolfVoteRequiresConfirmationAndAllowsChanges(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFREVOTE", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Villager A", RoleVillager, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager B", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfVote(room.ID, "u1", "p2", false); err != nil {
		t.Fatalf("select vote: %v", err)
	}
	if vote := room.Werewolf.Votes["p1"]; vote.TargetID != "p2" || vote.Confirmed {
		t.Fatalf("expected unconfirmed selection p2, got %+v", vote)
	}
	if room.Phase != PhaseWerewolfVote {
		t.Fatalf("unconfirmed vote should not resolve, got phase %q", room.Phase)
	}

	if _, err := manager.WerewolfVote(room.ID, "u1", "p3", false); err != nil {
		t.Fatalf("change vote: %v", err)
	}
	if vote := room.Werewolf.Votes["p1"]; vote.TargetID != "p3" || vote.Confirmed {
		t.Fatalf("expected changed unconfirmed selection p3, got %+v", vote)
	}

	if _, err := manager.WerewolfVote(room.ID, "u1", "p3", true); err != nil {
		t.Fatalf("confirm vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u2", "p1", true); err != nil {
		t.Fatalf("wolf confirm: %v", err)
	}
	if room.Phase != PhaseWerewolfVote {
		t.Fatalf("not all voters confirmed yet, got phase %q", room.Phase)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1", true); err != nil {
		t.Fatalf("final confirm: %v", err)
	}
	if room.Phase == PhaseWerewolfVote {
		t.Fatalf("expected vote to resolve after all voters confirm")
	}
}

func TestWerewolfVoteCanBeCanceledBeforeConfirmation(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFCANCEL", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Villager A", RoleVillager, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager B", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfVote(room.ID, "u1", "p2", false); err != nil {
		t.Fatalf("select vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u1", "", false); err != nil {
		t.Fatalf("cancel vote: %v", err)
	}
	if _, ok := room.Werewolf.Votes["p1"]; ok {
		t.Fatalf("expected canceled vote to be removed, got %+v", room.Werewolf.Votes["p1"])
	}
	if room.Phase != PhaseWerewolfVote {
		t.Fatalf("canceling vote should not resolve phase, got %q", room.Phase)
	}
}

func TestWerewolfDayAutoAdvancesAfterAllVotersSpeak(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFAUTOVOTE", PhaseWerewolfDay, []*Player{
		testPlayer("p1", "u1", "Villager A", RoleVillager, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager B", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.Say(room.ID, "u1", "我先发言。"); err != nil {
		t.Fatalf("first speech: %v", err)
	}
	if room.Phase != PhaseWerewolfDay {
		t.Fatalf("expected day to continue until all voters speak, got %q", room.Phase)
	}
	if _, err := manager.Say(room.ID, "u2", "我跟一手。"); err != nil {
		t.Fatalf("second speech: %v", err)
	}
	if _, err := manager.Say(room.ID, "u3", "我也发言。"); err != nil {
		t.Fatalf("final speech: %v", err)
	}
	if room.Phase != PhaseWerewolfVote {
		t.Fatalf("expected all speeches to auto-start vote, got %q", room.Phase)
	}
	if len(room.Werewolf.Votes) != 0 {
		t.Fatalf("expected fresh vote map after auto-start, got %+v", room.Werewolf.Votes)
	}
}

func TestSpeechLogKeepsStablePlayerID(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFLOGID", PhaseWerewolfDay, []*Player{
		testPlayer("p1", "u1", "同名", RoleVillager, AlignmentGood),
		testPlayer("p2", "u2", "同名", RoleWerewolf, AlignmentEvil),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.Say(room.ID, "u2", "我发一条。"); err != nil {
		t.Fatalf("say: %v", err)
	}
	if len(room.Log) == 0 {
		t.Fatal("expected speech log")
	}
	entry := room.Log[len(room.Log)-1]
	if entry.PlayerID != "p2" || entry.PlayerName != "同名" {
		t.Fatalf("expected stable speech log identity, got %+v", entry)
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

	if _, err := manager.WerewolfVote(room.ID, "u1", "p2", true); err != nil {
		t.Fatalf("idiot vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u2", "p1", true); err != nil {
		t.Fatalf("wolf vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1", true); err != nil {
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

func TestSocialAIContextUsesSeatAliasesAndHidesIdentityFields(t *testing.T) {
	room := &Room{
		ID:    "AVLCTX",
		Game:  GameAvalon,
		Phase: PhaseAvalonTeam,
		Players: []*Player{
			testAIPlayer("raw_merlin_id", "梅林", RoleMerlin, AlignmentGood),
			testAIPlayer("raw_assassin_id", "刺客", RoleAssassin, AlignmentEvil),
			testAIPlayer("raw_loyal_id", "忠臣", RoleLoyal, AlignmentGood),
			testAIPlayer("raw_minion_id", "爪牙", RoleMinion, AlignmentEvil),
			testAIPlayer("raw_loyal_two_id", "忠臣二", RoleLoyal, AlignmentGood),
		},
		Avalon: AvalonState{
			Round:         1,
			LeaderID:      "raw_merlin_id",
			Team:          []string{"raw_merlin_id", "raw_loyal_id"},
			TeamVotes:     map[string]bool{"raw_merlin_id": true, "raw_assassin_id": false},
			QuestCards:    map[string]string{},
			RequiredTeam:  2,
			RequiredFails: 1,
		},
		Speeches: []SpeechEntry{{ID: "speech_1", PlayerID: "raw_assassin_id", PlayerName: "刺客", Text: "我先听队长安排。", SpokenAt: time.Now().UTC()}},
	}

	state := avalonAIState(room, room.Players[2], "team")
	actions, _ := avalonTeamActionsForLLM(room, avalonTeamActions(room, room.Players[0]))
	payload, err := json.Marshal(map[string]any{"state": state, "actions": actions})
	if err != nil {
		t.Fatalf("marshal ai context: %v", err)
	}
	assertAIContextDoesNotLeak(t, string(payload), []string{
		"raw_merlin_id",
		"raw_assassin_id",
		"raw_loyal_id",
		"raw_minion_id",
		"raw_loyal_two_id",
		"ai_raw_",
		"isAI",
		"kind",
		"userId",
		"connected",
		"aiProfile",
		"assassin",
		"minion",
	})
	if !strings.Contains(string(payload), "seat_1") || !strings.Contains(string(payload), "seat_2") {
		t.Fatalf("expected seat aliases in AI context: %s", string(payload))
	}
}

func TestUndercoverAIContextUsesSeatAliasesAndMapsVoteBack(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "vote:seat_2", Source: "llm"},
	}
	manager := NewManager(GameUndercover, provider)
	room := &Room{
		ID:    "UNDCTX",
		Game:  GameUndercover,
		Phase: PhaseUndercoverVote,
		Players: []*Player{
			testAIPlayer("raw_civilian_id", "平民", RoleCivilian, AlignmentGood),
			testAIPlayer("raw_undercover_id", "卧底", RoleUndercover, AlignmentEvil),
			testAIPlayer("raw_blank_id", "白板", RoleBlank, AlignmentNeutral),
		},
		Undercover: UndercoverState{
			Round:     1,
			WordPair:  UndercoverWordPair{CivilianWord: "苹果", UndercoverWord: "梨"},
			Described: map[string]bool{"raw_civilian_id": true, "raw_undercover_id": true},
			Votes:     map[string]UndercoverVoteIntent{"raw_undercover_id": {TargetID: "raw_civilian_id", Confirmed: true}},
		},
		Speeches: []SpeechEntry{{ID: "speech_1", PlayerID: "raw_undercover_id", PlayerName: "卧底", Text: "我说一个偏甜的方向。", SpokenAt: time.Now().UTC()}},
	}

	state := undercoverAIState(room, room.Players[0], "vote")
	actions, _ := playerTargetActionsForLLM(room, undercoverVoteActions(room, room.Players[0]), []string{"vote:"})
	payload, err := json.Marshal(map[string]any{"state": state, "actions": actions})
	if err != nil {
		t.Fatalf("marshal ai context: %v", err)
	}
	assertAIContextDoesNotLeak(t, string(payload), []string{
		"raw_civilian_id",
		"raw_undercover_id",
		"raw_blank_id",
		"ai_raw_",
		"isAI",
		"kind",
		"userId",
		"connected",
		"aiProfile",
	})
	if strings.Contains(string(payload), `"role":"undercover"`) || strings.Contains(string(payload), `"role":"blank"`) {
		t.Fatalf("undercover context leaked other hidden roles: %s", string(payload))
	}

	manager.mu.Lock()
	target, speech := manager.chooseUndercoverVote(room, room.Players[0])
	manager.mu.Unlock()
	if target == nil || target.ID != "raw_undercover_id" {
		t.Fatalf("expected vote alias to map back to raw_undercover_id, got %+v", target)
	}
	if speech != "" {
		t.Fatalf("expected empty speech when LLM did not provide a reason, got %q", speech)
	}
}

func TestUndercoverVoteDoesNotRecordGenericTemplateSpeech(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:seat_2",
			Speech:   "我先票这个位置。",
			Source:   "llm",
		},
	}
	manager := NewManager(GameUndercover, provider)
	room := &Room{
		ID:    "UNDVOTETPL",
		Game:  GameUndercover,
		Phase: PhaseUndercoverVote,
		Players: []*Player{
			testAIPlayer("p1", "北风", RoleCivilian, AlignmentGood),
			testAIPlayer("p2", "南星", RoleUndercover, AlignmentEvil),
			testPlayer("p3", "u3", "阿澈", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			Round:     1,
			WordPair:  UndercoverWordPair{CivilianWord: "苹果", UndercoverWord: "梨"},
			Described: map[string]bool{"p1": true, "p2": true, "p3": true},
			Votes:     map[string]UndercoverVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai vote: %v", err)
	}
	if vote := room.Undercover.Votes["p1"]; vote.TargetID != "p2" || !vote.Confirmed {
		t.Fatalf("expected confirmed vote p2, got %+v", vote)
	}
	if len(room.Speeches) != 0 {
		t.Fatalf("expected generic vote speech to be skipped, got %+v", room.Speeches)
	}
}

func TestUndercoverVoteRecordsNaturalReasonSpeech(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:seat_2",
			Speech:   "他刚才那个甜味方向和前面几个人不太搭。",
			Source:   "llm",
		},
	}
	manager := NewManager(GameUndercover, provider)
	room := &Room{
		ID:    "UNDVOTEREASON",
		Game:  GameUndercover,
		Phase: PhaseUndercoverVote,
		Players: []*Player{
			testAIPlayer("p1", "北风", RoleCivilian, AlignmentGood),
			testAIPlayer("p2", "南星", RoleUndercover, AlignmentEvil),
			testPlayer("p3", "u3", "阿澈", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			Round:     1,
			WordPair:  UndercoverWordPair{CivilianWord: "苹果", UndercoverWord: "梨"},
			Described: map[string]bool{"p1": true, "p2": true, "p3": true},
			Votes:     map[string]UndercoverVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai vote: %v", err)
	}
	if len(room.Speeches) != 1 || room.Speeches[0].Text != "他刚才那个甜味方向和前面几个人不太搭。" {
		t.Fatalf("expected natural vote reason speech, got %+v", room.Speeches)
	}
}

func TestUndercoverVoteRequiresConfirmationAndAllowsChanges(t *testing.T) {
	manager := NewManager(GameUndercover, nil)
	room := &Room{
		ID:         "UNDREVOTE",
		Game:       GameUndercover,
		HostUserID: "u1",
		Phase:      PhaseUndercoverVote,
		Players: []*Player{
			testPlayer("p1", "u1", "玩家一", RoleCivilian, AlignmentGood),
			testPlayer("p2", "u2", "玩家二", RoleUndercover, AlignmentEvil),
			testPlayer("p3", "u3", "玩家三", RoleCivilian, AlignmentGood),
			testPlayer("p4", "u4", "玩家四", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			Round:     1,
			Described: map[string]bool{"p1": true, "p2": true, "p3": true, "p4": true},
			Votes:     map[string]UndercoverVoteIntent{},
		},
	}
	manager.rooms[room.ID] = room

	if _, err := manager.UndercoverVote(room.ID, "u1", "p2", false); err != nil {
		t.Fatalf("select vote: %v", err)
	}
	if vote := room.Undercover.Votes["p1"]; vote.TargetID != "p2" || vote.Confirmed {
		t.Fatalf("expected unconfirmed selection p2, got %+v", vote)
	}
	if room.Phase != PhaseUndercoverVote {
		t.Fatalf("unconfirmed vote should not resolve, got phase %q", room.Phase)
	}

	if _, err := manager.UndercoverVote(room.ID, "u1", "p3", false); err != nil {
		t.Fatalf("change vote: %v", err)
	}
	if vote := room.Undercover.Votes["p1"]; vote.TargetID != "p3" || vote.Confirmed {
		t.Fatalf("expected changed unconfirmed selection p3, got %+v", vote)
	}

	if _, err := manager.UndercoverVote(room.ID, "u1", "p3", true); err != nil {
		t.Fatalf("confirm vote: %v", err)
	}
	if _, err := manager.UndercoverVote(room.ID, "u2", "p3", true); err != nil {
		t.Fatalf("second vote: %v", err)
	}
	if _, err := manager.UndercoverVote(room.ID, "u3", "p2", true); err != nil {
		t.Fatalf("third vote: %v", err)
	}
	if room.Phase != PhaseUndercoverVote {
		t.Fatalf("not all voters confirmed yet, got phase %q", room.Phase)
	}
	if _, err := manager.UndercoverVote(room.ID, "u4", "p2", true); err != nil {
		t.Fatalf("final vote: %v", err)
	}
	if room.Phase == PhaseUndercoverVote {
		t.Fatalf("expected vote to resolve after all living voters confirm")
	}
}

func TestSocialDecisionDoesNotBecomeStaleFromSpeechUpdate(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "skip:witch", Source: "llm"},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFSTALE", PhaseWerewolfNight, []*Player{
		testPlayer("human_villager", "u_human", "真人", RoleVillager, AlignmentGood),
		testAIPlayer("ai_witch", "女巫", RoleWitch, AlignmentGood),
		testAIPlayer("ai_wolf", "狼人", RoleWerewolf, AlignmentEvil),
	})
	manager.rooms[room.ID] = room
	provider.onDecide = func(aiplayer.DecisionInput) {
		if _, err := manager.Say(room.ID, "u_human", "我插一句，不应该打断夜晚行动。"); err != nil {
			t.Fatalf("say during decision: %v", err)
		}
	}

	manager.mu.Lock()
	actionID, _ := manager.chooseWerewolfNightAction(room, room.Players[1])
	manager.mu.Unlock()

	if actionID != "skip:witch" {
		t.Fatalf("expected speech-only update not to stale required action, got %q", actionID)
	}
}

func TestSocialDecisionDoesNotBecomeStaleFromPresenceUpdate(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "skip:witch", Source: "llm"},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFPRESENCE", PhaseWerewolfNight, []*Player{
		testPlayer("human_villager", "u_human", "真人", RoleVillager, AlignmentGood),
		testAIPlayer("ai_witch", "女巫", RoleWitch, AlignmentGood),
		testAIPlayer("ai_wolf", "狼人", RoleWerewolf, AlignmentEvil),
	})
	manager.rooms[room.ID] = room
	provider.onDecide = func(aiplayer.DecisionInput) {
		touchPresence(room)
	}

	manager.mu.Lock()
	actionID, _ := manager.chooseWerewolfNightAction(room, room.Players[1])
	manager.mu.Unlock()

	if actionID != "skip:witch" {
		t.Fatalf("expected presence-only update not to stale required action, got %q", actionID)
	}
}

func TestSocialDecisionBecomesStaleFromRuleUpdate(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled:  true,
		decision: aiplayer.Decision{ActionID: "skip:witch", Source: "llm"},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFRULE", PhaseWerewolfNight, []*Player{
		testPlayer("human_villager", "u_human", "真人", RoleVillager, AlignmentGood),
		testAIPlayer("ai_witch", "女巫", RoleWitch, AlignmentGood),
		testAIPlayer("ai_wolf", "狼人", RoleWerewolf, AlignmentEvil),
	})
	touchRule(room)
	manager.rooms[room.ID] = room
	provider.onDecide = func(aiplayer.DecisionInput) {
		touchRule(room)
	}

	manager.mu.Lock()
	actionID, _ := manager.chooseWerewolfNightAction(room, room.Players[1])
	manager.mu.Unlock()

	if actionID != "" {
		t.Fatalf("expected rule update to stale required action, got %q", actionID)
	}
}

func TestSocialOptionalSpeechBecomesStaleFromSpeechUpdate(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "speak",
			Speech:   "我插一句自己的判断。",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFSPEECH", PhaseWerewolfDay, []*Player{
		testPlayer("human_villager", "u_human", "真人", RoleVillager, AlignmentGood),
		testAIPlayer("ai_villager", "北风", RoleVillager, AlignmentGood),
	})
	now := time.Now().UTC()
	room.RuleUpdatedAt = now
	room.SpeechUpdatedAt = now
	room.Speeches = []SpeechEntry{{ID: "speech_1", PlayerID: "human_villager", PlayerName: "真人", Text: "先听听大家。", SpokenAt: now}}
	manager.rooms[room.ID] = room
	provider.onDecide = func(aiplayer.DecisionInput) {
		if _, err := manager.Say(room.ID, "u_human", "我又补一句。"); err != nil {
			t.Fatalf("say during optional speech: %v", err)
		}
	}

	_, changed, err := manager.RunAIOptionalSpeech(room.ID)
	if err == nil {
		t.Fatal("expected optional speech to become stale")
	}
	if changed {
		t.Fatal("stale optional speech should not be broadcast as changed")
	}
	if len(room.Speeches) != 2 || room.Speeches[1].PlayerName != "真人" {
		t.Fatalf("expected only human follow-up speech to be recorded, got %+v", room.Speeches)
	}
}

func TestNextAISpeechPlayerRotatesFromLastSpeaker(t *testing.T) {
	room := testWerewolfRoom("WWFROTATE", PhaseWerewolfDay, []*Player{
		testPlayer("human_1", "u_human", "真人", RoleVillager, AlignmentGood),
		testAIPlayer("ai_2", "北风", RoleVillager, AlignmentGood),
		testAIPlayer("ai_3", "南星", RoleVillager, AlignmentGood),
		testAIPlayer("ai_4", "阿澈", RoleVillager, AlignmentGood),
	})

	if player := nextAISpeechPlayer(room, "human_1"); player == nil || player.ID != "ai_2" {
		t.Fatalf("expected next seat AI to respond after human, got %+v", player)
	}
	if player := nextAISpeechPlayer(room, "ai_2"); player == nil || player.ID != "ai_3" {
		t.Fatalf("expected AI response to rotate to the next seat, got %+v", player)
	}
	if player := nextAISpeechPlayer(room, "ai_4"); player == nil || player.ID != "ai_2" {
		t.Fatalf("expected AI response to wrap around, got %+v", player)
	}
}

func TestShouldContinueSocialAIOptionalSpeechLimitsShortBurst(t *testing.T) {
	room := PublicRoom{
		Players: []PublicPlayer{
			{ID: "human_1", IsAI: false, Alive: true},
			{ID: "ai_2", IsAI: true, Alive: true},
			{ID: "ai_3", IsAI: true, Alive: true},
		},
		Speeches: []SpeechEntry{
			{ID: "speech_1", PlayerID: "human_1"},
			{ID: "speech_2", PlayerID: "ai_2"},
		},
	}
	if !shouldContinueSocialAIOptionalSpeech(room) {
		t.Fatal("expected one AI reply to allow a second short response")
	}
	room.Speeches = append(room.Speeches, SpeechEntry{ID: "speech_3", PlayerID: "ai_3"})
	if shouldContinueSocialAIOptionalSpeech(room) {
		t.Fatal("expected AI optional speech burst to stop after two consecutive AI replies")
	}
}

func assertAIContextDoesNotLeak(t *testing.T, payload string, forbidden []string) {
	t.Helper()
	for _, token := range forbidden {
		if strings.Contains(payload, token) {
			t.Fatalf("AI context leaked %q: %s", token, payload)
		}
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
			Votes:          map[string]WerewolfVoteIntent{},
			DaySpeakers:    map[string]bool{},
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
	onDecide func(aiplayer.DecisionInput)
}

func (p *fakeDecisionProvider) Enabled() bool {
	return p.enabled
}

func (p *fakeDecisionProvider) Decide(_ context.Context, input aiplayer.DecisionInput) (aiplayer.Decision, error) {
	p.input = input
	if p.onDecide != nil {
		p.onDecide(input)
	}
	return p.decision, nil
}
