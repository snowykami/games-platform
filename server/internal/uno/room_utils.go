package uno

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func currentHuman(room *Room, actorID string) (*Player, error) {
	player, err := playingPlayer(room, actorID)
	if err != nil {
		return nil, err
	}
	if player.UserID != actorID || player.IsAI || room.Players[room.CurrentPlayerIndex].ID != player.ID {
		return nil, errors.New("not_current_turn")
	}
	return player, nil
}

func playingPlayer(room *Room, actorID string) (*Player, error) {
	if room.Phase != PhasePlaying {
		return nil, errors.New("game_not_playing")
	}
	if len(room.Players) == 0 || room.CurrentPlayerIndex >= len(room.Players) {
		return nil, errors.New("invalid_turn")
	}

	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return nil, errors.New("not_current_turn")
	}
	return player, nil
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

func publicRoom(room *Room, viewerID string) PublicRoom {
	players := make([]Player, 0, len(room.Players))
	for _, player := range room.Players {
		copy := *player
		copy.HandCount = len(player.Hand)
		if player.UserID != viewerID {
			copy.Hand = nil
		}
		players = append(players, copy)
	}

	var topCard *Card
	if len(room.DiscardPile) > 0 {
		card := room.DiscardPile[len(room.DiscardPile)-1]
		topCard = &card
	}

	currentPlayerID := ""
	playableCardIDs := []string{}
	if room.Phase == PhasePlaying && len(room.Players) > 0 {
		currentPlayer := room.Players[room.CurrentPlayerIndex]
		currentPlayerID = currentPlayer.ID
		if currentPlayer.UserID == viewerID && !currentPlayer.IsAI {
			playableCardIDs = playableCardsForPlayer(currentPlayer, room)
		}
	}

	logs := room.Log
	if len(logs) > 10 {
		logs = logs[len(logs)-10:]
	}

	return PublicRoom{
		ID:                   room.ID,
		HostUserID:           room.HostUserID,
		HostPlayerID:         playerIDForUser(room, room.HostUserID),
		YouPlayerID:          playerIDForUser(room, viewerID),
		VariantKey:           room.VariantKey,
		ThemeKey:             room.ThemeKey,
		Phase:                room.Phase,
		Players:              players,
		TopCard:              topCard,
		DrawPileCount:        len(room.DrawPile),
		CurrentPlayerID:      currentPlayerID,
		Direction:            room.Direction,
		ActiveColor:          room.ActiveColor,
		PendingDrawCount:     room.PendingDrawCount,
		FlipSide:             room.FlipSide,
		Rules:                room.Rules,
		PlayableCardIDs:      playableCardIDs,
		WinnerID:             room.WinnerID,
		Log:                  append([]LogEntry{}, logs...),
		Speeches:             append([]SpeechEntry{}, room.Speeches...),
		ActionSeq:            room.ActionSeq,
		RecentActions:        append([]PublicAction{}, room.RecentActions...),
		TurnDeadline:         cloneTimePtr(room.TurnDeadline),
		TurnRemainingSeconds: turnRemainingSeconds(room.TurnDeadline),
	}
}

func playerIDForUser(room *Room, userID string) string {
	if player := findPlayerByUserID(room, userID); player != nil {
		return player.ID
	}
	return ""
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
	return room.CurrentPlayerIndex
}

func createRoomID() string {
	return strings.ToUpper(randomToken(6))
}

func normalizeVariantKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	switch key {
	case "", "classic":
		return "classic"
	case "party", "stacking", "all-wild", "flip", "no-mercy", "wild-plus":
		return key
	default:
		return "classic"
	}
}

func rulesForVariant(key string) RuleSet {
	switch key {
	case "stacking":
		return RuleSet{Stacking: true}
	case "party":
		return RuleSet{Stacking: true, SevenZero: true, JumpIn: true}
	case "all-wild":
		return RuleSet{AllWild: true, Stacking: true}
	case "flip":
		return RuleSet{Flip: true}
	case "no-mercy":
		return RuleSet{NoMercy: true, Stacking: true, SevenZero: true, JumpIn: true}
	case "wild-plus":
		return RuleSet{Stacking: true, SevenZero: true, JumpIn: true, Flip: true, NoMercy: true}
	default:
		return RuleSet{}
	}
}

func normalizeThemeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	switch key {
	case "", "classic":
		return "classic"
	case "neon", "anime-collab":
		return key
	default:
		return "classic"
	}
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	token := make([]byte, length)
	for index := range token {
		token[index] = alphabet[rand.IntN(len(alphabet))]
	}
	return string(token)
}

func createLog(text string) LogEntry {
	return LogEntry{ID: "log_" + randomToken(8), Text: text}
}

func recordPlayAction(room *Room, player *Player, card Card, message string) {
	recordAction(room, PublicAction{
		Type:      ActionPlay,
		ActorID:   player.ID,
		ActorName: player.Name,
		TargetID:  player.ID,
		Card:      &card,
		Message:   message,
	})
}

func recordDrawAction(room *Room, actor *Player, target *Player, count int, message string) {
	if count <= 0 {
		return
	}

	recordAction(room, PublicAction{
		Type:      ActionDraw,
		ActorID:   actor.ID,
		ActorName: actor.Name,
		TargetID:  target.ID,
		Count:     count,
		Message:   message,
	})
}

func recordEffectAction(room *Room, actor *Player, target *Player, message string) {
	recordAction(room, PublicAction{
		Type:      ActionEffect,
		ActorID:   actor.ID,
		ActorName: actor.Name,
		TargetID:  target.ID,
		Message:   message,
	})
}

func recordAction(room *Room, action PublicAction) {
	room.ActionSeq++
	action.Seq = room.ActionSeq
	room.RecentActions = append(room.RecentActions, action)
	if len(room.RecentActions) > 12 {
		room.RecentActions = room.RecentActions[len(room.RecentActions)-12:]
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

func formatCard(card Card) string {
	if card.Kind == KindNumber && card.Value != nil {
		return fmt.Sprintf("%s %d", card.Color, *card.Value)
	}
	return fmt.Sprintf("%s %s", card.Color, card.Kind)
}
