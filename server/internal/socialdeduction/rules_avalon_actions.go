package socialdeduction

import (
	"slices"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func avalonTeamActions(room *Room, leader *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	players := append([]*Player{}, room.Players...)
	for _, first := range players {
		for _, second := range players {
			if room.Avalon.RequiredTeam == 2 {
				team := uniqueTeam([]string{first.ID, second.ID})
				if len(team) == 2 && slices.Contains(team, leader.ID) {
					actions = append(actions, avalonTeamAction(room, team))
				}
				continue
			}
			for _, third := range players {
				team := uniqueTeam([]string{leader.ID, first.ID, second.ID, third.ID})
				if len(team) == room.Avalon.RequiredTeam {
					actions = append(actions, avalonTeamAction(room, team))
				}
				if len(actions) >= 24 {
					return actions
				}
			}
		}
	}
	if len(actions) == 0 {
		actions = append(actions, avalonTeamAction(room, []string{leader.ID}))
	}
	return actions
}

func avalonTeamAction(room *Room, team []string) aiplayer.LegalAction {
	names := []string{}
	for _, id := range team {
		if player := findPlayerByID(room, id); player != nil {
			names = append(names, player.Name)
		}
	}
	return aiplayer.LegalAction{ID: "team:" + strings.Join(team, ","), Label: "提名 " + strings.Join(names, "、")}
}

func uniqueTeam(ids []string) []string {
	seen := map[string]bool{}
	team := []string{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		team = append(team, id)
	}
	return team
}

func goodPlayers(room *Room) []*Player {
	players := []*Player{}
	for _, player := range room.Players {
		if player.Alignment == AlignmentGood {
			players = append(players, player)
		}
	}
	return players
}

func livingCount(room *Room) int {
	count := 0
	for _, player := range room.Players {
		if player.Alive {
			count++
		}
	}
	return count
}

func playerFromAction(room *Room, actionID string, prefix string) *Player {
	if !strings.HasPrefix(actionID, prefix) {
		return nil
	}
	return findPlayerByID(room, strings.TrimPrefix(actionID, prefix))
}
