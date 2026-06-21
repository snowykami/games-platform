package socialdeduction

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/roommeta"
)

func mostVotedTarget(votes map[string]string, actorFilter func(string) bool) string {
	counts := map[string]int{}
	bestID := ""
	bestCount := 0
	for actorID, targetID := range votes {
		if !actorFilter(actorID) {
			continue
		}
		counts[targetID]++
		if counts[targetID] > bestCount {
			bestID = targetID
			bestCount = counts[targetID]
		}
	}
	return bestID
}

func shuffledPlayers(players []*Player) []*Player {
	next := append([]*Player{}, players...)
	rand.Shuffle(len(next), func(i, j int) { next[i], next[j] = next[j], next[i] })
	return next
}

func createHumanPlayer(user UserView, role string, seat int) *Player {
	return &Player{
		ID:        "plr_" + randomToken(8),
		UserID:    user.ID,
		Name:      user.DisplayName,
		Seat:      seat,
		RoomRole:  role,
		Kind:      user.Kind,
		Connected: true,
		Alive:     true,
		JoinedAt:  time.Now().UTC(),
	}
}

func normalizePlayerDisplayName(displayName string) (string, error) {
	return roommeta.NormalizeDisplayName(displayName)
}

func findPlayerByUserID(room *Room, userID string) *Player {
	for _, player := range room.Players {
		if player.UserID == userID {
			return player
		}
	}
	return nil
}

func findPlayerByID(room *Room, playerID string) *Player {
	for _, player := range room.Players {
		if player.ID == playerID {
			return player
		}
	}
	return nil
}

func nextAIProfile(room *Room, level aiplayer.Level) AIProfile {
	profile := aiplayer.NextProfile(usedAINames(room))
	return AIProfile{Name: profile.Name, Personality: profile.Personality, SpeechStyle: profile.SpeechStyle, Level: string(level)}
}

func usedAINames(room *Room) map[string]bool {
	used := map[string]bool{}
	for _, player := range room.Players {
		used[player.Name] = true
	}
	return used
}

func recordAction(room *Room, action PublicAction) {
	room.ActionSeq++
	action.Seq = room.ActionSeq
	room.RecentActions = append(room.RecentActions, action)
	if len(room.RecentActions) > 8 {
		room.RecentActions = room.RecentActions[len(room.RecentActions)-8:]
	}
}

func recordSpeech(room *Room, player *Player, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	runes := []rune(text)
	if len(runes) > 120 {
		text = string(runes[:120])
	}
	room.Speeches = append(room.Speeches, SpeechEntry{
		ID:         "speech_" + randomToken(8),
		PlayerID:   player.ID,
		PlayerName: player.Name,
		Text:       text,
		SpokenAt:   time.Now().UTC(),
	})
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 说：%s", player.Name, text)))
	if len(room.Speeches) > 18 {
		room.Speeches = room.Speeches[len(room.Speeches)-18:]
	}
	return true
}

func nextAISpeechPlayer(room *Room, lastSpeakerID string) *Player {
	if room == nil || len(room.Players) == 0 {
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
		if player.IsAI && player.Alive && player.ID != lastSpeakerID && player.AI != nil {
			return player
		}
	}
	return nil
}

func speechActions() []aiplayer.LegalAction {
	return []aiplayer.LegalAction{
		{ID: "speak", Label: "说一句话", Description: "互动阶段优先用自然、简短的玩家语气回应最近发言、投票、怀疑、组队或描述。"},
		{ID: "skip", Label: "不发言", Description: "仅在夜晚、秘密行动、没有安全话可说，或继续发言会泄露隐藏信息时选择。"},
	}
}

func playerNote(room *Room, viewer *Player, targetID string) string {
	if viewer == nil || room.PlayerNotes == nil {
		return ""
	}
	return room.PlayerNotes[viewer.ID][targetID]
}

func setPlayerNote(room *Room, viewerID string, targetID string, note string) {
	note = roommeta.NormalizeNote(note)
	if room.PlayerNotes == nil {
		room.PlayerNotes = map[string]map[string]string{}
	}
	if room.PlayerNotes[viewerID] == nil {
		room.PlayerNotes[viewerID] = map[string]string{}
	}
	if note == "" {
		delete(room.PlayerNotes[viewerID], targetID)
		return
	}
	room.PlayerNotes[viewerID][targetID] = note
}

func reconcileLobbyConfig(room *Room) {
	if room.Game == GameWerewolf {
		reconcileWerewolfConfig(room)
	}
	if room.Game == GameUndercover {
		applyDefaultUndercoverConfig(room)
	}
}

func createLog(text string) LogEntry {
	return LogEntry{ID: "log_" + randomToken(8), Text: text}
}

func createRoomID(game GameKind) string {
	prefix := "AVL"
	if game == GameWerewolf {
		prefix = "WWF"
	}
	if game == GameUndercover {
		prefix = "UND"
	}
	return prefix + randomToken(5)
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range length {
		builder.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return builder.String()
}

func hasDuplicate(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func cloneStringMap(source map[string]string) map[string]string {
	next := map[string]string{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneWerewolfVotes(source map[string]WerewolfVoteIntent) map[string]WerewolfVoteIntent {
	next := map[string]WerewolfVoteIntent{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneUndercoverVotes(source map[string]UndercoverVoteIntent) map[string]UndercoverVoteIntent {
	next := map[string]UndercoverVoteIntent{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	next := map[string]bool{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneAlignmentMap(source map[string]Alignment) map[string]Alignment {
	next := map[string]Alignment{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func seerChecksForViewer(room *Room, viewer *Player) map[string]Alignment {
	if viewer == nil || viewer.Role != RoleSeer {
		return nil
	}
	next := map[string]Alignment{}
	for targetID, alignment := range room.Werewolf.SeerChecks {
		next[targetID] = alignment
	}
	return next
}
