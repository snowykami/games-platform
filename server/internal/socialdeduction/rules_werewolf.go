package socialdeduction

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func startWerewolf(room *Room) {
	if err := validateWerewolfCounts(room.Werewolf.RoleConfig.Counts, len(room.Players)); err != nil {
		applyDefaultWerewolfConfig(room)
	}
	roles := expandWerewolfRoles(room.Werewolf.RoleConfig.Counts)
	for index, player := range shuffledPlayers(room.Players) {
		player.Role = roles[index]
		player.Alignment = werewolfAlignment(player.Role)
	}
	room.Phase = PhaseWerewolfNight
	room.Werewolf.Day = 1
	room.Werewolf.RolePresets = nil
	room.Werewolf.NightActions = map[string]string{}
	room.Werewolf.SeerChecks = map[string]Alignment{}
	room.Werewolf.Votes = map[string]WerewolfVoteIntent{}
	room.Werewolf.DaySpeakers = map[string]bool{}
	room.Werewolf.RevealedIdiots = map[string]bool{}
	room.Werewolf.LastNight = ""
	room.Log = append(room.Log, createLog(fmt.Sprintf("狼人杀开始，角色组：%s。天黑请闭眼。", room.Werewolf.RoleConfig.Name)))
	recordAction(room, PublicAction{Type: "start", Message: "狼人杀开始，进入第一个夜晚。"})
}

func werewolfAlignment(role Role) Alignment {
	if role == RoleWerewolf {
		return AlignmentEvil
	}
	return AlignmentGood
}

func (m *Manager) advanceWerewolfNight(room *Room) {
	if !allRequiredNightActions(room) {
		return
	}

	killID := mostVotedTarget(room.Werewolf.NightActions, func(playerID string) bool {
		player := findPlayerByID(room, playerID)
		return player != nil && player.Role == RoleWerewolf
	})
	protectedID := ""
	for playerID, targetID := range room.Werewolf.NightActions {
		player := findPlayerByID(room, playerID)
		if player != nil && player.Role == RoleGuard {
			protectedID = targetID
		}
	}
	if room.Werewolf.WitchSaveTargetID != "" && room.Werewolf.WitchSaveTargetID == killID {
		protectedID = killID
	}

	deaths := []*Player{}
	if killID != "" && killID != protectedID {
		if target := findPlayerByID(room, killID); target != nil && target.Alive {
			deaths = append(deaths, target)
		}
	}
	if room.Werewolf.WitchPoisonID != "" {
		if target := findPlayerByID(room, room.Werewolf.WitchPoisonID); target != nil && target.Alive && !slices.Contains(deaths, target) {
			deaths = append(deaths, target)
		}
	}

	if len(deaths) == 0 {
		room.Werewolf.LastNight = "昨夜无人出局。"
		room.Log = append(room.Log, createLog(room.Werewolf.LastNight))
	} else {
		names := []string{}
		for _, target := range deaths {
			target.Alive = false
			names = append(names, target.Name)
		}
		room.Werewolf.LastNight = fmt.Sprintf("%s 在夜晚出局。", strings.Join(names, "、"))
		room.Log = append(room.Log, createLog(room.Werewolf.LastNight))
	}
	room.Werewolf.WitchSaveTargetID = ""
	room.Werewolf.WitchPoisonID = ""

	if hunter := firstDeadHunter(deaths); hunter != nil {
		room.Phase = PhaseWerewolfHunter
		room.Werewolf.HunterPendingID = hunter.ID
		room.Werewolf.HunterAfterPhase = PhaseWerewolfDay
		recordAction(room, PublicAction{Type: "hunter_pending", ActorID: hunter.ID, ActorName: hunter.Name, Message: fmt.Sprintf("%s 可以发动猎人技能。", hunter.Name)})
		return
	}

	if checkWerewolfWin(room) {
		return
	}
	startWerewolfDay(room)
}

func allRequiredNightActions(room *Room) bool {
	werewolfActed := false
	werewolfAlive := false
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		switch player.Role {
		case RoleWerewolf:
			werewolfAlive = true
			if _, ok := room.Werewolf.NightActions[player.ID]; ok {
				werewolfActed = true
			}
		case RoleSeer, RoleGuard:
			if _, ok := room.Werewolf.NightActions[player.ID]; !ok {
				return false
			}
		case RoleWitch:
			if witchCanAct(room) {
				if _, ok := room.Werewolf.NightActions[player.ID]; !ok {
					return false
				}
			}
		}
	}
	return !werewolfAlive || werewolfActed
}

func canActAtNight(player *Player) bool {
	return player.Role == RoleWerewolf || player.Role == RoleSeer || player.Role == RoleGuard || player.Role == RoleWitch
}

func witchCanAct(room *Room) bool {
	return !room.Werewolf.WitchAntidoteUsed || !room.Werewolf.WitchPoisonUsed
}

func applyWerewolfNightAction(room *Room, player *Player, actionID string) (*Player, error) {
	actions := werewolfNightActions(room, player)
	if !aiplayer.ValidateAction(actionID, actions) {
		return nil, errors.New("invalid_target")
	}
	switch {
	case strings.HasPrefix(actionID, "skip:"):
		room.Werewolf.NightActions[player.ID] = actionID
		return nil, nil
	case strings.HasPrefix(actionID, "save:"):
		target := playerFromAction(room, actionID, "save:")
		if target == nil || !target.Alive || room.Werewolf.WitchAntidoteUsed {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.WitchAntidoteUsed = true
		room.Werewolf.WitchSaveTargetID = target.ID
		room.Werewolf.NightActions[player.ID] = actionID
		return target, nil
	case strings.HasPrefix(actionID, "poison:"):
		target := playerFromAction(room, actionID, "poison:")
		if target == nil || !target.Alive || room.Werewolf.WitchPoisonUsed {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.WitchPoisonUsed = true
		room.Werewolf.WitchPoisonID = target.ID
		room.Werewolf.NightActions[player.ID] = actionID
		return target, nil
	default:
		target := playerFromAction(room, actionID, "target:")
		if target == nil || !target.Alive {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.NightActions[player.ID] = target.ID
		if player.Role == RoleSeer {
			room.Werewolf.SeerChecks[target.ID] = target.Alignment
		}
		return target, nil
	}
}

func resolveHunterShot(room *Room, targetID string) error {
	hunter := findPlayerByID(room, room.Werewolf.HunterPendingID)
	if hunter == nil || hunter.Role != RoleHunter {
		return errors.New("hunter_not_found")
	}
	targetID = strings.TrimSpace(targetID)
	if targetID != "" {
		target := findPlayerByID(room, targetID)
		if target == nil || !target.Alive || target.ID == hunter.ID {
			return errors.New("invalid_target")
		}
		target.Alive = false
		message := fmt.Sprintf("%s 发动猎人技能带走了 %s。", hunter.Name, target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "hunter_shot", ActorID: hunter.ID, ActorName: hunter.Name, TargetID: target.ID, Message: message})
	} else {
		message := fmt.Sprintf("%s 放弃发动猎人技能。", hunter.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "hunter_skip", ActorID: hunter.ID, ActorName: hunter.Name, Message: message})
	}
	afterPhase := room.Werewolf.HunterAfterPhase
	room.Werewolf.HunterPendingID = ""
	room.Werewolf.HunterAfterPhase = ""
	if checkWerewolfWin(room) {
		return nil
	}
	if afterPhase == PhaseWerewolfDay {
		startWerewolfDay(room)
		return nil
	}
	startNextWerewolfNight(room)
	return nil
}

func firstDeadHunter(players []*Player) *Player {
	for _, player := range players {
		if player.Role == RoleHunter {
			return player
		}
	}
	return nil
}

func werewolfVoterCount(room *Room) int {
	count := 0
	for _, player := range room.Players {
		if player.Alive && !room.Werewolf.RevealedIdiots[player.ID] {
			count++
		}
	}
	return count
}

func startWerewolfDay(room *Room) {
	room.Phase = PhaseWerewolfDay
	room.Werewolf.NightActions = map[string]string{}
	room.Werewolf.Votes = map[string]WerewolfVoteIntent{}
	room.Werewolf.DaySpeakers = map[string]bool{}
	recordAction(room, PublicAction{Type: "day_started", Message: room.Werewolf.LastNight})
}

func markWerewolfDaySpeech(room *Room, player *Player) {
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay || player == nil || !player.Alive || room.Werewolf.RevealedIdiots[player.ID] {
		return
	}
	if room.Werewolf.DaySpeakers == nil {
		room.Werewolf.DaySpeakers = map[string]bool{}
	}
	room.Werewolf.DaySpeakers[player.ID] = true
	if allWerewolfDaySpeakersReady(room) {
		advanceWerewolfDayToVote(room)
	}
}

func allWerewolfDaySpeakersReady(room *Room) bool {
	if werewolfVoterCount(room) == 0 {
		return false
	}
	for _, player := range room.Players {
		if player.Alive && !room.Werewolf.RevealedIdiots[player.ID] && !room.Werewolf.DaySpeakers[player.ID] {
			return false
		}
	}
	return true
}

func nextPendingWerewolfDayAISpeaker(room *Room, lastSpeakerID string) *Player {
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay {
		return nil
	}
	start := 0
	for i, player := range room.Players {
		if player.ID == lastSpeakerID {
			start = i + 1
			break
		}
	}
	for offset := range room.Players {
		player := room.Players[(start+offset)%len(room.Players)]
		if player.IsAI && player.Alive && player.ID != lastSpeakerID && player.AI != nil && !room.Werewolf.RevealedIdiots[player.ID] && !room.Werewolf.DaySpeakers[player.ID] {
			return player
		}
	}
	return nil
}

func advanceWerewolfDayToVote(room *Room) {
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay {
		return
	}
	room.Phase = PhaseWerewolfVote
	room.Werewolf.Votes = map[string]WerewolfVoteIntent{}
	room.Log = append(room.Log, createLog("白天发言结束，自动进入放逐投票。"))
	recordAction(room, PublicAction{Type: "vote_started", Message: "所有可投票玩家已发言，开始放逐投票。"})
}

func (m *Manager) resolveWerewolfVote(room *Room) {
	confirmedVotes := confirmedWerewolfVotes(room)
	if len(confirmedVotes) < werewolfVoterCount(room) {
		return
	}
	targetID := mostVotedTarget(confirmedVotes, func(string) bool { return true })
	if target := findPlayerByID(room, targetID); target != nil {
		if target.Role == RoleIdiot && !room.Werewolf.RevealedIdiots[target.ID] {
			if room.Werewolf.RevealedIdiots == nil {
				room.Werewolf.RevealedIdiots = map[string]bool{}
			}
			room.Werewolf.RevealedIdiots[target.ID] = true
			message := fmt.Sprintf("%s 是白痴，翻牌免疫本次放逐。", target.Name)
			room.Log = append(room.Log, createLog(message))
			recordAction(room, PublicAction{Type: "idiot_revealed", TargetID: target.ID, Message: message})
			startNextWerewolfNight(room)
			return
		}
		target.Alive = false
		message := fmt.Sprintf("%s 被放逐出局。", target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "exile", TargetID: target.ID, Message: message})
		if target.Role == RoleHunter {
			room.Phase = PhaseWerewolfHunter
			room.Werewolf.HunterPendingID = target.ID
			room.Werewolf.HunterAfterPhase = PhaseWerewolfNight
			recordAction(room, PublicAction{Type: "hunter_pending", ActorID: target.ID, ActorName: target.Name, Message: fmt.Sprintf("%s 可以发动猎人技能。", target.Name)})
			return
		}
	}
	if checkWerewolfWin(room) {
		return
	}
	startNextWerewolfNight(room)
}

func confirmedWerewolfVotes(room *Room) map[string]string {
	votes := map[string]string{}
	for actorID, vote := range room.Werewolf.Votes {
		if vote.Confirmed && vote.TargetID != "" {
			votes[actorID] = vote.TargetID
		}
	}
	return votes
}

func startNextWerewolfNight(room *Room) {
	room.Werewolf.Day++
	room.Werewolf.Votes = map[string]WerewolfVoteIntent{}
	room.Werewolf.DaySpeakers = map[string]bool{}
	room.Werewolf.NightActions = map[string]string{}
	room.Phase = PhaseWerewolfNight
	room.Log = append(room.Log, createLog("夜幕再次降临。"))
	recordAction(room, PublicAction{Type: "night_started", Message: "夜幕再次降临。"})
}

func checkWerewolfWin(room *Room) bool {
	wolves := 0
	good := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		if player.Alignment == AlignmentEvil {
			wolves++
		} else {
			good++
		}
	}
	if wolves == 0 {
		finish(room, AlignmentGood, "所有狼人出局，好人阵营获胜。")
		return true
	}
	if wolves >= good {
		finish(room, AlignmentEvil, "狼人数量已压制好人，狼人阵营获胜。")
		return true
	}
	return false
}
