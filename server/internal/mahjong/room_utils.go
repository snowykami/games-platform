package mahjong

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func publicRoom(room *Room, viewerID string) PublicRoom {
	players := make([]PublicPlayer, 0, len(room.Players))
	for _, player := range room.Players {
		hand := []Tile{}
		if player.UserID == viewerID {
			hand = append([]Tile{}, player.Hand...)
		}
		players = append(players, PublicPlayer{
			ID:             player.ID,
			UserID:         player.UserID,
			Name:           player.Name,
			Role:           player.Role,
			Kind:           player.Kind,
			IsAI:           player.IsAI,
			Connected:      player.Connected,
			DisconnectedAt: player.DisconnectedAt,
			AI:             player.AI,
			Wind:           player.Wind,
			Hand:           hand,
			HandCount:      len(player.Hand),
			Melds:          append([]Meld{}, player.Melds...),
			Discards:       append([]Tile{}, player.Discards...),
		})
	}
	currentPlayerID := ""
	if (room.Phase == PhasePlaying || room.Phase == PhaseClaiming) && len(room.Players) > 0 {
		currentPlayerID = room.Players[room.CurrentPlayerIndex].ID
	}
	dealerID := ""
	if len(room.Players) > 0 {
		dealerID = room.Players[room.DealerIndex].ID
	}
	claims := []ClaimOption{}
	for _, option := range room.ClaimOptions {
		player := findPlayerByID(room, option.PlayerID)
		if player != nil && player.UserID == viewerID {
			claims = append(claims, option)
		}
	}
	logs := room.Log
	if len(logs) > 12 {
		logs = logs[len(logs)-12:]
	}

	return PublicRoom{
		ID:              room.ID,
		HostUserID:      room.HostUserID,
		HostPlayerID:    playerIDForUser(room, room.HostUserID),
		YouPlayerID:     playerIDForUser(room, viewerID),
		Phase:           room.Phase,
		Players:         players,
		WallCount:       len(room.Wall),
		DeadWallCount:   len(room.DeadWall),
		CurrentPlayerID: currentPlayerID,
		DealerID:        dealerID,
		RoundWind:       room.RoundWind,
		HasDrawn:        room.HasDrawn,
		LastDiscard:     room.LastDiscard,
		ClaimOptions:    claims,
		RuleSet:         room.RuleSet,
		WinnerID:        room.WinnerID,
		WinResult:       room.WinResult,
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
		ID:        "player_" + randomToken(10),
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

func playerIndex(room *Room, playerID string) int {
	for index, player := range room.Players {
		if player.ID == playerID {
			return index
		}
	}
	return -1
}

func nextPlayerIndex(room *Room, playerID string) int {
	index := playerIndex(room, playerID)
	if index < 0 {
		return 0
	}
	return (index + 1) % len(room.Players)
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
	if len([]rune(text)) > 120 {
		text = string([]rune(text)[:120])
	}
	room.Speeches = append(room.Speeches, SpeechEntry{ID: "speech_" + randomToken(10), PlayerID: player.ID, PlayerName: player.Name, Text: text, SpokenAt: time.Now().UTC()})
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
	return LogEntry{ID: "log_" + randomToken(10), Text: text}
}

func createRoomID() string {
	return randomToken(5)
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range length {
		builder.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return builder.String()
}
