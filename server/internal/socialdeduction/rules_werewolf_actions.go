package socialdeduction

import (
	"fmt"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func werewolfNightActions(room *Room, actor *Player) []aiplayer.LegalAction {
	if !canActAtNight(actor) {
		return nil
	}
	if actor.Role == RoleWitch {
		return witchNightActions(room, actor)
	}
	labelPrefix := "选择"
	switch actor.Role {
	case RoleWerewolf:
		labelPrefix = "击杀"
	case RoleSeer:
		labelPrefix = "查验"
	case RoleGuard:
		labelPrefix = "守护"
	}
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if !target.Alive {
			continue
		}
		if actor.Role == RoleSeer && target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "target:" + target.ID,
			Label:       fmt.Sprintf("%s %s", labelPrefix, target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}

func witchNightActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	killID := currentWerewolfKillTarget(room)
	if killID != "" && !room.Werewolf.WitchAntidoteUsed {
		if target := findPlayerByID(room, killID); target != nil && target.Alive {
			actions = append(actions, aiplayer.LegalAction{
				ID:          "save:" + target.ID,
				Label:       fmt.Sprintf("使用解药救 %s", target.Name),
				Description: "消耗一次解药，阻止今晚狼刀出局。",
			})
		}
	}
	if !room.Werewolf.WitchPoisonUsed {
		for _, target := range room.Players {
			if !target.Alive || target.ID == actor.ID {
				continue
			}
			actions = append(actions, aiplayer.LegalAction{
				ID:          "poison:" + target.ID,
				Label:       fmt.Sprintf("使用毒药毒 %s", target.Name),
				Description: fmt.Sprintf("消耗一次毒药，令座位 %d 出局。", target.Seat+1),
			})
		}
	}
	actions = append(actions, aiplayer.LegalAction{ID: "skip:witch", Label: "今晚不用药"})
	return actions
}

func hunterShotActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{{ID: "shoot:skip", Label: "不开枪"}}
	for _, target := range room.Players {
		if !target.Alive || target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "shoot:" + target.ID,
			Label:       fmt.Sprintf("开枪带走 %s", target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}

func currentWerewolfKillTarget(room *Room) string {
	return mostVotedTarget(room.Werewolf.NightActions, func(playerID string) bool {
		player := findPlayerByID(room, playerID)
		return player != nil && player.Role == RoleWerewolf
	})
}

func werewolfVoteActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if !target.Alive || target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "vote:" + target.ID,
			Label:       fmt.Sprintf("投票给 %s", target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}
