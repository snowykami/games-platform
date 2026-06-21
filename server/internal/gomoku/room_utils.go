package gomoku

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func publicRoom(room *Room, viewerID string) PublicRoom {
	players := make([]Player, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, *player)
	}

	currentPlayerID := ""
	if room.Phase == PhasePlaying && len(room.Players) > 0 {
		currentPlayerID = room.Players[room.CurrentPlayerIndex].ID
	}

	logs := room.Log
	if len(logs) > 10 {
		logs = logs[len(logs)-10:]
	}

	return PublicRoom{
		ID:              room.ID,
		HostUserID:      room.HostUserID,
		HostPlayerID:    playerIDForUser(room, room.HostUserID),
		YouPlayerID:     playerIDForUser(room, viewerID),
		Phase:           room.Phase,
		Players:         players,
		BoardSize:       BoardSize,
		Moves:           append([]Move{}, room.Moves...),
		CurrentPlayerID: currentPlayerID,
		WinnerID:        room.WinnerID,
		WinningLine:     append([]Point{}, room.WinningLine...),
		IsDraw:          room.IsDraw,
		Log:             append([]LogEntry{}, logs...),
		Speeches:        append([]SpeechEntry{}, room.Speeches...),
		ActionSeq:       room.ActionSeq,
		RecentActions:   append([]PublicAction{}, room.RecentActions...),
	}
}

func playerIDForUser(room *Room, userID string) string {
	if player := findPlayerByUserID(room, userID); player != nil {
		return player.ID
	}
	return ""
}

func createHumanPlayer(user UserView, role string) *Player {
	return &Player{
		ID:        "plr_" + randomToken(8),
		UserID:    user.ID,
		Name:      user.DisplayName,
		Role:      role,
		Kind:      user.Kind,
		Connected: true,
		JoinedAt:  time.Now().UTC(),
	}
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

func recordAction(room *Room, action PublicAction) {
	room.ActionSeq++
	action.Seq = room.ActionSeq
	room.RecentActions = append(room.RecentActions, action)
	if len(room.RecentActions) > 6 {
		room.RecentActions = room.RecentActions[len(room.RecentActions)-6:]
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
	if len(room.Speeches) > 18 {
		room.Speeches = room.Speeches[len(room.Speeches)-18:]
	}
	return true
}

func recentSpeeches(room *Room) []SpeechEntry {
	speeches := room.Speeches
	if len(speeches) > 8 {
		speeches = speeches[len(speeches)-8:]
	}
	return append([]SpeechEntry{}, speeches...)
}

func nextAISpeechPlayer(room *Room, lastSpeakerID string) *Player {
	for _, player := range room.Players {
		if player.IsAI && player.ID != lastSpeakerID && player.AI != nil {
			return player
		}
	}
	return nil
}

func speechActions() []aiplayer.LegalAction {
	return []aiplayer.LegalAction{
		{ID: "speak", Label: "说一句话", Description: "用自然、简短的玩家语气回应桌面发言。"},
		{ID: "skip", Label: "不发言", Description: "没有必要回应时选择。"},
	}
}

func createLog(text string) LogEntry {
	return LogEntry{ID: "log_" + randomToken(8), Text: text}
}

func createRoomID() string {
	return "GMK" + randomToken(5)
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range length {
		builder.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return builder.String()
}
