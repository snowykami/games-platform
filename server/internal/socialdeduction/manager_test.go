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

func TestPublicRoomShowsSeerCheckedAlignmentWithoutRole(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := &Room{
		ID:         "WWFSEERVIEW",
		Game:       GameWerewolf,
		HostUserID: "u1",
		Phase:      PhaseWerewolfNight,
		Players: []*Player{
			testPlayer("seer", "u_seer", "Seer", RoleSeer, AlignmentGood),
			testPlayer("wolf", "u_wolf", "Wolf", RoleWerewolf, AlignmentEvil),
			testPlayer("villager", "u_villager", "Villager", RoleVillager, AlignmentGood),
		},
		Werewolf: WerewolfState{
			Day:          1,
			NightActions: map[string]string{"seer": "wolf"},
			SeerChecks:   map[string]Alignment{"wolf": AlignmentEvil},
			Votes:        map[string]WerewolfVoteIntent{},
		},
	}

	seerView := manager.publicRoom(room, "u_seer")
	if !seerView.Werewolf.NightSubmitted {
		t.Fatal("expected seer view to mark own night action submitted")
	}
	if seerView.Players[1].Role != "" || seerView.Players[1].Alignment != "" || seerView.Players[1].VisibleToYou {
		t.Fatalf("expected checked wolf role to stay hidden in public player, got %+v", seerView.Players[1])
	}
	if got := seerView.Werewolf.SeerChecks["wolf"]; got != AlignmentEvil {
		t.Fatalf("expected seer checks to expose only checked alignment, got %q", got)
	}

	wolfView := manager.publicRoom(room, "u_wolf")
	if wolfView.Werewolf.SeerChecks != nil {
		t.Fatalf("expected non-seer not to receive seer checks, got %+v", wolfView.Werewolf.SeerChecks)
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

func TestRestartClearsPreviousGameConversationAndAIMemory(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFRESTART", PhaseFinished, []*Player{
		testPlayer("p1", "u1", "Host", RoleVillager, AlignmentGood),
		testAIPlayer("p2", "North", RoleWerewolf, AlignmentEvil),
		testAIPlayer("p3", "South", RoleVillager, AlignmentGood),
		testAIPlayer("p4", "West", RoleVillager, AlignmentGood),
		testAIPlayer("p5", "East", RoleVillager, AlignmentGood),
		testAIPlayer("p6", "Moon", RoleVillager, AlignmentGood),
	})
	room.HostUserID = "u1"
	room.Winner = AlignmentGood
	room.WinnerMessage = "上一局好人胜利。"
	room.Log = []LogEntry{{ID: "log_old", Text: "上一局遗留桌面记录。"}}
	room.Speeches = []SpeechEntry{{ID: "speech_old", PlayerID: "p2", PlayerName: "North", Text: "上一局我是狼。", SpokenAt: time.Now().UTC()}}
	room.LastAISpeechSourceID = "speech_old"
	room.ActionSeq = 12
	room.RecentActions = []PublicAction{{Seq: 12, Type: "old", Message: "上一局动作。"}}
	room.AIDebugTraces = []AIDebugTrace{{ID: "ai_trace_old", PlayerID: "p2", PlayerName: "North", Thinking: "上一局 thinking"}}
	room.PlayerNotes = map[string]map[string]string{
		"p1": {"p2": "主人自己的跨局备注"},
		"p2": {"p1": "AI 上局记住了房主身份"},
	}
	manager.rooms[room.ID] = room
	manager.aiSessions[socialAISessionKey(room.ID, "p2")] = &socialAISession{
		Game:     GameWerewolf,
		RoomID:   room.ID,
		PlayerID: "p2",
		Memory:   []string{"上一局投过 p1"},
	}

	if _, err := manager.Start(room.ID, "u1"); err != nil {
		t.Fatalf("restart room: %v", err)
	}

	if room.Winner != "" || room.WinnerMessage != "" {
		t.Fatalf("expected winner state to reset, got winner=%q message=%q", room.Winner, room.WinnerMessage)
	}
	if len(room.Speeches) != 0 {
		t.Fatalf("expected previous speeches to be cleared, got %+v", room.Speeches)
	}
	if room.LastAISpeechSourceID != "" {
		t.Fatalf("expected AI speech source to reset, got %q", room.LastAISpeechSourceID)
	}
	if _, ok := manager.aiSessions[socialAISessionKey(room.ID, "p2")]; ok {
		t.Fatal("expected AI session memory to be removed on restart")
	}
	if _, ok := room.PlayerNotes["p2"]; ok {
		t.Fatalf("expected AI private notes to be cleared, got %+v", room.PlayerNotes["p2"])
	}
	if got := room.PlayerNotes["p1"]["p2"]; got != "主人自己的跨局备注" {
		t.Fatalf("expected human private note to persist, got %q", got)
	}
	if len(room.Log) != 1 || strings.Contains(room.Log[0].Text, "上一局") {
		t.Fatalf("expected previous table log to be replaced by new start log, got %+v", room.Log)
	}
	if len(room.RecentActions) != 1 || room.RecentActions[0].Seq != 1 || room.ActionSeq != 1 {
		t.Fatalf("expected public actions to restart from first action, seq=%d actions=%+v", room.ActionSeq, room.RecentActions)
	}
	if len(room.AIDebugTraces) != 0 {
		t.Fatalf("expected AI debug traces to be cleared, got %+v", room.AIDebugTraces)
	}
}

func TestAIDebugTraceOnlyVisibleInGodView(t *testing.T) {
	provider := &fakeDecisionProvider{
		enabled: true,
		decision: aiplayer.Decision{
			ActionID: "vote:seat_1",
			Reason:   "二号票型异常",
			Speech:   "我先压二号看反应。",
			Thinking: "这里是模型 thinking，只能给管理员上帝视角看。",
			Source:   "llm",
		},
	}
	manager := NewManager(GameWerewolf, provider)
	room := testWerewolfRoom("WWFTRACE", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Host", RoleVillager, AlignmentGood),
		testAIPlayer("p2", "North", RoleWerewolf, AlignmentEvil),
		testAIPlayer("p3", "South", RoleVillager, AlignmentGood),
	})
	room.HostUserID = "u1"
	manager.rooms[room.ID] = room

	if _, _, err := manager.RunAIAction(room.ID); err != nil {
		t.Fatalf("run ai action: %v", err)
	}
	if len(room.AIDebugTraces) != 1 {
		t.Fatalf("expected one AI debug trace, got %+v", room.AIDebugTraces)
	}

	normalView := manager.publicRoom(room, "u1")
	if len(normalView.AIDebugTraces) != 0 {
		t.Fatalf("expected normal view to hide AI debug traces, got %+v", normalView.AIDebugTraces)
	}

	godView := manager.publicRoomWithOptions(room, "u1", PublicRoomOptions{GodViewAvailable: true, GodView: true})
	if len(godView.AIDebugTraces) != 1 {
		t.Fatalf("expected god view to include AI debug trace, got %+v", godView.AIDebugTraces)
	}
	trace := godView.AIDebugTraces[0]
	if trace.Thinking == "" || !trace.ThinkingAvailable {
		t.Fatalf("expected thinking to be visible in god view trace, got %+v", trace)
	}
	if trace.ActionID != "vote:seat_1" || trace.Reason == "" || trace.Speech == "" || len(trace.Actions) == 0 {
		t.Fatalf("expected trace to include decision debug fields, got %+v", trace)
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

func TestAvalonRolesIncludeMainSpecials(t *testing.T) {
	roles := avalonRoles(10)
	for _, role := range []Role{RoleMerlin, RolePercival, RoleAssassin, RoleMorgana, RoleMordred, RoleOberon} {
		if !roleInSlice(roles, role) {
			t.Fatalf("expected 10-player avalon role set to include %q, got %#v", role, roles)
		}
	}
	if evilRoleCount(roles) != 4 {
		t.Fatalf("expected 4 evil roles in 10-player avalon, got %d from %#v", evilRoleCount(roles), roles)
	}
}

func TestAvalonSpecialRoleVisibility(t *testing.T) {
	manager := NewManager(GameAvalon, nil)
	room := &Room{
		ID:         "AVLSEER",
		Game:       GameAvalon,
		HostUserID: "u1",
		Phase:      PhaseAvalonTeam,
		Players: []*Player{
			testPlayer("p1", "u1", "Merlin", RoleMerlin, AlignmentGood),
			testPlayer("p2", "u2", "Percival", RolePercival, AlignmentGood),
			testPlayer("p3", "u3", "Morgana", RoleMorgana, AlignmentEvil),
			testPlayer("p4", "u4", "Mordred", RoleMordred, AlignmentEvil),
			testPlayer("p5", "u5", "Oberon", RoleOberon, AlignmentEvil),
			testPlayer("p6", "u6", "Assassin", RoleAssassin, AlignmentEvil),
			testPlayer("p7", "u7", "Loyal", RoleLoyal, AlignmentGood),
		},
		Avalon: AvalonState{Round: 1, LeaderID: "p1", TeamVotes: map[string]bool{}, QuestCards: map[string]string{}, RequiredTeam: 2, RequiredFails: 1},
	}

	merlinView := manager.publicRoom(room, "u1")
	if merlinView.Players[2].Role != RoleMorgana || merlinView.Players[4].Role != RoleOberon {
		t.Fatalf("expected Merlin to see Morgana and Oberon, got %#v", merlinView.Players)
	}
	if merlinView.Players[3].Role != "" {
		t.Fatalf("expected Merlin not to see Mordred, got %q", merlinView.Players[3].Role)
	}

	assassinView := manager.publicRoom(room, "u6")
	if assassinView.Players[2].Role != RoleMorgana || assassinView.Players[3].Role != RoleMordred {
		t.Fatalf("expected evil players to see non-Oberon evil teammates, got %#v", assassinView.Players)
	}
	if assassinView.Players[4].Role != "" {
		t.Fatalf("expected Oberon to be hidden from evil teammates, got %q", assassinView.Players[4].Role)
	}

	oberonView := manager.publicRoom(room, "u5")
	if oberonView.Players[2].Role != "" || oberonView.Players[5].Role != "" {
		t.Fatalf("expected Oberon not to see evil teammates, got %#v", oberonView.Players)
	}

	percivalView := manager.publicRoom(room, "u2")
	if percivalView.Players[0].Role != "" || percivalView.Players[2].Role != "" {
		t.Fatalf("expected Percival marks to avoid exact role reveals, got %#v", percivalView.Players)
	}
	if !roleIDInSlice(percivalView.Avalon.PercivalMarks, "p1") || !roleIDInSlice(percivalView.Avalon.PercivalMarks, "p3") {
		t.Fatalf("expected Percival to receive Merlin/Morgana marks, got %#v", percivalView.Avalon.PercivalMarks)
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

func roleInSlice(roles []Role, expected Role) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func roleIDInSlice(ids []string, expected string) bool {
	for _, id := range ids {
		if id == expected {
			return true
		}
	}
	return false
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

func TestWerewolfSeerCanOnlyCheckOncePerNight(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFSEER1", PhaseWerewolfNight, []*Player{
		testPlayer("seer", "u_seer", "Seer", RoleSeer, AlignmentGood),
		testPlayer("villager", "u_villager", "Villager", RoleVillager, AlignmentGood),
		testPlayer("wolf", "u_wolf", "Wolf", RoleWerewolf, AlignmentEvil),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u_seer", "target:villager"); err != nil {
		t.Fatalf("first seer check should succeed: %v", err)
	}
	if got := room.Werewolf.SeerChecks["villager"]; got != AlignmentGood {
		t.Fatalf("expected first checked alignment to be stored, got %q", got)
	}
	if _, err := manager.NightAction(room.ID, "u_seer", "target:wolf"); err == nil {
		t.Fatal("expected second seer check in the same night to fail")
	}
	if _, ok := room.Werewolf.SeerChecks["wolf"]; ok {
		t.Fatal("expected second target not to be checked")
	}
	if actions := werewolfNightActions(room, room.Players[0]); len(actions) != 0 {
		t.Fatalf("expected no seer actions after submitting, got %+v", actions)
	}
}

func TestWerewolfOptionalSpeechStateIncludesNightResultAndDeaths(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFSPEECHFACTS", PhaseWerewolfDay, []*Player{
		testPlayer("dead", "u_dead", "Dead Player", RoleVillager, AlignmentGood),
		testAIPlayer("speaker", "Speaker Bot", RoleVillager, AlignmentGood),
		testPlayer("wolf", "u_wolf", "Wolf", RoleWerewolf, AlignmentEvil),
	})
	room.Players[0].Alive = false
	room.Werewolf.LastNight = "Dead Player 在夜晚出局。"

	state := manager.aiSpeechState(room, room.Players[1])
	if state["lastNight"] != "座位 1 在夜晚出局。" {
		t.Fatalf("expected optional speech state to include lastNight, got %+v", state["lastNight"])
	}
	facts, ok := state["publicFacts"].([]string)
	if !ok {
		t.Fatalf("expected publicFacts in optional speech state, got %+v", state["publicFacts"])
	}
	joinedFacts := strings.Join(facts, "\n")
	if !strings.Contains(joinedFacts, "座位 1 在夜晚出局") || !strings.Contains(joinedFacts, "已出局玩家") {
		t.Fatalf("expected public facts to include death result and out players, got %q", joinedFacts)
	}
	guide := fmt.Sprint(state["speechGuide"])
	if !strings.Contains(guide, "绝不能说平安夜") {
		t.Fatalf("expected speech guide to forbid false peaceful night claims, got %q", guide)
	}
}

func TestWerewolfAIStateIncludesRevealedIdiotPublicFact(t *testing.T) {
	room := testWerewolfRoom("WWFIDIOTFACT", PhaseWerewolfNight, []*Player{
		testPlayer("idiot", "u_idiot", "Idiot Player", RoleIdiot, AlignmentGood),
		testAIPlayer("witch", "Witch Bot", RoleWitch, AlignmentGood),
		testPlayer("wolf", "u_wolf", "Wolf", RoleWerewolf, AlignmentEvil),
	})
	room.Werewolf.RevealedIdiots = map[string]bool{"idiot": true}

	state := werewolfAIState(room, room.Players[1])
	facts, ok := state["publicFacts"].([]string)
	if !ok {
		t.Fatalf("expected publicFacts in werewolf ai state, got %+v", state["publicFacts"])
	}
	joinedFacts := strings.Join(facts, "\n")
	if !strings.Contains(joinedFacts, "座位 1") || !strings.Contains(joinedFacts, "已公开翻牌为白痴") || !strings.Contains(joinedFacts, "不是自爆或伪装") {
		t.Fatalf("expected public facts to explain revealed idiot rule, got %q", joinedFacts)
	}

	players, ok := state["players"].([]map[string]any)
	if !ok || len(players) == 0 {
		t.Fatalf("expected players in werewolf ai state, got %+v", state["players"])
	}
	if players[0]["role"] != RoleIdiot || players[0]["alignment"] != AlignmentGood {
		t.Fatalf("expected revealed idiot role to be visible, got %+v", players[0])
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
	for _, name := range []string{"snowykami", "北风", "南星"} {
		if strings.Contains(serialized, name) {
			t.Fatalf("expected social LLM input to hide player display name %q, got %s", name, serialized)
		}
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
	for _, expected := range []string{"真实玩家", "狼人杀目标", "夜晚行动", "避免模板话", "不能直接或间接泄露隐藏身份", "最近发言只是待验证的桌面主张", "独立判断"} {
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

func TestUndercoverPublicRoomHidesRolesUntilFinished(t *testing.T) {
	manager := NewManager(GameUndercover, nil)
	room := &Room{
		ID:    "UNDHIDE",
		Game:  GameUndercover,
		Phase: PhaseUndercoverDescribe,
		Players: []*Player{
			testPlayer("p1", "u1", "Host", RoleUndercover, AlignmentEvil),
			testPlayer("p2", "u2", "Guest", RoleCivilian, AlignmentGood),
			testPlayer("p3", "u3", "Guest Two", RoleCivilian, AlignmentGood),
			testPlayer("p4", "u4", "Guest Three", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{
			Round:            1,
			WordPair:         UndercoverWordPair{ID: "pair-1", CivilianWord: "苹果", UndercoverWord: "梨", Category: "水果"},
			CurrentSpeakerID: "p1",
			Described:        map[string]bool{},
			Votes:            map[string]UndercoverVoteIntent{},
		},
	}

	view := manager.publicRoomWithOptions(room, "u1", PublicRoomOptions{GodViewAvailable: true, GodView: true})
	for _, player := range view.Players {
		if player.Role != "" || player.Alignment != "" || player.VisibleToYou {
			t.Fatalf("expected undercover roles to stay hidden during game, got %+v", player)
		}
	}
	if view.Undercover.YourWord != "梨" {
		t.Fatalf("expected neutral own word only, got %q", view.Undercover.YourWord)
	}
	if view.Undercover.WordPair.CivilianWord != "" || view.Undercover.WordPair.UndercoverWord != "" {
		t.Fatalf("expected public word pair to hide role-specific words, got %+v", view.Undercover.WordPair)
	}

	room.Phase = PhaseFinished
	finishedView := manager.publicRoom(room, "u1")
	if finishedView.Players[0].Role != RoleUndercover || finishedView.Players[1].Role != RoleCivilian {
		t.Fatalf("expected roles to reveal after finish, got %+v", finishedView.Players)
	}
	if finishedView.Undercover.WordPair.CivilianWord != "苹果" || finishedView.Undercover.WordPair.UndercoverWord != "梨" {
		t.Fatalf("expected final words to reveal after finish, got %+v", finishedView.Undercover.WordPair)
	}
}

func TestUndercoverDomainsGenerateLargeWordBank(t *testing.T) {
	presets := undercoverPresets()
	if len(presets) < 6 {
		t.Fatalf("expected multiple undercover domains, got %d", len(presets))
	}
	if total := undercoverTotalGroupCount(); total < len(presets)*minUndercoverGroupsPerDomain {
		t.Fatalf("expected at least %d undercover word groups, got %d", len(presets)*minUndercoverGroupsPerDomain, total)
	}
	computingGroups := undercoverGroupsForDomain("computing")
	if len(computingGroups) < minUndercoverGroupsPerDomain {
		t.Fatalf("expected computing domain to have %d groups, got %d", minUndercoverGroupsPerDomain, len(computingGroups))
	}
	foundTCPUDP := false
	for _, group := range computingGroups {
		if len(group.Words) >= 2 && group.Words[0] == "TCP" && group.Words[1] == "UDP" {
			foundTCPUDP = true
			break
		}
	}
	if !foundTCPUDP {
		t.Fatalf("expected computing domain to include TCP/UDP group")
	}
	for _, preset := range presets {
		if len(preset.Pairs) > 0 {
			t.Fatalf("lobby domain metadata should not expose full word bank, got %d pairs for %s", len(preset.Pairs), preset.ID)
		}
		if preset.PairCount == 0 {
			t.Fatalf("expected group count for domain %s", preset.ID)
		}
		if preset.PairCount < minUndercoverGroupsPerDomain {
			t.Fatalf("expected at least %d groups for domain %s, got %d", minUndercoverGroupsPerDomain, preset.ID, preset.PairCount)
		}
	}
}

func TestChooseUndercoverPairDrawsTwoWordsFromMultiWordGroup(t *testing.T) {
	pairSeen := map[string]bool{}
	group := undercoverWordGroup{
		DomainID:   "computing",
		Category:   "计算机、网络和 AI 等",
		GroupIndex: 155,
		Words:      []string{"大语言模型", "多模态模型", "视觉语言模型", "语音模型"},
	}
	for range 80 {
		pair := chooseUndercoverPairFromGroup(group)
		if pair.CivilianWord == pair.UndercoverWord {
			t.Fatalf("expected different words, got %+v", pair)
		}
		if !strings.HasPrefix(pair.ID, "computing-155-") {
			t.Fatalf("expected stable group id prefix, got %q", pair.ID)
		}
		pairSeen[pair.CivilianWord] = true
		pairSeen[pair.UndercoverWord] = true
	}
	if len(pairSeen) < 3 {
		t.Fatalf("expected multi-word group to draw different words over repeated picks, got %v", pairSeen)
	}
}

func TestUndercoverConfigSupportsMultipleDomains(t *testing.T) {
	manager := NewManager(GameUndercover, nil)
	room := &Room{
		ID:         "UNDDOMAINS",
		Game:       GameUndercover,
		HostUserID: "u1",
		Phase:      PhaseLobby,
		Players: []*Player{
			testPlayer("p1", "u1", "Host", RoleCivilian, AlignmentGood),
			testPlayer("p2", "u2", "Guest", RoleCivilian, AlignmentGood),
			testPlayer("p3", "u3", "Guest Two", RoleCivilian, AlignmentGood),
			testPlayer("p4", "u4", "Guest Three", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{DomainIDs: []string{defaultUndercoverPresetID()}, Described: map[string]bool{}, Votes: map[string]UndercoverVoteIntent{}},
	}
	manager.rooms[room.ID] = room

	view, err := manager.UpdateUndercoverConfig(room.ID, "u1", []string{"computing", "academic"}, true)
	if err != nil {
		t.Fatalf("update domains: %v", err)
	}
	if got := strings.Join(view.Undercover.DomainIDs, ","); got != "computing,academic" {
		t.Fatalf("expected selected domains to persist in public view, got %q", got)
	}
	if !view.Undercover.IncludeBlank {
		t.Fatalf("expected include blank to persist")
	}
	if _, err := manager.UpdateUndercoverConfig(room.ID, "u1", []string{"unknown"}, false); err == nil || err.Error() != "invalid_undercover_domain" {
		t.Fatalf("expected invalid domain error, got %v", err)
	}

	startUndercover(room)
	if got := strings.Join(room.Undercover.DomainIDs, ","); got != "computing,academic" {
		t.Fatalf("expected selected domains to survive start, got %q", got)
	}
	if room.Undercover.WordPair.CivilianWord == "" || room.Undercover.WordPair.UndercoverWord == "" {
		t.Fatalf("expected selected domains to produce a word pair, got %+v", room.Undercover.WordPair)
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

func TestWerewolfNightRequiresEveryLivingWolfAction(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFALLWOLVES", PhaseWerewolfNight, []*Player{
		testPlayer("wolf_1", "u1", "Wolf One", RoleWerewolf, AlignmentEvil),
		testPlayer("wolf_2", "u2", "Wolf Two", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager One", RoleVillager, AlignmentGood),
		testPlayer("p4", "u4", "Villager Two", RoleVillager, AlignmentGood),
		testPlayer("p5", "u5", "Villager Three", RoleVillager, AlignmentGood),
		testPlayer("p6", "u6", "Villager Four", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u1", "target:p3"); err != nil {
		t.Fatalf("first wolf action: %v", err)
	}
	if room.Phase != PhaseWerewolfNight || !room.Players[2].Alive {
		t.Fatalf("expected night to wait for second wolf, phase=%q p3Alive=%v", room.Phase, room.Players[2].Alive)
	}
	if pending := pendingWerewolfNightActions(room); len(pending) != 1 || !strings.Contains(pending[0], "wolf_2") {
		t.Fatalf("expected second wolf to be pending, got %+v", pending)
	}

	if _, err := manager.NightAction(room.ID, "u2", "target:p3"); err != nil {
		t.Fatalf("second wolf action: %v", err)
	}
	if room.Phase != PhaseWerewolfDay || room.Players[2].Alive {
		t.Fatalf("expected night to resolve after both wolves act, phase=%q p3Alive=%v", room.Phase, room.Players[2].Alive)
	}
}

func TestWerewolfNightRequiresWolfConsensusBeforeKill(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWOLFTIE", PhaseWerewolfNight, []*Player{
		testPlayer("wolf_1", "u1", "Wolf One", RoleWerewolf, AlignmentEvil),
		testPlayer("wolf_2", "u2", "Wolf Two", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager One", RoleVillager, AlignmentGood),
		testPlayer("p4", "u4", "Villager Two", RoleVillager, AlignmentGood),
		testPlayer("p5", "u5", "Villager Three", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u1", "target:p3"); err != nil {
		t.Fatalf("first wolf action: %v", err)
	}
	if _, err := manager.NightAction(room.ID, "u2", "target:p4"); err != nil {
		t.Fatalf("second wolf action: %v", err)
	}
	if room.Phase != PhaseWerewolfNight {
		t.Fatalf("expected night to continue until wolves agree, got %q", room.Phase)
	}
	if !room.Players[2].Alive || !room.Players[3].Alive {
		t.Fatalf("expected no one to die before consensus, p3=%v p4=%v", room.Players[2].Alive, room.Players[3].Alive)
	}
	if pending := pendingWerewolfNightActions(room); len(pending) == 0 || pending[len(pending)-1] != "werewolf_consensus:team" {
		t.Fatalf("expected wolf consensus to be pending, got %+v", pending)
	}

	if _, err := manager.NightAction(room.ID, "u2", "target:p3"); err != nil {
		t.Fatalf("second wolf realign: %v", err)
	}
	if room.Phase == PhaseWerewolfNight || room.Players[2].Alive {
		t.Fatalf("expected consensus kill to resolve, phase=%q p3Alive=%v", room.Phase, room.Players[2].Alive)
	}
}

func TestWerewolfWitchWaitsForAllWolvesBeforeSeeingVictim(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWITCHWOLVES", PhaseWerewolfNight, []*Player{
		testPlayer("wolf_1", "u1", "Wolf One", RoleWerewolf, AlignmentEvil),
		testPlayer("wolf_2", "u2", "Wolf Two", RoleWerewolf, AlignmentEvil),
		testPlayer("witch", "u3", "Witch", RoleWitch, AlignmentGood),
		testPlayer("p4", "u4", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u1", "target:p4"); err != nil {
		t.Fatalf("first wolf action: %v", err)
	}
	if victim := currentWerewolfKillTarget(room); victim != "" {
		t.Fatalf("expected no visible wolf victim until every wolf acts, got %q", victim)
	}
	if actions := werewolfNightActions(room, room.Players[2]); len(actions) != 0 {
		t.Fatalf("expected witch to wait for all wolves, got %+v", actions)
	}

	if _, err := manager.NightAction(room.ID, "u2", "target:p4"); err != nil {
		t.Fatalf("second wolf action: %v", err)
	}
	if victim := currentWerewolfKillTarget(room); victim != "p4" {
		t.Fatalf("expected visible victim after all wolves agree, got %q", victim)
	}
	if actions := werewolfNightActions(room, room.Players[2]); len(actions) == 0 {
		t.Fatal("expected witch actions after all wolves submit")
	}
}

func TestWerewolfWitchCannotActBeforeWolfTarget(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWITCHWAIT", PhaseWerewolfNight, []*Player{
		testPlayer("p1", "u1", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p2", "u2", "Witch", RoleWitch, AlignmentGood),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if actions := werewolfNightActions(room, room.Players[1]); len(actions) != 0 {
		t.Fatalf("expected witch to wait before wolf target is known, got %+v", actions)
	}
	if _, err := manager.NightAction(room.ID, "u2", "skip:witch"); err == nil {
		t.Fatal("expected witch action before wolf target to fail")
	}

	if _, err := manager.NightAction(room.ID, "u1", "target:p3"); err != nil {
		t.Fatalf("wolf action: %v", err)
	}
	if room.Phase != PhaseWerewolfNight {
		t.Fatalf("expected night to wait for witch after wolf target, got %q", room.Phase)
	}
	if actions := werewolfNightActions(room, room.Players[1]); len(actions) == 0 {
		t.Fatal("expected witch actions after wolf target is known")
	}
}

func TestWerewolfNightAllowsWolfConsensusSkip(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFSKIPKILL", PhaseWerewolfNight, []*Player{
		testPlayer("wolf_1", "u1", "Wolf One", RoleWerewolf, AlignmentEvil),
		testPlayer("wolf_2", "u2", "Wolf Two", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager One", RoleVillager, AlignmentGood),
		testPlayer("p4", "u4", "Villager Two", RoleVillager, AlignmentGood),
		testPlayer("p5", "u5", "Villager Three", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.NightAction(room.ID, "u1", werewolfSkipActionID); err != nil {
		t.Fatalf("first wolf skip: %v", err)
	}
	if room.Phase != PhaseWerewolfNight {
		t.Fatalf("expected night to wait for second wolf, got %q", room.Phase)
	}
	if _, err := manager.NightAction(room.ID, "u2", werewolfSkipActionID); err != nil {
		t.Fatalf("second wolf skip: %v", err)
	}
	if room.Phase != PhaseWerewolfDay {
		t.Fatalf("expected unanimous skip to advance to day, got %q", room.Phase)
	}
	if room.Werewolf.LastNight != "昨夜无人出局。" {
		t.Fatalf("expected no-death night result, got %q", room.Werewolf.LastNight)
	}
}

func TestWerewolfWolfSpeechOnlyVisibleToWolvesAndGodView(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFWOLFCHAT", PhaseWerewolfNight, []*Player{
		testPlayer("wolf_1", "u1", "Wolf One", RoleWerewolf, AlignmentEvil),
		testPlayer("wolf_2", "u2", "Wolf Two", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfWolfSpeech(room.ID, "u1", "先压 3 号，看看女巫救不救。"); err != nil {
		t.Fatalf("wolf speech: %v", err)
	}
	if _, err := manager.NightAction(room.ID, "u1", "target:p3"); err != nil {
		t.Fatalf("wolf action: %v", err)
	}

	wolfView := manager.publicRoom(room, "u2")
	if len(wolfView.Werewolf.WolfSpeeches) != 1 || wolfView.Werewolf.WolfNightActions["wolf_1"] != "p3" {
		t.Fatalf("expected wolf view to include wolf chat and choices, got speeches=%+v actions=%+v", wolfView.Werewolf.WolfSpeeches, wolfView.Werewolf.WolfNightActions)
	}
	villagerView := manager.publicRoom(room, "u3")
	if len(villagerView.Werewolf.WolfSpeeches) != 0 || len(villagerView.Werewolf.WolfNightActions) != 0 {
		t.Fatalf("expected villager view to hide wolf chat and choices, got speeches=%+v actions=%+v", villagerView.Werewolf.WolfSpeeches, villagerView.Werewolf.WolfNightActions)
	}
	godView := manager.publicRoomWithOptions(room, "u3", PublicRoomOptions{GodViewAvailable: true, GodView: true})
	if len(godView.Werewolf.WolfSpeeches) != 1 || godView.Werewolf.WolfNightActions["wolf_1"] != "p3" {
		t.Fatalf("expected god view to include wolf chat and choices, got speeches=%+v actions=%+v", godView.Werewolf.WolfSpeeches, godView.Werewolf.WolfNightActions)
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

func TestWerewolfVoteTieExilesNoOne(t *testing.T) {
	manager := NewManager(GameWerewolf, nil)
	room := testWerewolfRoom("WWFVOTETIE", PhaseWerewolfVote, []*Player{
		testPlayer("p1", "u1", "Villager One", RoleVillager, AlignmentGood),
		testPlayer("p2", "u2", "Wolf", RoleWerewolf, AlignmentEvil),
		testPlayer("p3", "u3", "Villager Two", RoleVillager, AlignmentGood),
		testPlayer("p4", "u4", "Villager Three", RoleVillager, AlignmentGood),
	})
	manager.rooms[room.ID] = room

	if _, err := manager.WerewolfVote(room.ID, "u1", "p2", true); err != nil {
		t.Fatalf("p1 vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u2", "p1", true); err != nil {
		t.Fatalf("p2 vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u3", "p1", true); err != nil {
		t.Fatalf("p3 vote: %v", err)
	}
	if _, err := manager.WerewolfVote(room.ID, "u4", "p2", true); err != nil {
		t.Fatalf("p4 vote: %v", err)
	}
	if room.Phase != PhaseWerewolfNight {
		t.Fatalf("expected tied vote to move to next night, got %q", room.Phase)
	}
	if !room.Players[0].Alive || !room.Players[1].Alive {
		t.Fatalf("expected tied vote to exile nobody, p1=%v p2=%v", room.Players[0].Alive, room.Players[1].Alive)
	}
	foundTieLog := false
	for _, entry := range room.Log {
		if strings.Contains(entry.Text, "平票") {
			foundTieLog = true
			break
		}
	}
	if !foundTieLog {
		t.Fatalf("expected tie log, got %+v", room.Log)
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
			DomainIDs: []string{"computing", "academic"},
			WordPair:  UndercoverWordPair{CivilianWord: "TCP", UndercoverWord: "UDP", Category: "计算机与网络"},
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
		"yourRole",
	})
	if strings.Contains(string(payload), `"role":"undercover"`) || strings.Contains(string(payload), `"role":"blank"`) {
		t.Fatalf("undercover context leaked other hidden roles: %s", string(payload))
	}
	if !strings.Contains(string(payload), "计算机与网络") || !strings.Contains(string(payload), "学术与校园") {
		t.Fatalf("expected undercover AI context to include selected domain names, got %s", string(payload))
	}
	if !strings.Contains(string(payload), `"wordCategory":"计算机与网络"`) {
		t.Fatalf("expected undercover AI context to include current word category, got %s", string(payload))
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

func TestSeatAliasesUseAssignedSeatOverJoinOrder(t *testing.T) {
	room := testWerewolfRoom("WWFSEATS", PhaseWerewolfDay, []*Player{
		testPlayer("host", "u1", "Host", RoleVillager, AlignmentGood),
		testPlayer("guest", "u2", "Guest", RoleWerewolf, AlignmentEvil),
		testPlayer("third", "u3", "Third", RoleVillager, AlignmentGood),
	})
	room.Players[0].Seat = 2
	room.Players[1].Seat = 0
	room.Players[2].Seat = 1

	if got := aiPlayerNumber(room, room.Players[0]); got != 3 {
		t.Fatalf("expected host to use assigned seat 3, got %d", got)
	}
	if got := aiPlayerNumber(room, room.Players[1]); got != 1 {
		t.Fatalf("expected guest to use assigned seat 1, got %d", got)
	}
	if player := playerFromAIRef(room, "seat_2"); player == nil || player.ID != "third" {
		t.Fatalf("expected seat_2 to resolve by assigned seat, got %+v", player)
	}
}

func TestSocialStartOrderUsesAssignedSeats(t *testing.T) {
	avalonRoom := &Room{
		ID:    "AVLSEATS",
		Game:  GameAvalon,
		Phase: PhaseLobby,
		Players: []*Player{
			testPlayer("host", "u1", "Host", RoleLoyal, AlignmentGood),
			testPlayer("seat_one", "u2", "Seat One", RoleLoyal, AlignmentGood),
			testPlayer("seat_two", "u3", "Seat Two", RoleLoyal, AlignmentGood),
			testPlayer("seat_three", "u4", "Seat Three", RoleLoyal, AlignmentGood),
			testPlayer("seat_four", "u5", "Seat Four", RoleLoyal, AlignmentGood),
		},
	}
	for index, seat := range []int{4, 0, 1, 2, 3} {
		avalonRoom.Players[index].Seat = seat
	}
	startAvalon(avalonRoom)
	if avalonRoom.Avalon.LeaderID != "seat_one" {
		t.Fatalf("expected first Avalon leader to be assigned seat 1, got %q", avalonRoom.Avalon.LeaderID)
	}

	undercoverRoom := &Room{
		ID:    "UNDSEATS",
		Game:  GameUndercover,
		Phase: PhaseLobby,
		Players: []*Player{
			testPlayer("host", "u1", "Host", RoleCivilian, AlignmentGood),
			testPlayer("seat_one", "u2", "Seat One", RoleCivilian, AlignmentGood),
			testPlayer("seat_two", "u3", "Seat Two", RoleCivilian, AlignmentGood),
			testPlayer("seat_three", "u4", "Seat Three", RoleCivilian, AlignmentGood),
		},
		Undercover: UndercoverState{PresetID: defaultUndercoverPresetID()},
	}
	for index, seat := range []int{3, 0, 1, 2} {
		undercoverRoom.Players[index].Seat = seat
	}
	startUndercover(undercoverRoom)
	if undercoverRoom.Undercover.CurrentSpeakerID != "seat_one" {
		t.Fatalf("expected first Undercover speaker to be assigned seat 1, got %q", undercoverRoom.Undercover.CurrentSpeakerID)
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
	actionCountAfterConfirm := len(room.RecentActions)
	if _, err := manager.UndercoverVote(room.ID, "u1", "p3", true); err != nil {
		t.Fatalf("duplicate confirm should be idempotent: %v", err)
	}
	if len(room.RecentActions) != actionCountAfterConfirm {
		t.Fatalf("duplicate confirm should not record another vote action")
	}
	if _, err := manager.UndercoverVote(room.ID, "u1", "p2", false); err == nil || err.Error() != "vote_already_confirmed" {
		t.Fatalf("expected confirmed voter cannot change selection, got %v", err)
	}
	if vote := room.Undercover.Votes["p1"]; vote.TargetID != "p3" || !vote.Confirmed {
		t.Fatalf("expected confirmed vote to stay locked, got %+v", vote)
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
	room.Werewolf.NightActions["ai_wolf"] = "human_villager"
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
	room.Werewolf.NightActions["ai_wolf"] = "human_villager"
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
	room.Werewolf.NightActions["ai_wolf"] = "human_villager"
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
	room := &Room{
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
	assignSequentialTestSeats(room)
	return room
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

func assignSequentialTestSeats(room *Room) {
	for index, player := range room.Players {
		player.Seat = index
	}
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
