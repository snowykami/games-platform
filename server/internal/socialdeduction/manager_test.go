package socialdeduction

import (
	"fmt"
	"testing"
	"time"
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
