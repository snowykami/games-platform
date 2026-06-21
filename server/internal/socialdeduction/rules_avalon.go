package socialdeduction

import "fmt"

func startAvalon(room *Room) {
	roles := avalonRoles(len(room.Players))
	for index, player := range shuffledPlayers(room.Players) {
		player.Role = roles[index]
		player.Alignment = avalonAlignment(player.Role)
	}
	room.Phase = PhaseAvalonTeam
	room.Avalon = AvalonState{
		Round:         1,
		LeaderID:      firstSeatPlayerID(room),
		TeamVotes:     map[string]bool{},
		QuestCards:    map[string]string{},
		RequiredTeam:  avalonTeamSize(len(room.Players), 1),
		RequiredFails: avalonRequiredFails(len(room.Players), 1),
	}
	room.Log = append(room.Log, createLog("阿瓦隆开始，队长提名第一支任务队伍。"))
	recordAction(room, PublicAction{Type: "start", Message: "阿瓦隆开始，进入组队阶段。"})
}

func avalonRoles(count int) []Role {
	evil := map[int]int{5: 2, 6: 2, 7: 3, 8: 3, 9: 3, 10: 4}[count]
	roles := []Role{RoleMerlin, RoleAssassin}
	for len(roles) < evil+1 {
		roles = append(roles, RoleMinion)
	}
	for len(roles) < count {
		roles = append(roles, RoleLoyal)
	}
	return roles
}

func avalonAlignment(role Role) Alignment {
	if role == RoleAssassin || role == RoleMinion {
		return AlignmentEvil
	}
	return AlignmentGood
}

func (m *Manager) resolveAvalonTeamVote(room *Room) {
	if len(room.Avalon.TeamVotes) < len(room.Players) {
		return
	}
	approve := 0
	for _, vote := range room.Avalon.TeamVotes {
		if vote {
			approve++
		}
	}
	if approve > len(room.Players)/2 {
		room.Phase = PhaseAvalonQuest
		room.Avalon.QuestCards = map[string]string{}
		room.Log = append(room.Log, createLog("任务队伍通过，队员开始提交任务牌。"))
		recordAction(room, PublicAction{Type: "team_approved", Message: "任务队伍通过。"})
		return
	}

	room.Avalon.RejectedTeams++
	if room.Avalon.RejectedTeams >= 5 {
		finish(room, AlignmentEvil, "连续五次组队失败，邪恶阵营获胜。")
		return
	}
	room.Phase = PhaseAvalonTeam
	room.Avalon.Team = nil
	room.Avalon.TeamVotes = map[string]bool{}
	advanceAvalonLeader(room)
	room.Log = append(room.Log, createLog("任务队伍未通过，下一位队长重新提名。"))
	recordAction(room, PublicAction{Type: "team_rejected", Message: "任务队伍未通过。"})
}

func (m *Manager) resolveAvalonQuest(room *Room) {
	if len(room.Avalon.QuestCards) < len(room.Avalon.Team) {
		return
	}
	failCards := 0
	for _, card := range room.Avalon.QuestCards {
		if card == "fail" {
			failCards++
		}
	}
	result := AvalonQuestResult{Round: room.Avalon.Round, TeamSize: len(room.Avalon.Team), FailCards: failCards}
	room.Avalon.QuestResults = append(room.Avalon.QuestResults, result)
	if failCards >= room.Avalon.RequiredFails {
		room.Avalon.Fails++
		room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 次任务失败。", room.Avalon.Round)))
	} else {
		room.Avalon.Successes++
		room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 次任务成功。", room.Avalon.Round)))
	}

	if room.Avalon.Fails >= 3 {
		finish(room, AlignmentEvil, "三次任务失败，邪恶阵营获胜。")
		return
	}
	if room.Avalon.Successes >= 3 {
		room.Phase = PhaseAssassination
		room.Log = append(room.Log, createLog("正义阵营完成三次任务，刺客最后寻找梅林。"))
		recordAction(room, PublicAction{Type: "assassination_started", Message: "刺杀阶段开始。"})
		return
	}

	room.Avalon.Round++
	room.Avalon.Team = nil
	room.Avalon.TeamVotes = map[string]bool{}
	room.Avalon.QuestCards = map[string]string{}
	room.Avalon.RequiredTeam = avalonTeamSize(len(room.Players), room.Avalon.Round)
	room.Avalon.RequiredFails = avalonRequiredFails(len(room.Players), room.Avalon.Round)
	room.Phase = PhaseAvalonTeam
	advanceAvalonLeader(room)
	recordAction(room, PublicAction{Type: "quest_resolved", Message: "任务结算完成。"})
}

func avalonTeamSize(players int, round int) int {
	table := map[int][]int{
		5:  {2, 3, 2, 3, 3},
		6:  {2, 3, 4, 3, 4},
		7:  {2, 3, 3, 4, 4},
		8:  {3, 4, 4, 5, 5},
		9:  {3, 4, 4, 5, 5},
		10: {3, 4, 4, 5, 5},
	}
	return table[players][round-1]
}

func avalonRequiredFails(players int, round int) int {
	if players >= 7 && round == 4 {
		return 2
	}
	return 1
}

func advanceAvalonLeader(room *Room) {
	players := playersBySeat(room)
	if len(players) == 0 {
		return
	}
	index := 0
	for currentIndex, player := range players {
		if player.ID == room.Avalon.LeaderID {
			index = currentIndex
			break
		}
	}
	room.Avalon.LeaderID = players[(index+1)%len(players)].ID
}
