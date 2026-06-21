package socialdeduction

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func werewolfActionsForLLM(room *Room, actions []aiplayer.LegalAction) ([]aiplayer.LegalAction, map[string]string) {
	return playerTargetActionsForLLM(room, actions, []string{"target:", "vote:", "shoot:", "save:", "poison:"})
}

func playerTargetActionsForLLM(room *Room, actions []aiplayer.LegalAction, prefixes []string) ([]aiplayer.LegalAction, map[string]string) {
	llmActions := make([]aiplayer.LegalAction, 0, len(actions))
	actionMap := map[string]string{}
	for _, action := range actions {
		llmAction := action
		for _, prefix := range prefixes {
			target := playerFromAction(room, action.ID, prefix)
			if target == nil {
				continue
			}
			targetRef := aiPlayerRef(room, target)
			llmAction.ID = prefix + targetRef
			llmAction.Label = strings.Replace(action.Label, target.Name, fmt.Sprintf("座位 %d", aiPlayerNumber(room, target)), 1)
			llmAction.Description = fmt.Sprintf("座位 %d 的存活玩家", aiPlayerNumber(room, target))
			break
		}
		llmActions = append(llmActions, llmAction)
		actionMap[llmAction.ID] = action.ID
	}
	return llmActions, actionMap
}

func avalonTeamActionsForLLM(room *Room, actions []aiplayer.LegalAction) ([]aiplayer.LegalAction, map[string]string) {
	llmActions := make([]aiplayer.LegalAction, 0, len(actions))
	actionMap := map[string]string{}
	for _, action := range actions {
		llmAction := action
		if team, ok := strings.CutPrefix(action.ID, "team:"); ok {
			refs := []string{}
			names := []string{}
			for _, id := range strings.Split(team, ",") {
				player := findPlayerByID(room, id)
				if player == nil {
					continue
				}
				refs = append(refs, aiPlayerRef(room, player))
				names = append(names, fmt.Sprintf("座位 %d", aiPlayerNumber(room, player)))
			}
			llmAction.ID = "team:" + strings.Join(refs, ",")
			llmAction.Label = "提名 " + strings.Join(names, "、")
			llmAction.Description = "选择这些座位组成任务队伍"
		}
		llmActions = append(llmActions, llmAction)
		actionMap[llmAction.ID] = action.ID
	}
	return llmActions, actionMap
}

func aiPlayerNumber(room *Room, target *Player) int {
	if target != nil && target.Seat >= 0 && hasUniqueSeat(room, target.Seat) {
		return target.Seat + 1
	}
	for index, player := range room.Players {
		if player.ID == target.ID {
			return index + 1
		}
	}
	return target.Seat + 1
}

func hasUniqueSeat(room *Room, seat int) bool {
	count := 0
	for _, player := range room.Players {
		if player.Seat == seat {
			count++
		}
	}
	return count == 1
}

func aiPlayerRef(room *Room, target *Player) string {
	return fmt.Sprintf("seat_%d", aiPlayerNumber(room, target))
}

func aliasWerewolfVoteIntents(room *Room, votes map[string]WerewolfVoteIntent) map[string]map[string]any {
	aliased := map[string]map[string]any{}
	for voterID, vote := range votes {
		voter := findPlayerByID(room, voterID)
		if voter == nil {
			continue
		}
		entry := map[string]any{"confirmed": vote.Confirmed}
		if target := findPlayerByID(room, vote.TargetID); target != nil {
			entry["targetId"] = aiPlayerRef(room, target)
		}
		aliased[aiPlayerRef(room, voter)] = entry
	}
	return aliased
}

func aliasUndercoverVoteIntents(room *Room, votes map[string]UndercoverVoteIntent) map[string]map[string]any {
	aliased := map[string]map[string]any{}
	for voterID, vote := range votes {
		voter := findPlayerByID(room, voterID)
		if voter == nil {
			continue
		}
		entry := map[string]any{"confirmed": vote.Confirmed}
		if target := findPlayerByID(room, vote.TargetID); target != nil {
			entry["targetId"] = aiPlayerRef(room, target)
		}
		aliased[aiPlayerRef(room, voter)] = entry
	}
	return aliased
}

func aliasAlignmentMap(room *Room, values map[string]Alignment) map[string]Alignment {
	aliased := map[string]Alignment{}
	for playerID, alignment := range values {
		if player := findPlayerByID(room, playerID); player != nil {
			aliased[aiPlayerRef(room, player)] = alignment
		}
	}
	return aliased
}

func aliasBoolMap(room *Room, values map[string]bool) map[string]bool {
	aliased := map[string]bool{}
	for playerID, value := range values {
		if player := findPlayerByID(room, playerID); player != nil {
			aliased[aiPlayerRef(room, player)] = value
		}
	}
	return aliased
}

func aliasStringMap(room *Room, values map[string]string) map[string]string {
	aliased := map[string]string{}
	for playerID, value := range values {
		player := findPlayerByID(room, playerID)
		if player == nil {
			continue
		}
		aliasedValue := value
		if target := findPlayerByID(room, value); target != nil {
			aliasedValue = aiPlayerRef(room, target)
		}
		aliased[aiPlayerRef(room, player)] = aliasedValue
	}
	return aliased
}

func aliasStringSlice(room *Room, values []string) []string {
	aliased := []string{}
	for _, value := range values {
		if player := findPlayerByID(room, value); player != nil {
			aliased = append(aliased, aiPlayerRef(room, player))
		}
	}
	return aliased
}

func aliasOptionalPlayerID(room *Room, playerID string) string {
	if playerID == "" {
		return ""
	}
	if player := findPlayerByID(room, playerID); player != nil {
		return aiPlayerRef(room, player)
	}
	return ""
}

func aliasPlayerNotes(room *Room, notes map[string]string) map[string]string {
	if len(notes) == 0 {
		return nil
	}
	aliased := map[string]string{}
	for playerID, note := range notes {
		if player := findPlayerByID(room, playerID); player != nil {
			aliased[aiPlayerRef(room, player)] = note
		}
	}
	return aliased
}

func playerFromAIRef(room *Room, ref string) *Player {
	ref = strings.TrimSpace(ref)
	seat, ok := strings.CutPrefix(ref, "seat_")
	if !ok {
		return nil
	}
	number, err := strconv.Atoi(seat)
	if err != nil || number < 1 {
		return nil
	}
	for _, player := range room.Players {
		if aiPlayerNumber(room, player) == number {
			return player
		}
	}
	return nil
}

func aiSpeeches(room *Room) []map[string]any {
	speeches := make([]map[string]any, 0, len(room.Speeches))
	for _, speech := range room.Speeches {
		playerRef := speech.PlayerID
		if player := findPlayerByID(room, speech.PlayerID); player != nil {
			playerRef = aiPlayerRef(room, player)
		}
		speeches = append(speeches, map[string]any{
			"id":         speech.ID,
			"playerId":   playerRef,
			"playerName": speech.PlayerName,
			"text":       speech.Text,
			"spokenAt":   speech.SpokenAt,
		})
	}
	return speeches
}

func aiSpeechForWerewolf(room *Room) []map[string]any {
	return aiSpeeches(room)
}
