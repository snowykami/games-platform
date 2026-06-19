package uno

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	minPlayers = 2
	maxPlayers = 10
)

var colors = []Color{ColorRed, ColorYellow, ColorGreen, ColorBlue}

type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: map[string]*Room{}}
}

func (m *Manager) CreateRoom(user UserView, options RoomOptions) PublicRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := &Room{
		ID:         createRoomID(),
		HostUserID: user.ID,
		VariantKey: normalizeVariantKey(options.VariantKey),
		ThemeKey:   normalizeThemeKey(options.ThemeKey),
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
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}

	player := findPlayerByUserID(room, user.ID)
	if player == nil {
		if room.Phase != PhaseLobby {
			return PublicRoom{}, errors.New("game already started")
		}
		if len(room.Players) >= maxPlayers {
			return PublicRoom{}, errors.New("room is full")
		}

		player = createHumanPlayer(user, "player")
		room.Players = append(room.Players, player)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 加入了房间。", user.DisplayName)))
	}

	player.Connected = true
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, user.ID), nil
}

func (m *Manager) Leave(roomID string, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return
	}

	if player := findPlayerByUserID(room, userID); player != nil {
		player.Connected = false
	}
}

func (m *Manager) AddAI(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only host can add ai")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("ai can only be added in lobby")
	}
	if len(room.Players) >= maxPlayers {
		return PublicRoom{}, errors.New("room is full")
	}

	profile := nextAIProfile(room)
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

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only host can start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game already started")
	}
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need at least two players")
	}

	deck := shuffle(createDeck())
	room.DrawPile = deck
	room.DiscardPile = nil
	room.Direction = 1
	room.CurrentPlayerIndex = 0
	room.WinnerID = ""
	room.Phase = PhasePlaying
	room.Log = append(room.Log, createLog("游戏开始。"))
	for _, player := range room.Players {
		player.Hand = drawCards(room, 7)
		player.HandCount = len(player.Hand)
	}
	for len(room.DrawPile) > 0 {
		card := drawCards(room, 1)[0]
		if card.Color != ColorWild {
			room.DiscardPile = append(room.DiscardPile, card)
			room.ActiveColor = card.Color
			break
		}
		room.DrawPile = append(room.DrawPile, card)
	}

	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Play(roomID string, actorID string, cardID string, color Color) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

	cardIndex := slices.IndexFunc(player.Hand, func(card Card) bool {
		return card.ID == cardID
	})
	if cardIndex < 0 {
		return PublicRoom{}, errors.New("card not found")
	}

	card := player.Hand[cardIndex]
	if !isPlayable(card, room) {
		return PublicRoom{}, errors.New("card is not playable")
	}
	if card.Color == ColorWild && !isRealColor(color) {
		return PublicRoom{}, errors.New("wild color required")
	}

	playCard(room, player, cardIndex, color)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Draw(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

	drawn := drawCards(room, 1)
	player.Hand = append(player.Hand, drawn...)
	player.HandCount = len(player.Hand)
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 摸了一张牌。", player.Name)))
	recordDrawAction(room, player, player, len(drawn), fmt.Sprintf("%s 摸了一张牌。", player.Name))
	advanceTurn(room)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) RunNextAI(roomID string) (PublicRoom, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase != PhasePlaying || len(room.Players) == 0 {
		return publicRoom(room, ""), false, nil
	}

	player := room.Players[room.CurrentPlayerIndex]
	if !player.IsAI {
		return publicRoom(room, ""), false, nil
	}

	cardIndex := slices.IndexFunc(player.Hand, func(card Card) bool {
		return isPlayable(card, room)
	})
	if cardIndex < 0 {
		drawn := drawCards(room, 1)
		player.Hand = append(player.Hand, drawn...)
		player.HandCount = len(player.Hand)
		message := fmt.Sprintf("%s 摸了一张牌。", player.Name)
		room.Log = append(room.Log, createLog(message))
		recordDrawAction(room, player, player, len(drawn), message)
		advanceTurn(room)
	} else {
		playCard(room, player, cardIndex, chooseBestColor(player.Hand))
	}

	room.UpdatedAt = time.Now().UTC()
	shouldContinue := room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
	return publicRoom(room, ""), shouldContinue, nil
}

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	return publicRoom(room, viewerID), nil
}

func (m *Manager) currentHuman(roomID string, actorID string) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Phase != PhasePlaying {
		return nil, nil, errors.New("game is not playing")
	}
	if len(room.Players) == 0 || room.CurrentPlayerIndex >= len(room.Players) {
		return nil, nil, errors.New("invalid turn")
	}

	player := room.Players[room.CurrentPlayerIndex]
	if player.UserID != actorID || player.IsAI {
		return nil, nil, errors.New("not your turn")
	}
	return room, player, nil
}

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room not found")
	}
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
		ID:              room.ID,
		HostUserID:      room.HostUserID,
		VariantKey:      room.VariantKey,
		ThemeKey:        room.ThemeKey,
		Phase:           room.Phase,
		Players:         players,
		TopCard:         topCard,
		DrawPileCount:   len(room.DrawPile),
		CurrentPlayerID: currentPlayerID,
		Direction:       room.Direction,
		ActiveColor:     room.ActiveColor,
		PlayableCardIDs: playableCardIDs,
		WinnerID:        room.WinnerID,
		Log:             append([]LogEntry{}, logs...),
		ActionSeq:       room.ActionSeq,
		RecentActions:   append([]PublicAction{}, room.RecentActions...),
	}
}

func playCard(room *Room, player *Player, cardIndex int, color Color) {
	card := player.Hand[cardIndex]
	player.Hand = slices.Delete(player.Hand, cardIndex, cardIndex+1)
	player.HandCount = len(player.Hand)
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

	applyEffect(room, card)
}

func applyEffect(room *Room, card Card) {
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
		next := nextIndex(room)
		drawn := drawCards(room, 2)
		room.Players[next].Hand = append(room.Players[next].Hand, drawn...)
		room.Players[next].HandCount = len(room.Players[next].Hand)
		recordDrawAction(room, room.Players[room.CurrentPlayerIndex], room.Players[next], len(drawn), fmt.Sprintf("%s 摸了 %d 张牌。", room.Players[next].Name, len(drawn)))
		advanceTurn(room)
		advanceTurn(room)
	case KindWildDrawFour:
		next := nextIndex(room)
		drawn := drawCards(room, 4)
		room.Players[next].Hand = append(room.Players[next].Hand, drawn...)
		room.Players[next].HandCount = len(room.Players[next].Hand)
		recordDrawAction(room, room.Players[room.CurrentPlayerIndex], room.Players[next], len(drawn), fmt.Sprintf("%s 摸了 %d 张牌。", room.Players[next].Name, len(drawn)))
		advanceTurn(room)
		advanceTurn(room)
	default:
		advanceTurn(room)
	}
}

func isPlayable(card Card, room *Room) bool {
	if len(room.DiscardPile) == 0 {
		return true
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

func playableCardsForPlayer(player *Player, room *Room) []string {
	playableCardIDs := []string{}
	for _, card := range player.Hand {
		if isPlayable(card, room) {
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

func advanceTurn(room *Room) {
	room.CurrentPlayerIndex = nextIndex(room)
}

func nextIndex(room *Room) int {
	total := len(room.Players)
	return (room.CurrentPlayerIndex + room.Direction + total) % total
}

func createDeck() []Card {
	cards := []Card{}
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

func nextAIProfile(room *Room) AIProfile {
	profiles := []AIProfile{
		{Name: "北风", Personality: "谨慎控牌，喜欢保留功能牌。"},
		{Name: "南星", Personality: "进攻型玩家，优先压缩手牌。"},
		{Name: "阿澈", Personality: "观察型玩家，偏好改变颜色。"},
		{Name: "小满", Personality: "轻快随机，偶尔打出意外节奏。"},
	}
	for _, profile := range profiles {
		if findPlayerByName(room, profile.Name) == nil {
			return profile
		}
	}
	return AIProfile{Name: fmt.Sprintf("AI %d", len(room.Players)), Personality: "规则驱动玩家，后续可由 LLM provider 生成完整人格。"}
}

func findPlayerByName(room *Room, name string) *Player {
	for _, player := range room.Players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func createRoomID() string {
	return strings.ToUpper(randomToken(6))
}

func normalizeVariantKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	switch key {
	case "", "classic":
		return "classic"
	case "party", "stacking", "wild-plus":
		return key
	default:
		return "classic"
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

func formatCard(card Card) string {
	if card.Kind == KindNumber && card.Value != nil {
		return fmt.Sprintf("%s %d", card.Color, *card.Value)
	}
	return fmt.Sprintf("%s %s", card.Color, card.Kind)
}
