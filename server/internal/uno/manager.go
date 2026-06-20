package uno

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/roommeta"
)

const (
	minPlayers     = 2
	maxPlayers     = 10
	turnTimeout    = 30 * time.Second
	offlineRoomTTL = 60 * time.Second
)

var colors = []Color{ColorRed, ColorYellow, ColorGreen, ColorBlue}

type Manager struct {
	aiProvider aiplayer.Provider
	mu         sync.RWMutex
	rooms      map[string]*Room
}

type TickResult struct {
	BroadcastRoomIDs  []string
	ScheduleAIRoomIDs []string
	DestroyedRoomIDs  []string
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{aiProvider: aiProvider, rooms: map[string]*Room{}}
}

func (m *Manager) CreateRoom(user UserView, options RoomOptions) PublicRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	variantKey := normalizeVariantKey(options.VariantKey)
	room := &Room{
		ID:         createRoomID(),
		HostUserID: user.ID,
		VariantKey: variantKey,
		ThemeKey:   normalizeThemeKey(options.ThemeKey),
		Rules:      rulesForVariant(variantKey),
		Phase:      PhaseLobby,
		Direction:  1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	room.Players = append(room.Players, createHumanPlayer(user, "host"))
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 创建了房间。", user.DisplayName)))
	m.rooms[room.ID] = room

	return publicRoom(room, user.ID)
}

func (m *Manager) JoinRoom(roomID string, user UserView) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()

	player := findPlayerByUserID(room, user.ID)
	if player == nil {
		if room.Phase != PhaseLobby {
			return PublicRoom{}, errors.New("game_already_started")
		}
		if len(room.Players) >= maxPlayers {
			return PublicRoom{}, errors.New("room_full")
		}

		player = createHumanPlayer(user, "player")
		room.Players = append(room.Players, player)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 加入了房间。", user.DisplayName)))
	}

	player.Connected = true
	player.DisconnectedAt = nil
	room.AllHumansOfflineSince = nil
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, user.ID), nil
}

func (m *Manager) Leave(roomID string, userID string) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return
	}
	defer room.mu.Unlock()

	now := time.Now().UTC()
	if player := findPlayerByUserID(room, userID); player != nil {
		player.Connected = false
		if !player.IsAI {
			player.DisconnectedAt = &now
		}
	}
	updateAllHumansOfflineSince(room, now)
}

func (m *Manager) AddAI(roomID string, actorID string, options AIOptions) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_add_ai")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("ai_only_lobby")
	}
	if len(room.Players) >= maxPlayers {
		return PublicRoom{}, errors.New("room_full")
	}

	if strings.TrimSpace(options.Level) == string(aiplayer.LevelLLM) && (m.aiProvider == nil || !m.aiProvider.Enabled()) {
		return PublicRoom{}, errors.New("llm_not_configured")
	}
	level := aiplayer.NormalizeLevel(options.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	profile := nextAIProfile(room, level)
	room.Players = append(room.Players, &Player{
		ID:        "ai_" + randomToken(8),
		UserID:    "ai_" + randomToken(8),
		Name:      profile.Name,
		Role:      "player",
		Kind:      "ai",
		IsAI:      true,
		Connected: true,
		AI:        &profile,
		JoinedAt:  time.Now().UTC(),
	})
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 加入了房间。", profile.Name)))
	room.UpdatedAt = time.Now().UTC()

	return publicRoom(room, actorID), nil
}

func (m *Manager) UpdateAI(roomID string, actorID string, playerID string, options AIOptions) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_add_ai")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("ai_only_lobby")
	}
	if strings.TrimSpace(options.Level) == string(aiplayer.LevelLLM) && (m.aiProvider == nil || !m.aiProvider.Enabled()) {
		return PublicRoom{}, errors.New("llm_not_configured")
	}

	player := findPlayerByID(room, playerID)
	if player == nil || !player.IsAI || player.AI == nil {
		return PublicRoom{}, errors.New("ai_player_not_found")
	}
	level := aiplayer.NormalizeLevel(options.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	player.AI.Level = string(level)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) RemovePlayer(roomID string, actorID string, playerID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_remove_player")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("remove_player_only_lobby")
	}
	index := slices.IndexFunc(room.Players, func(player *Player) bool {
		return player.ID == playerID
	})
	if index < 0 {
		return PublicRoom{}, errors.New("player_not_found")
	}
	player := room.Players[index]
	if player.UserID == room.HostUserID || player.Role == "host" {
		return PublicRoom{}, errors.New("cannot_remove_host")
	}
	room.Players = slices.Delete(room.Players, index, index+1)
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 被房主移出了房间。", player.Name)))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Say(roomID string, actorID string, text string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	player := findPlayerByUserID(room, actorID)
	if player == nil {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if !recordSpeech(room, player, text) {
		return PublicRoom{}, errors.New("invalid_speech")
	}
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) RunAISpeech(roomID string) (PublicRoom, bool, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return PublicRoom{}, false, nil
	}
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || len(room.Speeches) == 0 {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	if hasPendingAIRequiredAction(room) {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	lastSpeech := room.Speeches[len(room.Speeches)-1]
	if lastSpeech.ID == room.LastAISpeechSourceID {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	player := nextAISpeechPlayer(room, lastSpeech.PlayerID)
	if player == nil || player.AI == nil {
		room.LastAISpeechSourceID = lastSpeech.ID
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	room.LastAISpeechSourceID = lastSpeech.ID
	input := aiplayer.DecisionInput{
		Game:        "uno",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID + ":speech",
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State: map[string]any{
			"phase":        room.Phase,
			"activeColor":  room.ActiveColor,
			"topCard":      discardTopCard(room),
			"handCount":    len(player.Hand),
			"recentSpeech": recentSpeeches(room),
			"speechGuide":  "像 UNO 朋友局里自然接一句，短句即可；如果没必要回应就跳过。",
		},
		Actions: speechActions(),
	}
	updatedAt := room.UpdatedAt
	playerID := player.ID
	room.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), aiplayer.DecisionTimeout)
	decision, err := m.aiProvider.Decide(ctx, input)
	cancel()
	if err != nil {
		return PublicRoom{}, false, err
	}
	if decision.ActionID != "speak" || strings.TrimSpace(decision.Speech) == "" {
		return PublicRoom{}, false, nil
	}

	room, err = m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	defer room.mu.Unlock()
	player = findPlayerByID(room, playerID)
	if player == nil || !player.IsAI || !room.UpdatedAt.Equal(updatedAt) {
		return publicRoom(room, ""), false, nil
	}
	if !recordSpeech(room, player, decision.Speech) {
		return publicRoom(room, ""), false, nil
	}
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, ""), true, nil
}

func (m *Manager) RenamePlayer(roomID string, actorID string, displayName string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	nextName, err := roommeta.NormalizeDisplayName(displayName)
	if err != nil {
		return PublicRoom{}, err
	}
	oldName := player.Name
	player.Name = nextName
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 改名为 %s。", oldName, nextName)))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need_two_players")
	}

	deck := shuffle(createDeck(room.Rules))
	room.DrawPile = deck
	room.DiscardPile = nil
	room.Direction = 1
	room.CurrentPlayerIndex = 0
	room.WinnerID = ""
	room.PendingDrawCount = 0
	room.PendingDrawKind = ""
	room.FlipSide = false
	room.TurnDeadline = nil
	room.Phase = PhasePlaying
	room.Log = append(room.Log, createLog("游戏开始。"))
	for _, player := range room.Players {
		player.Hand = drawCards(room, 7)
		player.HandCount = len(player.Hand)
		player.NeedsUNO = false
	}
	for len(room.DrawPile) > 0 {
		card := drawCards(room, 1)[0]
		if card.Color != ColorWild || room.Rules.AllWild {
			room.DiscardPile = append(room.DiscardPile, card)
			room.ActiveColor = startingColor(card)
			break
		}
		room.DrawPile = append(room.DrawPile, card)
	}

	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	return publicRoom(room, actorID), nil
}

func (m *Manager) Play(roomID string, actorID string, cardID string, color Color) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()

	player, err := playingPlayer(room, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

	cardIndex := slices.IndexFunc(player.Hand, func(card Card) bool {
		return card.ID == cardID
	})
	if cardIndex < 0 {
		return PublicRoom{}, errors.New("card_not_found")
	}

	card := player.Hand[cardIndex]
	isCurrentPlayer := room.CurrentPlayerIndex < len(room.Players) && room.Players[room.CurrentPlayerIndex].ID == player.ID
	if isCurrentPlayer && !isPlayable(card, room) {
		return PublicRoom{}, errors.New("card_not_playable")
	}
	if !isCurrentPlayer && !canJumpIn(card, player, room) {
		return PublicRoom{}, errors.New("card_not_playable")
	}
	if card.Color == ColorWild && !isRealColor(color) {
		return PublicRoom{}, errors.New("wild_color_required")
	}
	if room.CurrentPlayerIndex >= len(room.Players) || room.Players[room.CurrentPlayerIndex].ID != player.ID {
		room.CurrentPlayerIndex = playerIndex(room, player.ID)
	}

	playCard(room, player, cardIndex, color)
	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	return publicRoom(room, actorID), nil
}

func (m *Manager) Draw(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()

	player, err := currentHuman(room, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

	drawCount := 1
	if room.PendingDrawCount > 0 {
		drawCount = room.PendingDrawCount
		room.PendingDrawCount = 0
		room.PendingDrawKind = ""
	}
	drawn := drawCards(room, drawCount)
	player.Hand = append(player.Hand, drawn...)
	player.HandCount = len(player.Hand)
	player.NeedsUNO = false
	message := fmt.Sprintf("%s 摸了 %d 张牌。", player.Name, len(drawn))
	room.Log = append(room.Log, createLog(message))
	recordDrawAction(room, player, player, len(drawn), message)
	advanceTurn(room)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) CallUNO(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	player := findPlayerByUserID(room, actorID)
	if player == nil || len(player.Hand) != 1 || !player.NeedsUNO {
		return PublicRoom{}, errors.New("uno_call_unavailable")
	}

	player.NeedsUNO = false
	message := fmt.Sprintf("%s 喊了 UNO。", player.Name)
	room.Log = append(room.Log, createLog(message))
	recordEffectAction(room, player, player, message)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) CatchUNO(roomID string, actorID string, targetID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	actor := findPlayerByUserID(room, actorID)
	target := findPlayerByID(room, targetID)
	if actor == nil || target == nil || !target.NeedsUNO {
		return PublicRoom{}, errors.New("uno_catch_unavailable")
	}

	drawn := drawCards(room, 2)
	target.Hand = append(target.Hand, drawn...)
	target.HandCount = len(target.Hand)
	target.NeedsUNO = false
	message := fmt.Sprintf("%s 抓到 %s 没喊 UNO，%s 摸了 %d 张牌。", actor.Name, target.Name, target.Name, len(drawn))
	room.Log = append(room.Log, createLog(message))
	recordDrawAction(room, actor, target, len(drawn), message)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) RunNextAI(roomID string) (PublicRoom, bool, error) {
	startedAt := time.Now()

	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	defer room.mu.Unlock()
	if room.Phase != PhasePlaying || len(room.Players) == 0 {
		return publicRoom(room, ""), false, nil
	}

	player := room.Players[room.CurrentPlayerIndex]
	if !player.IsAI {
		return publicRoom(room, ""), false, nil
	}

	level := ""
	if player.AI != nil {
		level = player.AI.Level
	}
	slog.Info("uno ai turn started",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
	)

	action, speech := m.chooseAIAction(room, player)
	if action.Kind == "draw" {
		drawCount := 1
		if room.PendingDrawCount > 0 {
			drawCount = room.PendingDrawCount
			room.PendingDrawCount = 0
			room.PendingDrawKind = ""
		}
		drawn := drawCards(room, drawCount)
		player.Hand = append(player.Hand, drawn...)
		player.HandCount = len(player.Hand)
		message := fmt.Sprintf("%s 摸了 %d 张牌。", player.Name, len(drawn))
		room.Log = append(room.Log, createLog(message))
		recordDrawAction(room, player, player, len(drawn), message)
		advanceTurn(room)
	} else {
		playCard(room, player, action.CardIndex, action.Color)
	}
	recordSpeech(room, player, speech)

	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	shouldContinue := room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
	slog.Info("uno ai turn completed",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
		"action", action.Kind,
		"duration", time.Since(startedAt),
		"continue", shouldContinue,
	)
	return publicRoom(room, ""), shouldContinue, nil
}

func hasPendingAIRequiredAction(room *Room) bool {
	return room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
}

func (m *Manager) Tick(now time.Time) TickResult {
	now = now.UTC()
	m.mu.RLock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, room := range m.rooms {
		rooms = append(rooms, room)
	}
	m.mu.RUnlock()

	result := TickResult{}
	for _, room := range rooms {
		room.mu.Lock()
		roomID := room.ID
		destroy := shouldDestroyOfflineRoom(room, now)
		if destroy {
			room.mu.Unlock()
			result.DestroyedRoomIDs = append(result.DestroyedRoomIDs, roomID)
			continue
		}

		if room.Phase == PhasePlaying && len(room.Players) > 0 {
			result.BroadcastRoomIDs = append(result.BroadcastRoomIDs, roomID)
			if applyTimeoutTurn(room, now) {
				result.ScheduleAIRoomIDs = append(result.ScheduleAIRoomIDs, roomID)
			}
			if room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI {
				result.ScheduleAIRoomIDs = append(result.ScheduleAIRoomIDs, roomID)
			}
		}
		room.mu.Unlock()
	}

	if len(result.DestroyedRoomIDs) > 0 {
		m.mu.Lock()
		for _, roomID := range result.DestroyedRoomIDs {
			delete(m.rooms, roomID)
		}
		m.mu.Unlock()
	}

	return result
}

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	return publicRoom(room, viewerID), nil
}

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

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	m.mu.RLock()
	defer m.mu.RUnlock()
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}

func (m *Manager) lockRoom(roomID string) (*Room, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, err
	}
	room.mu.Lock()
	return room, nil
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

func playCard(room *Room, player *Player, cardIndex int, color Color) {
	card := player.Hand[cardIndex]
	player.Hand = slices.Delete(player.Hand, cardIndex, cardIndex+1)
	player.HandCount = len(player.Hand)
	player.NeedsUNO = false
	room.DiscardPile = append(room.DiscardPile, card)
	if card.Color == ColorWild {
		room.ActiveColor = color
	} else {
		room.ActiveColor = card.Color
	}
	message := fmt.Sprintf("%s 打出了 %s。", player.Name, formatCard(card))
	room.Log = append(room.Log, createLog(message))
	recordPlayAction(room, player, card, message)

	if len(player.Hand) == 0 {
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		winMessage := fmt.Sprintf("%s 获胜。", player.Name)
		room.Log = append(room.Log, createLog(winMessage))
		recordAction(room, PublicAction{
			Type:      ActionWin,
			ActorID:   player.ID,
			ActorName: player.Name,
			TargetID:  player.ID,
			Message:   winMessage,
		})
		return
	}
	if len(player.Hand) == 1 {
		player.NeedsUNO = !player.IsAI
		if player.IsAI {
			room.Log = append(room.Log, createLog(fmt.Sprintf("%s 喊了 UNO。", player.Name)))
		}
	}

	applyEffect(room, card)
}

func applyEffect(room *Room, card Card) {
	if applySevenZero(room, card) {
		advanceTurn(room)
		return
	}

	switch card.Kind {
	case KindSkip:
		target := room.Players[nextIndex(room)]
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], target, fmt.Sprintf("%s 被跳过。", target.Name))
		advanceTurn(room)
		advanceTurn(room)
	case KindReverse:
		room.Direction *= -1
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "回合方向反转。")
		if len(room.Players) == 2 {
			advanceTurn(room)
			advanceTurn(room)
			return
		}
		advanceTurn(room)
	case KindDrawTwo:
		applyDrawPenalty(room, card, 2)
	case KindWildDrawFour:
		applyDrawPenalty(room, card, 4)
	case KindWildDrawSix:
		applyDrawPenalty(room, card, 6)
	case KindWildDrawTen:
		applyDrawPenalty(room, card, 10)
	case KindFlip:
		room.FlipSide = !room.FlipSide
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "牌桌翻面。")
		advanceTurn(room)
	default:
		advanceTurn(room)
	}
}

func applyDrawPenalty(room *Room, card Card, count int) {
	if room.Rules.Stacking || room.Rules.NoMercy {
		room.PendingDrawCount += count
		room.PendingDrawKind = card.Kind
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[nextIndex(room)], fmt.Sprintf("罚牌累积到 %d 张。", room.PendingDrawCount))
		advanceTurn(room)
		return
	}

	next := nextIndex(room)
	drawn := drawCards(room, count)
	room.Players[next].Hand = append(room.Players[next].Hand, drawn...)
	room.Players[next].HandCount = len(room.Players[next].Hand)
	recordDrawAction(room, room.Players[room.CurrentPlayerIndex], room.Players[next], len(drawn), fmt.Sprintf("%s 摸了 %d 张牌。", room.Players[next].Name, len(drawn)))
	advanceTurn(room)
	advanceTurn(room)
}

func applySevenZero(room *Room, card Card) bool {
	if !room.Rules.SevenZero || card.Kind != KindNumber || card.Value == nil {
		return false
	}
	if *card.Value == 7 && len(room.Players) > 1 {
		current := room.Players[room.CurrentPlayerIndex]
		target := room.Players[nextIndex(room)]
		current.Hand, target.Hand = target.Hand, current.Hand
		current.HandCount = len(current.Hand)
		target.HandCount = len(target.Hand)
		recordEffectAction(room, current, target, fmt.Sprintf("%s 与 %s 交换手牌。", current.Name, target.Name))
		return true
	}
	if *card.Value == 0 && len(room.Players) > 1 {
		rotateHands(room)
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "所有玩家按当前方向轮换手牌。")
		return true
	}
	return false
}

func isPlayable(card Card, room *Room) bool {
	if len(room.DiscardPile) == 0 {
		return true
	}
	if room.Rules.AllWild {
		return true
	}
	if room.PendingDrawCount > 0 {
		return canStackDraw(card, room)
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	if card.Color == ColorWild || card.Color == room.ActiveColor {
		return true
	}
	if card.Kind == KindNumber {
		return top.Kind == KindNumber && card.Value != nil && top.Value != nil && *card.Value == *top.Value
	}
	return card.Kind == top.Kind
}

func canStackDraw(card Card, room *Room) bool {
	if !(room.Rules.Stacking || room.Rules.NoMercy) {
		return false
	}
	if room.Rules.NoMercy {
		return drawPenalty(card.Kind) > 0
	}
	return card.Kind == room.PendingDrawKind && drawPenalty(card.Kind) > 0
}

func canJumpIn(card Card, player *Player, room *Room) bool {
	if !room.Rules.JumpIn || len(room.DiscardPile) == 0 || room.PendingDrawCount > 0 {
		return false
	}
	if room.CurrentPlayerIndex < len(room.Players) && room.Players[room.CurrentPlayerIndex].ID == player.ID {
		return false
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	return sameFace(card, top)
}

func playableCardsForPlayer(player *Player, room *Room) []string {
	playableCardIDs := []string{}
	for _, card := range player.Hand {
		if isPlayable(card, room) || canJumpIn(card, player, room) {
			playableCardIDs = append(playableCardIDs, card.ID)
		}
	}
	return playableCardIDs
}

func drawCards(room *Room, count int) []Card {
	cards := []Card{}
	for range count {
		if len(room.DrawPile) == 0 {
			recycleDiscards(room)
		}
		if len(room.DrawPile) == 0 {
			return cards
		}
		card := room.DrawPile[len(room.DrawPile)-1]
		room.DrawPile = room.DrawPile[:len(room.DrawPile)-1]
		cards = append(cards, card)
	}
	return cards
}

func recycleDiscards(room *Room) {
	if len(room.DiscardPile) <= 1 {
		return
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	recycled := shuffle(room.DiscardPile[:len(room.DiscardPile)-1])
	room.DiscardPile = []Card{top}
	room.DrawPile = append(room.DrawPile, recycled...)
}

func rotateHands(room *Room) {
	hands := make([][]Card, len(room.Players))
	for index, player := range room.Players {
		hands[index] = player.Hand
	}
	for index, player := range room.Players {
		source := (index - room.Direction + len(room.Players)) % len(room.Players)
		player.Hand = hands[source]
		player.HandCount = len(player.Hand)
	}
}

func advanceTurn(room *Room) {
	room.CurrentPlayerIndex = nextIndex(room)
	refreshTurnDeadline(room, time.Now().UTC())
}

func nextIndex(room *Room) int {
	total := len(room.Players)
	return (room.CurrentPlayerIndex + room.Direction + total) % total
}

func refreshTurnDeadline(room *Room, now time.Time) {
	if room.Phase != PhasePlaying || len(room.Players) == 0 {
		room.TurnDeadline = nil
		return
	}
	deadline := now.UTC().Add(turnTimeout)
	room.TurnDeadline = &deadline
}

func applyTimeoutTurn(room *Room, now time.Time) bool {
	if room.Phase != PhasePlaying || len(room.Players) == 0 || room.TurnDeadline == nil || now.Before(*room.TurnDeadline) {
		return false
	}
	if room.CurrentPlayerIndex >= len(room.Players) {
		room.CurrentPlayerIndex = 0
	}

	player := room.Players[room.CurrentPlayerIndex]
	actions := legalAIActions(room, player)
	if len(actions) == 0 {
		actions = []aiTurnAction{{Kind: "draw"}}
	}
	action := actions[0]
	timeoutMessage := fmt.Sprintf("%s 超时，服务端自动行动。", player.Name)
	room.Log = append(room.Log, createLog(timeoutMessage))
	recordEffectAction(room, player, player, timeoutMessage)

	if action.Kind == "draw" {
		drawCount := 1
		if room.PendingDrawCount > 0 {
			drawCount = room.PendingDrawCount
			room.PendingDrawCount = 0
			room.PendingDrawKind = ""
		}
		drawn := drawCards(room, drawCount)
		player.Hand = append(player.Hand, drawn...)
		player.HandCount = len(player.Hand)
		player.NeedsUNO = false
		message := fmt.Sprintf("%s 超时后自动摸了 %d 张牌。", player.Name, len(drawn))
		room.Log = append(room.Log, createLog(message))
		recordDrawAction(room, player, player, len(drawn), message)
		advanceTurn(room)
	} else {
		playCard(room, player, action.CardIndex, action.Color)
	}

	room.UpdatedAt = now
	refreshTurnDeadline(room, now)
	slog.Info("uno turn timeout auto action",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"action", action.Kind,
	)
	return true
}

func shouldDestroyOfflineRoom(room *Room, now time.Time) bool {
	if room.Phase != PhaseLobby && room.Phase != PhasePlaying {
		return false
	}
	updateAllHumansOfflineSince(room, now)
	return room.AllHumansOfflineSince != nil && now.Sub(*room.AllHumansOfflineSince) >= offlineRoomTTL
}

func updateAllHumansOfflineSince(room *Room, now time.Time) {
	for _, player := range room.Players {
		if player.IsAI {
			continue
		}
		if player.Connected {
			room.AllHumansOfflineSince = nil
			return
		}
	}
	if room.AllHumansOfflineSince == nil {
		startedAt := now.UTC()
		room.AllHumansOfflineSince = &startedAt
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func turnRemainingSeconds(deadline *time.Time) int {
	if deadline == nil {
		return 0
	}
	remaining := time.Until(*deadline)
	if remaining <= 0 {
		return 0
	}
	return int((remaining + time.Second - time.Nanosecond) / time.Second)
}

func createDeck(rules RuleSet) []Card {
	cards := []Card{}
	if rules.AllWild {
		for range 18 {
			cards = append(cards, createCard(ColorWild, KindWild))
		}
		for range 8 {
			cards = append(cards, createCard(ColorWild, KindSkip), createCard(ColorWild, KindReverse), createCard(ColorWild, KindDrawTwo))
		}
		for range 4 {
			cards = append(cards, createCard(ColorWild, KindWildDrawFour), createCard(ColorWild, KindWildDrawSix))
		}
		return cards
	}
	for _, color := range colors {
		for value := range 10 {
			cards = append(cards, createNumberCard(color, value))
			if value != 0 {
				cards = append(cards, createNumberCard(color, value))
			}
		}
		for range 2 {
			cards = append(cards, createCard(color, KindSkip), createCard(color, KindReverse), createCard(color, KindDrawTwo))
		}
	}
	for range 4 {
		cards = append(cards, createCard(ColorWild, KindWild), createCard(ColorWild, KindWildDrawFour))
	}
	if rules.Flip {
		for range 8 {
			cards = append(cards, createCard(ColorWild, KindFlip))
		}
	}
	if rules.NoMercy {
		for range 4 {
			cards = append(cards, createCard(ColorWild, KindWildDrawSix), createCard(ColorWild, KindWildDrawTen))
		}
	}
	return cards
}

func createNumberCard(color Color, value int) Card {
	return Card{ID: "card_" + randomToken(10), Color: color, Kind: KindNumber, Value: &value}
}

func createCard(color Color, kind Kind) Card {
	return Card{ID: "card_" + randomToken(10), Color: color, Kind: kind}
}

func shuffle(cards []Card) []Card {
	shuffled := append([]Card(nil), cards...)
	rand.Shuffle(len(shuffled), func(i int, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func chooseBestColor(hand []Card) Color {
	score := map[Color]int{}
	for _, card := range hand {
		if isRealColor(card.Color) {
			score[card.Color]++
		}
	}

	best := ColorRed
	for _, color := range colors {
		if score[color] > score[best] {
			best = color
		}
	}
	return best
}

func startingColor(card Card) Color {
	if isRealColor(card.Color) {
		return card.Color
	}
	return ColorRed
}

func sameFace(a Card, b Card) bool {
	if a.Color != b.Color || a.Kind != b.Kind {
		return false
	}
	if a.Kind != KindNumber {
		return true
	}
	if a.Value == nil || b.Value == nil {
		return false
	}
	return *a.Value == *b.Value
}

func drawPenalty(kind Kind) int {
	switch kind {
	case KindDrawTwo:
		return 2
	case KindWildDrawFour:
		return 4
	case KindWildDrawSix:
		return 6
	case KindWildDrawTen:
		return 10
	default:
		return 0
	}
}

func isRealColor(color Color) bool {
	return color == ColorRed || color == ColorYellow || color == ColorGreen || color == ColorBlue
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

type aiTurnAction struct {
	Kind      string
	CardIndex int
	Color     Color
}

func (m *Manager) chooseAIAction(room *Room, player *Player) (aiTurnAction, string) {
	actions := legalAIActions(room, player)
	if len(actions) == 0 {
		return aiTurnAction{Kind: "draw"}, ""
	}

	level := aiplayer.NormalizeLevel(player.AI.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	switch level {
	case aiplayer.LevelBeginner:
		return actions[rand.IntN(len(actions))], ""
	case aiplayer.LevelMaster:
		return bestUnoAction(actions, player.Hand), ""
	case aiplayer.LevelLLM:
		decision, err := m.decideWithLLM(room, player, actions)
		if err == nil {
			for _, action := range actions {
				if actionID(action, player.Hand) == decision.ActionID {
					return action, decision.Speech
				}
			}
		} else {
			slog.Warn("uno llm decision failed, falling back",
				"room", room.ID,
				"player", player.ID,
				"playerName", player.Name,
				"actionCount", len(actions),
				"error", err,
			)
		}
		return bestUnoAction(actions, player.Hand), ""
	default:
		return actions[0], ""
	}
}

func (m *Manager) decideWithLLM(room *Room, player *Player, actions []aiTurnAction) (aiplayer.Decision, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return aiplayer.Decision{}, errors.New("llm_not_configured")
	}

	legalActions := make([]aiplayer.LegalAction, 0, len(actions))
	for _, action := range actions {
		legalActions = append(legalActions, aiplayer.LegalAction{
			ID:          actionID(action, player.Hand),
			Label:       actionLabel(action, player.Hand),
			Description: actionDescription(action),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiplayer.DecisionTimeout)
	defer cancel()
	return m.aiProvider.Decide(ctx, aiplayer.DecisionInput{
		Game:        "uno",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID,
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State: map[string]any{
			"activeColor":      room.ActiveColor,
			"direction":        room.Direction,
			"pendingDrawCount": room.PendingDrawCount,
			"topCard":          discardTopCard(room),
			"hand":             player.Hand,
			"opponents":        publicOpponentCounts(room, player.ID),
			"recentSpeech":     recentSpeeches(room),
			"speechGuide":      "UNO 发言像普通朋友局：短句、自然、可以吐槽牌不好或提醒颜色，不要中二台词，不要解释规则。",
		},
		Actions: legalActions,
	})
}

func legalAIActions(room *Room, player *Player) []aiTurnAction {
	actions := []aiTurnAction{}
	for index, card := range player.Hand {
		if !isPlayable(card, room) {
			continue
		}
		if card.Color == ColorWild {
			for _, color := range colors {
				actions = append(actions, aiTurnAction{Kind: "play", CardIndex: index, Color: color})
			}
			continue
		}
		actions = append(actions, aiTurnAction{Kind: "play", CardIndex: index, Color: card.Color})
	}
	if len(actions) == 0 {
		return []aiTurnAction{{Kind: "draw"}}
	}
	return actions
}

func discardTopCard(room *Room) *Card {
	if len(room.DiscardPile) == 0 {
		return nil
	}
	card := room.DiscardPile[len(room.DiscardPile)-1]
	return &card
}

func bestUnoAction(actions []aiTurnAction, hand []Card) aiTurnAction {
	best := actions[0]
	bestScore := -1
	for _, action := range actions {
		score := unoActionScore(action, hand)
		if score > bestScore {
			best = action
			bestScore = score
		}
	}
	return best
}

func unoActionScore(action aiTurnAction, hand []Card) int {
	if action.Kind == "draw" {
		return 0
	}
	card := hand[action.CardIndex]
	score := 10
	switch card.Kind {
	case KindWildDrawTen:
		score += 90
	case KindWildDrawSix:
		score += 70
	case KindWildDrawFour:
		score += 60
	case KindDrawTwo:
		score += 40
	case KindSkip, KindReverse, KindFlip:
		score += 24
	case KindWild:
		score += 18
	}
	if len(hand) <= 2 {
		score += 50
	}
	return score
}

func actionID(action aiTurnAction, hand []Card) string {
	if action.Kind == "draw" {
		return "draw"
	}
	return fmt.Sprintf("play:%s:%s", hand[action.CardIndex].ID, action.Color)
}

func actionLabel(action aiTurnAction, hand []Card) string {
	if action.Kind == "draw" {
		return "摸牌"
	}
	return fmt.Sprintf("打出 %s，指定 %s", formatCard(hand[action.CardIndex]), action.Color)
}

func actionDescription(action aiTurnAction) string {
	if action.Kind == "draw" {
		return "没有合适出牌时摸牌并结束回合。"
	}
	return "服务端已确认这是合法出牌。"
}

func publicOpponentCounts(room *Room, playerID string) []map[string]any {
	opponents := []map[string]any{}
	for _, player := range room.Players {
		if player.ID == playerID {
			continue
		}
		opponents = append(opponents, map[string]any{
			"name":      player.Name,
			"handCount": player.HandCount,
		})
	}
	return opponents
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
