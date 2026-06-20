package mahjong

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/roommeta"
)

const (
	minPlayers = 4
	maxPlayers = 4
)

var (
	winds = []Wind{WindEast, WindSouth, WindWest, WindNorth}
	rules = RuleSet{
		ID:          "chinese-official",
		Name:        "国标麻将",
		MinFan:      8,
		Description: "首版实现国标 8 番起胡与常用番型子集，规则层可继续扩展。",
	}
)

type Manager struct {
	aiProvider aiplayer.Provider
	mu         sync.Mutex
	rooms      map[string]*Room
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{aiProvider: aiProvider, rooms: map[string]*Room{}}
}

func (m *Manager) CreateRoom(user UserView) PublicRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := &Room{
		ID:         createRoomID(),
		HostUserID: user.ID,
		Phase:      PhaseLobby,
		RuleSet:    rules,
		RoundWind:  WindEast,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	room.Players = append(room.Players, createHumanPlayer(user, "host"))
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 创建了麻将房间。", user.DisplayName)))
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
	now := time.Now().UTC()
	if player := findPlayerByUserID(room, userID); player != nil {
		player.Connected = false
		if !player.IsAI {
			player.DisconnectedAt = &now
		}
	}
}

func (m *Manager) AddAI(roomID string, actorID string, options AIOptions) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
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
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
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
	m.mu.Lock()
	defer m.mu.Unlock()
	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_remove_player")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("remove_player_only_lobby")
	}
	for index, player := range room.Players {
		if player.ID != playerID {
			continue
		}
		if player.UserID == room.HostUserID || player.Role == "host" {
			return PublicRoom{}, errors.New("cannot_remove_host")
		}
		room.Players = append(room.Players[:index], room.Players[index+1:]...)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 被房主移出了房间。", player.Name)))
		room.UpdatedAt = time.Now().UTC()
		return publicRoom(room, actorID), nil
	}
	return PublicRoom{}, errors.New("player_not_found")
}

func (m *Manager) Say(roomID string, actorID string, text string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
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
	m.mu.Lock()
	room, err := m.room(roomID)
	if err != nil {
		m.mu.Unlock()
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || len(room.Speeches) == 0 {
		view := publicRoom(room, "")
		m.mu.Unlock()
		return view, false, nil
	}
	if hasPendingAIRequiredAction(room) {
		view := publicRoom(room, "")
		m.mu.Unlock()
		return view, false, nil
	}
	lastSpeech := room.Speeches[len(room.Speeches)-1]
	if lastSpeech.ID == room.LastAISpeechSourceID {
		view := publicRoom(room, "")
		m.mu.Unlock()
		return view, false, nil
	}
	player := nextAISpeechPlayer(room, lastSpeech.PlayerID)
	if player == nil || player.AI == nil {
		room.LastAISpeechSourceID = lastSpeech.ID
		view := publicRoom(room, "")
		m.mu.Unlock()
		return view, false, nil
	}
	room.LastAISpeechSourceID = lastSpeech.ID
	input := aiplayer.DecisionInput{
		Game:        "mahjong",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID + ":speech",
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State: map[string]any{
			"phase":        room.Phase,
			"wind":         player.Wind,
			"handCount":    len(player.Hand),
			"wallCount":    len(room.Wall),
			"recentSpeech": recentSpeeches(room),
			"speechGuide":  "像麻将桌上的自然短句，可以闲聊或轻微评价牌局，不要透露隐藏手牌。",
		},
		Actions: speechActions(),
	}
	updatedAt := room.UpdatedAt
	playerID := player.ID
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), aiplayer.DecisionTimeout)
	decision, err := m.aiProvider.Decide(ctx, input)
	cancel()
	if err != nil {
		return PublicRoom{}, false, err
	}
	if decision.ActionID != "speak" || strings.TrimSpace(decision.Speech) == "" {
		return PublicRoom{}, false, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	room, err = m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
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
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
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
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need_four_players")
	}

	resetGame(room)
	room.Phase = PhasePlaying
	room.HasDrawn = true
	room.Log = append(room.Log, createLog("东风起局，庄家先打。国标麻将 8 番起胡。"))
	recordAction(room, PublicAction{Type: ActionStart, ActorID: room.Players[0].ID, ActorName: room.Players[0].Name, Message: "麻将开始。"})
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
	if room.HasDrawn {
		return PublicRoom{}, errors.New("already_drawn")
	}
	drawForCurrent(room, player)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Discard(roomID string, actorID string, tileID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !room.HasDrawn {
		return PublicRoom{}, errors.New("must_draw_first")
	}
	if !discardTile(room, player, tileID) {
		return PublicRoom{}, errors.New("tile_not_found")
	}
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) SelfDraw(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !room.HasDrawn {
		return PublicRoom{}, errors.New("must_draw_first")
	}
	result := evaluateWin(player.Hand, player.Melds, true, player.Wind, room.RoundWind, room.RuleSet)
	if !result.CanWin {
		return PublicRoom{}, errors.New("mahjong_win_unavailable")
	}
	finishWin(room, player, result, fmt.Sprintf("%s 自摸，%d 番。", player.Name, result.Fan))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Claim(roomID string, actorID string, claimID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if room.Phase != PhaseClaiming {
		return PublicRoom{}, errors.New("claim_unavailable")
	}
	claimIndex := slices.IndexFunc(room.ClaimOptions, func(option ClaimOption) bool {
		return option.ID == claimID && option.PlayerID == player.ID
	})
	if claimIndex < 0 {
		return PublicRoom{}, errors.New("claim_unavailable")
	}
	applyClaim(room, room.ClaimOptions[claimIndex])
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) SkipClaims(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if room.Phase != PhaseClaiming {
		return PublicRoom{}, errors.New("claim_unavailable")
	}

	room.ClaimOptions = slices.DeleteFunc(room.ClaimOptions, func(option ClaimOption) bool {
		return option.PlayerID == player.ID
	})
	if len(room.ClaimOptions) == 0 {
		advanceAfterClaims(room)
	} else {
		room.Log = append(room.Log, createLog("暂不声明。"))
	}
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

	if !room.HasDrawn {
		drawForCurrent(room, player)
		result := evaluateWin(player.Hand, player.Melds, true, player.Wind, room.RoundWind, room.RuleSet)
		if result.CanWin {
			finishWin(room, player, result, fmt.Sprintf("%s 自摸，%d 番。", player.Name, result.Fan))
		}
		room.UpdatedAt = time.Now().UTC()
		return publicRoom(room, ""), room.Phase == PhasePlaying && room.Players[room.CurrentPlayerIndex].IsAI, nil
	}

	tile := chooseAIDiscard(player)
	discardTile(room, player, tile.ID)
	room.UpdatedAt = time.Now().UTC()
	shouldContinue := room.Phase == PhasePlaying && room.Players[room.CurrentPlayerIndex].IsAI
	return publicRoom(room, ""), shouldContinue, nil
}

func hasPendingAIRequiredAction(room *Room) bool {
	return room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
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
		return nil, nil, errors.New("game_not_playing")
	}
	if len(room.Players) == 0 || room.CurrentPlayerIndex >= len(room.Players) {
		return nil, nil, errors.New("invalid_turn")
	}

	player := room.Players[room.CurrentPlayerIndex]
	if player.UserID != actorID || player.IsAI {
		return nil, nil, errors.New("not_current_turn")
	}
	return room, player, nil
}

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}

func resetGame(room *Room) {
	wall := shuffle(createWall())
	room.Wall = wall
	room.DeadWall = append([]Tile{}, room.Wall[len(room.Wall)-14:]...)
	room.Wall = room.Wall[:len(room.Wall)-14]
	room.CurrentPlayerIndex = 0
	room.DealerIndex = 0
	room.RoundWind = WindEast
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.WinnerID = ""
	room.WinResult = WinResult{}
	room.RecentActions = nil
	for index, player := range room.Players {
		player.Wind = winds[index]
		player.Hand = nil
		player.Melds = nil
		player.Discards = nil
		count := 13
		if index == 0 {
			count = 14
		}
		player.Hand = sortTiles(shiftTiles(&room.Wall, count))
	}
}

func drawForCurrent(room *Room, player *Player) {
	if len(room.Wall) == 0 {
		room.Phase = PhaseFinished
		room.Log = append(room.Log, createLog("流局：牌墙已经摸完。"))
		return
	}
	tile := shiftTiles(&room.Wall, 1)[0]
	player.Hand = sortTiles(append(player.Hand, tile))
	room.HasDrawn = true
	room.LastDiscard = nil
	message := fmt.Sprintf("%s 摸牌。", player.Name)
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionDraw, ActorID: player.ID, ActorName: player.Name, Message: message})
}

func discardTile(room *Room, player *Player, tileID string) bool {
	tileIndex := slices.IndexFunc(player.Hand, func(tile Tile) bool { return tile.ID == tileID })
	if tileIndex < 0 {
		return false
	}
	tile := player.Hand[tileIndex]
	player.Hand = append(player.Hand[:tileIndex], player.Hand[tileIndex+1:]...)
	player.Discards = append(player.Discards, tile)
	room.HasDrawn = false
	room.LastDiscard = &LastDiscard{Tile: tile, PlayerID: player.ID}
	message := fmt.Sprintf("%s 打出 %s。", player.Name, formatTile(tile))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionDiscard, ActorID: player.ID, ActorName: player.Name, Tile: &tile, Message: message})
	openClaimWindow(room)
	return true
}

func openClaimWindow(room *Room) {
	options := createClaimOptions(room)
	botHuIndex := slices.IndexFunc(options, func(option ClaimOption) bool {
		player := findPlayerByID(room, option.PlayerID)
		return option.Kind == ClaimHu && player != nil && player.IsAI
	})
	if botHuIndex >= 0 {
		applyClaim(room, options[botHuIndex])
		return
	}

	humanOptions := slices.DeleteFunc(options, func(option ClaimOption) bool {
		player := findPlayerByID(room, option.PlayerID)
		return player == nil || player.IsAI
	})
	if len(humanOptions) > 0 {
		room.ClaimOptions = humanOptions
		room.Phase = PhaseClaiming
		room.Log = append(room.Log, createLog("有人可以声明吃、碰或胡。"))
		return
	}

	advanceAfterClaims(room)
}

func applyClaim(room *Room, claim ClaimOption) {
	player := findPlayerByID(room, claim.PlayerID)
	if player == nil || room.LastDiscard == nil {
		return
	}
	if claim.Kind == ClaimHu && claim.WinResult.CanWin {
		message := fmt.Sprintf("%s 荣和 %s，%d 番。", player.Name, formatTile(claim.Tile), claim.WinResult.Fan)
		finishWin(room, player, claim.WinResult, message)
		return
	}

	discarder := findPlayerByID(room, room.LastDiscard.PlayerID)
	if discarder != nil {
		discarder.Discards = slices.DeleteFunc(discarder.Discards, func(tile Tile) bool { return tile.ID == claim.Tile.ID })
	}
	used := map[string]bool{}
	for _, tile := range claim.TilesFromHand {
		used[tile.ID] = true
	}
	player.Hand = slices.DeleteFunc(player.Hand, func(tile Tile) bool { return used[tile.ID] })
	meldKind := MeldPung
	label := "碰"
	if claim.Kind == ClaimChi {
		meldKind = MeldChow
		label = "吃"
	}
	player.Melds = append(player.Melds, Meld{
		ID:           "meld_" + randomToken(10),
		Kind:         meldKind,
		Tiles:        sortTiles(append(append([]Tile{}, claim.TilesFromHand...), claim.Tile)),
		FromPlayerID: room.LastDiscard.PlayerID,
		Exposed:      true,
	})
	room.CurrentPlayerIndex = playerIndex(room, player.ID)
	room.HasDrawn = true
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.Phase = PhasePlaying
	message := fmt.Sprintf("%s %s了 %s。", player.Name, label, formatTile(claim.Tile))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionClaim, ActorID: player.ID, ActorName: player.Name, Tile: &claim.Tile, Message: message})
}

func finishWin(room *Room, player *Player, result WinResult, message string) {
	room.Phase = PhaseFinished
	room.WinnerID = player.ID
	room.WinResult = result
	room.ClaimOptions = nil
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionWin, ActorID: player.ID, ActorName: player.Name, Message: message})
}

func advanceAfterClaims(room *Room) {
	if room.LastDiscard == nil {
		return
	}
	room.CurrentPlayerIndex = nextPlayerIndex(room, room.LastDiscard.PlayerID)
	room.HasDrawn = false
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.Phase = PhasePlaying
}

func createClaimOptions(room *Room) []ClaimOption {
	if room.LastDiscard == nil {
		return nil
	}
	discard := room.LastDiscard
	discarderIndex := playerIndex(room, discard.PlayerID)
	nextIndex := (discarderIndex + 1) % len(room.Players)
	options := []ClaimOption{}
	for index, player := range room.Players {
		if player.ID == discard.PlayerID {
			continue
		}
		winTiles := sortTiles(append(append([]Tile{}, player.Hand...), discard.Tile))
		winResult := evaluateWin(winTiles, player.Melds, false, player.Wind, room.RoundWind, room.RuleSet)
		if winResult.CanWin {
			options = append(options, ClaimOption{
				ID:        fmt.Sprintf("hu_%s_%s", player.ID, discard.Tile.ID),
				PlayerID:  player.ID,
				Kind:      ClaimHu,
				Tile:      discard.Tile,
				WinResult: winResult,
			})
		}
		sameTiles := filterTiles(player.Hand, func(tile Tile) bool { return tile.Code == discard.Tile.Code })
		if len(sameTiles) >= 2 {
			options = append(options, ClaimOption{
				ID:            fmt.Sprintf("peng_%s_%s", player.ID, discard.Tile.ID),
				PlayerID:      player.ID,
				Kind:          ClaimPeng,
				Tile:          discard.Tile,
				TilesFromHand: sameTiles[:2],
			})
		}
		if index == nextIndex {
			options = append(options, chiOptions(player, discard.Tile)...)
		}
	}
	return options
}

func chiOptions(player *Player, tile Tile) []ClaimOption {
	if tile.Rank == 0 || !isSuited(tile.Code) {
		return nil
	}
	windows := [][]int{{tile.Rank - 2, tile.Rank - 1}, {tile.Rank - 1, tile.Rank + 1}, {tile.Rank + 1, tile.Rank + 2}}
	options := []ClaimOption{}
	for _, ranks := range windows {
		if ranks[0] < 1 || ranks[1] > 9 {
			continue
		}
		first, okFirst := firstTileByCode(player.Hand, fmt.Sprintf("%s%d", tile.Code[:1], ranks[0]))
		second, okSecond := firstTileByCode(player.Hand, fmt.Sprintf("%s%d", tile.Code[:1], ranks[1]))
		if !okFirst || !okSecond {
			continue
		}
		tiles := []Tile{first, second}
		options = append(options, ClaimOption{
			ID:            fmt.Sprintf("chi_%s_%s_%s_%s", player.ID, tile.ID, first.ID, second.ID),
			PlayerID:      player.ID,
			Kind:          ClaimChi,
			Tile:          tile,
			TilesFromHand: tiles,
		})
	}
	return options
}

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

func createWall() []Tile {
	wall := []Tile{}
	for _, suit := range []struct {
		prefix string
		kind   TileKind
	}{{"m", TileCharacters}, {"p", TileDots}, {"s", TileBamboo}} {
		for rank := 1; rank <= 9; rank++ {
			for copyIndex := 0; copyIndex < 4; copyIndex++ {
				wall = append(wall, Tile{ID: fmt.Sprintf("%s%d_%d_%s", suit.prefix, rank, copyIndex, randomToken(6)), Code: fmt.Sprintf("%s%d", suit.prefix, rank), Kind: suit.kind, Rank: rank})
			}
		}
	}
	for _, wind := range winds {
		for copyIndex := 0; copyIndex < 4; copyIndex++ {
			code := windCode(wind)
			wall = append(wall, Tile{ID: fmt.Sprintf("%s_%d_%s", code, copyIndex, randomToken(6)), Code: code, Kind: TileWind, Wind: wind})
		}
	}
	for _, dragon := range []Dragon{DragonRed, DragonGreen, DragonWhite} {
		for copyIndex := 0; copyIndex < 4; copyIndex++ {
			code := dragonCode(dragon)
			wall = append(wall, Tile{ID: fmt.Sprintf("%s_%d_%s", code, copyIndex, randomToken(6)), Code: code, Kind: TileDragon, Dragon: dragon})
		}
	}
	return wall
}

func evaluateWin(tiles []Tile, melds []Meld, selfDraw bool, seatWind Wind, roundWind Wind, ruleset RuleSet) WinResult {
	codes := tileCodes(tiles)
	if len(melds) == 0 && isSevenPairs(codes) {
		result := WinResult{Fan: 24, Patterns: []FanPattern{{Name: "七对", Fan: 24}}}
		if selfDraw {
			result.Fan++
			result.Patterns = append(result.Patterns, FanPattern{Name: "自摸", Fan: 1})
		}
		result.CanWin = result.Fan >= ruleset.MinFan
		return result
	}
	neededMelds := 4 - len(melds)
	if (len(codes)-2)/3 != neededMelds || (len(codes)-2)%3 != 0 {
		return WinResult{Reason: "牌型还没有组成 4 副面子加 1 对将。"}
	}
	decompositions := decompose(codes, neededMelds)
	if len(decompositions) == 0 {
		return WinResult{Reason: "牌型还没有组成 4 副面子加 1 对将。"}
	}
	best := WinResult{}
	for _, meldCodes := range decompositions {
		result := scoreWin(codes, meldCodes, melds, selfDraw, seatWind, roundWind)
		if result.Fan > best.Fan {
			best = result
		}
	}
	if best.Fan < ruleset.MinFan {
		best.CanWin = false
		best.Reason = fmt.Sprintf("国标麻将至少 %d 番起胡，当前 %d 番。", ruleset.MinFan, best.Fan)
		return best
	}
	best.CanWin = true
	return best
}

type codedMeld struct {
	kind  MeldKind
	codes []string
}

func scoreWin(codes []string, concealedMelds []codedMeld, exposedMelds []Meld, selfDraw bool, seatWind Wind, roundWind Wind) WinResult {
	patterns := []FanPattern{}
	allMelds := append([]codedMeld{}, concealedMelds...)
	for _, meld := range exposedMelds {
		allMelds = append(allMelds, codedMeld{kind: meld.Kind, codes: tileCodes(meld.Tiles)})
	}
	allCodes := append([]string{}, codes...)
	for _, meld := range exposedMelds {
		allCodes = append(allCodes, tileCodes(meld.Tiles)...)
	}
	if isPureOneSuit(allCodes) {
		patterns = append(patterns, FanPattern{Name: "清一色", Fan: 24})
	} else if isHalfFlush(allCodes) {
		patterns = append(patterns, FanPattern{Name: "混一色", Fan: 6})
	}
	if everyMeld(allMelds, func(meld codedMeld) bool { return meld.kind == MeldPung || meld.kind == MeldKong }) {
		patterns = append(patterns, FanPattern{Name: "碰碰和", Fan: 6})
	}
	dragonPungs := 0
	for _, meld := range allMelds {
		if (meld.kind == MeldPung || meld.kind == MeldKong) && isDragonCode(meld.codes[0]) {
			dragonPungs++
		}
	}
	for range dragonPungs {
		patterns = append(patterns, FanPattern{Name: "箭刻", Fan: 2})
	}
	if hasPung(allMelds, windCode(seatWind)) {
		patterns = append(patterns, FanPattern{Name: "门风刻", Fan: 2})
	}
	if hasPung(allMelds, windCode(roundWind)) {
		patterns = append(patterns, FanPattern{Name: "圈风刻", Fan: 2})
	}
	if selfDraw {
		patterns = append(patterns, FanPattern{Name: "自摸", Fan: 1})
	}
	if !slices.ContainsFunc(allCodes, isHonorCode) {
		patterns = append(patterns, FanPattern{Name: "无字", Fan: 1})
	}
	fan := 0
	for _, pattern := range patterns {
		fan += pattern.Fan
	}
	return WinResult{Fan: fan, Patterns: patterns}
}

func decompose(codes []string, neededMelds int) [][]codedMeld {
	counts := countCodes(codes)
	results := [][]codedMeld{}
	for code, count := range counts {
		if count < 2 {
			continue
		}
		nextCounts := cloneCounts(counts)
		nextCounts[code] -= 2
		for _, melds := range findMelds(nextCounts, neededMelds) {
			results = append(results, melds)
		}
	}
	return results
}

func findMelds(counts map[string]int, remaining int) [][]codedMeld {
	if remaining == 0 {
		if totalCount(counts) == 0 {
			return [][]codedMeld{{}}
		}
		return nil
	}
	code := firstRemainingCode(counts)
	if code == "" {
		return nil
	}
	results := [][]codedMeld{}
	if counts[code] >= 3 {
		nextCounts := cloneCounts(counts)
		nextCounts[code] -= 3
		for _, melds := range findMelds(nextCounts, remaining-1) {
			results = append(results, append([]codedMeld{{kind: MeldPung, codes: []string{code, code, code}}}, melds...))
		}
	}
	chowCodes := chowCodes(code)
	if len(chowCodes) == 3 && counts[chowCodes[0]] > 0 && counts[chowCodes[1]] > 0 && counts[chowCodes[2]] > 0 {
		nextCounts := cloneCounts(counts)
		for _, chowCode := range chowCodes {
			nextCounts[chowCode]--
		}
		for _, melds := range findMelds(nextCounts, remaining-1) {
			results = append(results, append([]codedMeld{{kind: MeldChow, codes: chowCodes}}, melds...))
		}
	}
	return results
}

func isSevenPairs(codes []string) bool {
	if len(codes) != 14 {
		return false
	}
	pairs := 0
	for _, count := range countCodes(codes) {
		if count == 2 {
			pairs++
		}
		if count == 4 {
			pairs += 2
		}
	}
	return pairs == 7
}

func chooseAIDiscard(player *Player) Tile {
	counts := countCodes(tileCodes(player.Hand))
	best := player.Hand[0]
	bestScore := -99
	for _, tile := range player.Hand {
		score := 6 - counts[tile.Code]*2
		if isHonorCode(tile.Code) && counts[tile.Code] == 1 {
			score = 8
		}
		if score > bestScore {
			best = tile
			bestScore = score
		}
	}
	return best
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

func shuffle(tiles []Tile) []Tile {
	next := append([]Tile{}, tiles...)
	rand.Shuffle(len(next), func(i int, j int) {
		next[i], next[j] = next[j], next[i]
	})
	return next
}

func shiftTiles(tiles *[]Tile, count int) []Tile {
	drawn := append([]Tile{}, (*tiles)[:count]...)
	*tiles = (*tiles)[count:]
	return drawn
}

func sortTiles(tiles []Tile) []Tile {
	next := append([]Tile{}, tiles...)
	sort.Slice(next, func(i int, j int) bool {
		return codeOrder(next[i].Code) < codeOrder(next[j].Code)
	})
	return next
}

func filterTiles(tiles []Tile, keep func(Tile) bool) []Tile {
	result := []Tile{}
	for _, tile := range tiles {
		if keep(tile) {
			result = append(result, tile)
		}
	}
	return result
}

func firstTileByCode(tiles []Tile, code string) (Tile, bool) {
	for _, tile := range tiles {
		if tile.Code == code {
			return tile, true
		}
	}
	return Tile{}, false
}

func tileCodes(tiles []Tile) []string {
	codes := make([]string, 0, len(tiles))
	for _, tile := range tiles {
		codes = append(codes, tile.Code)
	}
	sort.Slice(codes, func(i int, j int) bool { return codeOrder(codes[i]) < codeOrder(codes[j]) })
	return codes
}

func countCodes(codes []string) map[string]int {
	counts := map[string]int{}
	for _, code := range codes {
		counts[code]++
	}
	return counts
}

func cloneCounts(counts map[string]int) map[string]int {
	next := map[string]int{}
	for code, count := range counts {
		next[code] = count
	}
	return next
}

func totalCount(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		if count > 0 {
			total += count
		}
	}
	return total
}

func firstRemainingCode(counts map[string]int) string {
	codes := []string{}
	for code, count := range counts {
		if count > 0 {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i int, j int) bool { return codeOrder(codes[i]) < codeOrder(codes[j]) })
	if len(codes) == 0 {
		return ""
	}
	return codes[0]
}

func chowCodes(code string) []string {
	if !isSuited(code) || len(code) < 2 {
		return nil
	}
	rank := int(code[1] - '0')
	if rank > 7 {
		return nil
	}
	return []string{fmt.Sprintf("%s%d", code[:1], rank), fmt.Sprintf("%s%d", code[:1], rank+1), fmt.Sprintf("%s%d", code[:1], rank+2)}
}

func isPureOneSuit(codes []string) bool {
	suit := ""
	for _, code := range codes {
		if isHonorCode(code) {
			return false
		}
		if suit == "" {
			suit = code[:1]
		}
		if suit != code[:1] {
			return false
		}
	}
	return suit != ""
}

func isHalfFlush(codes []string) bool {
	suit := ""
	hasHonor := false
	for _, code := range codes {
		if isHonorCode(code) {
			hasHonor = true
			continue
		}
		if suit == "" {
			suit = code[:1]
		}
		if suit != code[:1] {
			return false
		}
	}
	return suit != "" && hasHonor
}

func hasPung(melds []codedMeld, code string) bool {
	for _, meld := range melds {
		if (meld.kind == MeldPung || meld.kind == MeldKong) && meld.codes[0] == code {
			return true
		}
	}
	return false
}

func everyMeld(melds []codedMeld, keep func(codedMeld) bool) bool {
	if len(melds) == 0 {
		return false
	}
	for _, meld := range melds {
		if !keep(meld) {
			return false
		}
	}
	return true
}

func isSuited(code string) bool {
	return strings.HasPrefix(code, "m") || strings.HasPrefix(code, "p") || strings.HasPrefix(code, "s")
}

func isHonorCode(code string) bool {
	return strings.HasPrefix(code, "z")
}

func isDragonCode(code string) bool {
	return code == "z5" || code == "z6" || code == "z7"
}

func windCode(wind Wind) string {
	switch wind {
	case WindEast:
		return "z1"
	case WindSouth:
		return "z2"
	case WindWest:
		return "z3"
	case WindNorth:
		return "z4"
	default:
		return "z1"
	}
}

func dragonCode(dragon Dragon) string {
	switch dragon {
	case DragonRed:
		return "z5"
	case DragonGreen:
		return "z6"
	case DragonWhite:
		return "z7"
	default:
		return "z7"
	}
}

func codeOrder(code string) int {
	base := map[byte]int{'m': 0, 'p': 20, 's': 40, 'z': 60}[code[0]]
	return base + int(code[1]-'0')
}

func formatTile(tile Tile) string {
	if tile.Rank > 0 {
		switch tile.Kind {
		case TileCharacters:
			return fmt.Sprintf("%d万", tile.Rank)
		case TileDots:
			return fmt.Sprintf("%d筒", tile.Rank)
		case TileBamboo:
			return fmt.Sprintf("%d条", tile.Rank)
		}
	}
	switch tile.Code {
	case "z1":
		return "东"
	case "z2":
		return "南"
	case "z3":
		return "西"
	case "z4":
		return "北"
	case "z5":
		return "中"
	case "z6":
		return "发"
	default:
		return "白"
	}
}
