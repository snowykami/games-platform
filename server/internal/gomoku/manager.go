package gomoku

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

const (
	minPlayers = 2
	maxPlayers = 2
)

var directions = []Point{
	{X: 1, Y: 0},
	{X: 0, Y: 1},
	{X: 1, Y: 1},
	{X: 1, Y: -1},
}

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
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	room.Players = append(room.Players, createHumanPlayer(user, "host"))
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 创建了房间。", user.DisplayName)))
	m.rooms[room.ID] = room

	return publicRoom(room)
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
	return publicRoom(room), nil
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

	return publicRoom(room), nil
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
	return publicRoom(room), nil
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
		return publicRoom(room), nil
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
	return publicRoom(room), nil
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
		return PublicRoom{}, errors.New("need_two_players")
	}

	resetBoard(room)
	room.Phase = PhasePlaying
	room.CurrentPlayerIndex = 0
	room.Players[0].Stone = StoneBlack
	room.Players[1].Stone = StoneWhite
	room.Log = append(room.Log, createLog("五子棋开始，黑棋先行。"))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room), nil
}

func (m *Manager) Place(roomID string, actorID string, x int, y int) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !inBounds(x, y) {
		return PublicRoom{}, errors.New("invalid_position")
	}
	if room.Board[y][x] != "" {
		return PublicRoom{}, errors.New("position_occupied")
	}

	placeStone(room, player, x, y)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room), nil
}

func (m *Manager) RunNextAI(roomID string) (PublicRoom, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	startedAt := time.Now()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase != PhasePlaying || len(room.Players) == 0 {
		return publicRoom(room), false, nil
	}

	player := room.Players[room.CurrentPlayerIndex]
	if !player.IsAI {
		return publicRoom(room), false, nil
	}

	level := ""
	if player.AI != nil {
		level = player.AI.Level
	}
	slog.Info("gomoku ai turn started",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
	)

	x, y, speech := m.chooseAIPosition(room, player)
	if !inBounds(x, y) {
		return publicRoom(room), false, nil
	}
	placeStone(room, player, x, y)
	recordSpeech(room, player, speech)
	room.UpdatedAt = time.Now().UTC()

	shouldContinue := room.Phase == PhasePlaying && room.Players[room.CurrentPlayerIndex].IsAI
	slog.Info("gomoku ai turn completed",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
		"x", x,
		"y", y,
		"duration", time.Since(startedAt),
		"continue", shouldContinue,
	)
	return publicRoom(room), shouldContinue, nil
}

func (m *Manager) Public(roomID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	return publicRoom(room), nil
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

func resetBoard(room *Room) {
	room.Board = [BoardSize][BoardSize]Stone{}
	room.Moves = nil
	room.WinnerID = ""
	room.WinningLine = nil
	room.IsDraw = false
	room.RecentActions = nil
}

func placeStone(room *Room, player *Player, x int, y int) {
	room.Board[y][x] = player.Stone
	move := Move{
		X:          x,
		Y:          y,
		Stone:      player.Stone,
		PlayerID:   player.ID,
		PlayerName: player.Name,
		PlacedAt:   time.Now().UTC(),
	}
	room.Moves = append(room.Moves, move)

	message := fmt.Sprintf("%s 落子 %s。", player.Name, formatPoint(x, y))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{
		Type:      ActionPlace,
		ActorID:   player.ID,
		ActorName: player.Name,
		X:         x,
		Y:         y,
		Stone:     player.Stone,
		Message:   message,
	})

	if line := winningLine(room, x, y, player.Stone); len(line) >= 5 {
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		room.WinningLine = line
		winMessage := fmt.Sprintf("%s 五连获胜。", player.Name)
		room.Log = append(room.Log, createLog(winMessage))
		recordAction(room, PublicAction{
			Type:      ActionWin,
			ActorID:   player.ID,
			ActorName: player.Name,
			Stone:     player.Stone,
			Message:   winMessage,
		})
		return
	}

	if len(room.Moves) == BoardSize*BoardSize {
		room.Phase = PhaseFinished
		room.IsDraw = true
		drawMessage := "棋盘已满，平局。"
		room.Log = append(room.Log, createLog(drawMessage))
		recordAction(room, PublicAction{
			Type:      ActionDraw,
			ActorID:   player.ID,
			ActorName: player.Name,
			Message:   drawMessage,
		})
		return
	}

	room.CurrentPlayerIndex = (room.CurrentPlayerIndex + 1) % len(room.Players)
}

func publicRoom(room *Room) PublicRoom {
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

func winningLine(room *Room, x int, y int, stone Stone) []Point {
	for _, direction := range directions {
		line := []Point{{X: x, Y: y}}
		line = append(collectLine(room, x, y, direction.X, direction.Y, stone), line...)
		line = append(line, collectLine(room, x, y, -direction.X, -direction.Y, stone)...)
		if len(line) >= 5 {
			return line
		}
	}
	return nil
}

func collectLine(room *Room, x int, y int, dx int, dy int, stone Stone) []Point {
	line := []Point{}
	for step := 1; step < 5; step++ {
		nextX := x + dx*step
		nextY := y + dy*step
		if !inBounds(nextX, nextY) || room.Board[nextY][nextX] != stone {
			break
		}
		line = append(line, Point{X: nextX, Y: nextY})
	}
	return line
}

func (m *Manager) chooseAIPosition(room *Room, player *Player) (int, int, string) {
	level := aiplayer.NormalizeLevel(player.AI.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	if level == aiplayer.LevelBeginner {
		empty := emptyPoints(room)
		if len(empty) == 0 {
			return -1, -1, ""
		}
		point := empty[rand.IntN(len(empty))]
		return point.X, point.Y, ""
	}

	if level == aiplayer.LevelLLM {
		if x, y, speech, err := m.decideGomokuWithLLM(room, player); err == nil {
			return x, y, speech
		} else {
			slog.Warn("gomoku llm decision failed, falling back",
				"room", room.ID,
				"player", player.ID,
				"playerName", player.Name,
				"error", err,
			)
		}
	}

	x, y := chooseAIMove(room, player.Stone, level)
	return x, y, ""
}

func (m *Manager) decideGomokuWithLLM(room *Room, player *Player) (int, int, string, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return 0, 0, "", errors.New("llm_not_configured")
	}

	actions := gomokuLegalActions(room)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	decision, err := m.aiProvider.Decide(ctx, aiplayer.DecisionInput{
		Game:        "gomoku",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID,
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State: map[string]any{
			"stone":        player.Stone,
			"boardSize":    BoardSize,
			"moves":        room.Moves,
			"recentSpeech": recentSpeeches(room),
		},
		Actions: actions,
	})
	if err != nil {
		return 0, 0, "", err
	}

	for _, point := range emptyPoints(room) {
		if fmt.Sprintf("place:%d:%d", point.X, point.Y) == decision.ActionID {
			return point.X, point.Y, decision.Speech, nil
		}
	}
	return 0, 0, "", errors.New("llm_illegal_action")
}

func gomokuLegalActions(room *Room) []aiplayer.LegalAction {
	points := emptyPoints(room)
	actions := make([]aiplayer.LegalAction, 0, len(points))
	for _, point := range points {
		actions = append(actions, aiplayer.LegalAction{
			ID:    fmt.Sprintf("place:%d:%d", point.X, point.Y),
			Label: fmt.Sprintf("落子 %s", formatPoint(point.X, point.Y)),
		})
	}
	return actions
}

func emptyPoints(room *Room) []Point {
	points := []Point{}
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if room.Board[y][x] == "" {
				points = append(points, Point{X: x, Y: y})
			}
		}
	}
	return points
}

func chooseAIMove(room *Room, stone Stone, level aiplayer.Level) (int, int) {
	opponent := StoneBlack
	if stone == StoneBlack {
		opponent = StoneWhite
	}

	if x, y, ok := findWinningMove(room, stone); ok {
		return x, y
	}
	if x, y, ok := findWinningMove(room, opponent); ok {
		return x, y
	}
	if len(room.Moves) == 0 {
		return BoardSize / 2, BoardSize / 2
	}

	bestX, bestY, bestScore := -1, -1, -1
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if room.Board[y][x] != "" {
				continue
			}
			score := moveScore(room, x, y, stone) + moveScore(room, x, y, opponent)
			if level == aiplayer.LevelMaster {
				score += openEndsScore(room, x, y, stone) + openEndsScore(room, x, y, opponent)
			}
			if score > bestScore {
				bestX, bestY, bestScore = x, y, score
			}
		}
	}
	return bestX, bestY
}

func openEndsScore(room *Room, x int, y int, stone Stone) int {
	score := 0
	for _, direction := range directions {
		for _, sign := range []int{1, -1} {
			nextX := x + direction.X*sign
			nextY := y + direction.Y*sign
			if inBounds(nextX, nextY) && room.Board[nextY][nextX] == "" {
				score++
			}
		}
	}
	if stone != "" {
		score += 1
	}
	return score
}

func findWinningMove(room *Room, stone Stone) (int, int, bool) {
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if room.Board[y][x] != "" {
				continue
			}
			room.Board[y][x] = stone
			line := winningLine(room, x, y, stone)
			room.Board[y][x] = ""
			if len(line) >= 5 {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func moveScore(room *Room, x int, y int, stone Stone) int {
	score := 0
	for _, direction := range directions {
		score += countStones(room, x, y, direction.X, direction.Y, stone)
		score += countStones(room, x, y, -direction.X, -direction.Y, stone)
	}
	return score
}

func countStones(room *Room, x int, y int, dx int, dy int, stone Stone) int {
	count := 0
	for step := 1; step < 5; step++ {
		nextX := x + dx*step
		nextY := y + dy*step
		if !inBounds(nextX, nextY) || room.Board[nextY][nextX] != stone {
			break
		}
		count++
	}
	return count
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

func inBounds(x int, y int) bool {
	return x >= 0 && x < BoardSize && y >= 0 && y < BoardSize
}

func formatPoint(x int, y int) string {
	return fmt.Sprintf("%c%d", 'A'+rune(x), y+1)
}
