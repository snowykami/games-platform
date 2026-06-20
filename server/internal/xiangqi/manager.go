package xiangqi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/roommeta"
)

const (
	minPlayers = 2
	maxPlayers = 2
)

var slidingDirections = []Position{
	{X: 1, Y: 0},
	{X: -1, Y: 0},
	{X: 0, Y: 1},
	{X: 0, Y: -1},
}

type Manager struct {
	mu         sync.Mutex
	rooms      map[string]*Room
	aiProvider aiplayer.Provider
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{rooms: map[string]*Room{}, aiProvider: aiProvider}
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
		Game:        "xiangqi",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID + ":speech",
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State: map[string]any{
			"phase":        room.Phase,
			"side":         player.Side,
			"check":        room.CheckSide,
			"recentMoves":  recentXiangqiMoves(room),
			"recentSpeech": recentSpeeches(room),
			"speechGuide":  "像象棋桌上的自然短句，可以评价局势或回应别人，不要长篇复盘。",
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
		return PublicRoom{}, errors.New("need_two_players")
	}

	resetGame(room)
	room.Phase = PhasePlaying
	room.CurrentPlayerIndex = 0
	room.Players[0].Side = SideRed
	room.Players[1].Side = SideBlack
	room.Log = append(room.Log, createLog("象棋开始，红方先行。"))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Move(roomID string, actorID string, pieceID string, to Position) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !insideBoard(to) {
		return PublicRoom{}, errors.New("invalid_position")
	}

	piece, ok := findPiece(room.Pieces, pieceID)
	if !ok {
		return PublicRoom{}, errors.New("piece_not_found")
	}
	if piece.Side != player.Side {
		return PublicRoom{}, errors.New("piece_wrong_side")
	}
	if !containsPosition(legalMoves(room.Pieces, piece), to) {
		return PublicRoom{}, errors.New("xiangqi_illegal_move")
	}

	applyPlayerMove(room, player, piece, to)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
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
	slog.Info("xiangqi ai turn started",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
	)

	piece, to, ok, speech := m.chooseAIPosition(room, player)
	if !ok {
		room.Phase = PhaseFinished
		room.WinnerID = opponentPlayer(room, player.Side).ID
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 无合法走法。", player.Name)))
		return publicRoom(room, ""), false, nil
	}

	applyPlayerMove(room, player, piece, to)
	recordSpeech(room, player, speech)
	room.UpdatedAt = time.Now().UTC()
	shouldContinue := room.Phase == PhasePlaying && room.Players[room.CurrentPlayerIndex].IsAI
	slog.Info("xiangqi ai turn completed",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
		"piece", piece.ID,
		"toX", to.X,
		"toY", to.Y,
		"duration", time.Since(startedAt),
		"continue", shouldContinue,
	)
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
	room.Pieces = initialPieces()
	room.Moves = nil
	room.WinnerID = ""
	room.CheckSide = ""
	room.RecentActions = nil
	for _, player := range room.Players {
		player.Side = ""
	}
}

func applyPlayerMove(room *Room, player *Player, piece Piece, to Position) {
	from := Position{X: piece.X, Y: piece.Y}
	captured, hasCaptured := pieceAt(room.Pieces, to)
	nextPieces := applyMove(room.Pieces, piece, to)
	opponent := oppositeSide(piece.Side)
	check := sideInCheck(nextPieces, opponent)
	checkmate := !hasAnyLegalMove(nextPieces, opponent)
	move := Move{
		ID:         "mov_" + randomToken(8),
		PieceID:    piece.ID,
		PieceType:  piece.Type,
		Side:       piece.Side,
		From:       from,
		To:         to,
		Check:      check,
		Checkmate:  checkmate,
		PlayerID:   player.ID,
		PlayerName: player.Name,
		PlayedAt:   time.Now().UTC(),
	}
	if hasCaptured {
		move.Captured = &captured
	}

	room.Pieces = nextPieces
	room.Moves = append(room.Moves, move)
	room.CheckSide = ""
	if check {
		room.CheckSide = opponent
	}

	message := formatMoveMessage(player, move)
	room.Log = append(room.Log, createLog(message))
	actionType := ActionMove
	if hasCaptured {
		actionType = ActionCapture
	}
	if check {
		actionType = ActionCheck
	}
	if hasCaptured && captured.Type == PieceGeneral || checkmate {
		actionType = ActionCheckmate
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		message = fmt.Sprintf("%s 获胜。", player.Name)
		room.Log = append(room.Log, createLog(message))
	}
	recordAction(room, PublicAction{
		Type:      actionType,
		ActorID:   player.ID,
		ActorName: player.Name,
		Move:      &move,
		Message:   message,
	})

	if room.Phase == PhasePlaying {
		room.CurrentPlayerIndex = nextPlayerIndex(room, player.Side)
	}
}

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
		Pieces:          append([]Piece{}, room.Pieces...),
		Moves:           append([]Move{}, room.Moves...),
		CurrentPlayerID: currentPlayerID,
		WinnerID:        room.WinnerID,
		CheckSide:       room.CheckSide,
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

func legalMoves(pieces []Piece, piece Piece) []Position {
	moves := pseudoMoves(pieces, piece)
	legal := []Position{}
	for _, move := range moves {
		target, occupied := pieceAt(pieces, move)
		if occupied && target.Side == piece.Side {
			continue
		}
		if !sideInCheck(applyMove(pieces, piece, move), piece.Side) {
			legal = append(legal, move)
		}
	}
	return legal
}

func allLegalMoves(pieces []Piece, side Side) []struct {
	piece Piece
	to    Position
} {
	moves := []struct {
		piece Piece
		to    Position
	}{}
	for _, piece := range pieces {
		if piece.Side != side {
			continue
		}
		for _, to := range legalMoves(pieces, piece) {
			moves = append(moves, struct {
				piece Piece
				to    Position
			}{piece: piece, to: to})
		}
	}
	return moves
}

func pseudoMoves(pieces []Piece, piece Piece) []Position {
	switch piece.Type {
	case PieceGeneral:
		return generalMoves(pieces, piece)
	case PieceAdvisor:
		return advisorMoves(piece)
	case PieceElephant:
		return elephantMoves(pieces, piece)
	case PieceHorse:
		return horseMoves(pieces, piece)
	case PieceRook:
		return slidingMoves(pieces, piece, false)
	case PieceCannon:
		return slidingMoves(pieces, piece, true)
	default:
		return soldierMoves(piece)
	}
}

func generalMoves(pieces []Piece, piece Piece) []Position {
	moves := filterPositions([]Position{
		{X: piece.X + 1, Y: piece.Y},
		{X: piece.X - 1, Y: piece.Y},
		{X: piece.X, Y: piece.Y + 1},
		{X: piece.X, Y: piece.Y - 1},
	}, func(position Position) bool {
		return insidePalace(position, piece.Side)
	})

	for _, other := range pieces {
		if other.Side != piece.Side && other.Type == PieceGeneral && other.X == piece.X && piecesBetween(pieces, Position{X: piece.X, Y: piece.Y}, Position{X: other.X, Y: other.Y}) == 0 {
			moves = append(moves, Position{X: other.X, Y: other.Y})
		}
	}
	return moves
}

func advisorMoves(piece Piece) []Position {
	return filterPositions([]Position{
		{X: piece.X + 1, Y: piece.Y + 1},
		{X: piece.X + 1, Y: piece.Y - 1},
		{X: piece.X - 1, Y: piece.Y + 1},
		{X: piece.X - 1, Y: piece.Y - 1},
	}, func(position Position) bool {
		return insidePalace(position, piece.Side)
	})
}

func elephantMoves(pieces []Piece, piece Piece) []Position {
	return filterPositions([]Position{
		{X: piece.X + 2, Y: piece.Y + 2},
		{X: piece.X + 2, Y: piece.Y - 2},
		{X: piece.X - 2, Y: piece.Y + 2},
		{X: piece.X - 2, Y: piece.Y - 2},
	}, func(position Position) bool {
		eye := Position{X: piece.X + (position.X-piece.X)/2, Y: piece.Y + (position.Y-piece.Y)/2}
		_, blocked := pieceAt(pieces, eye)
		return insideBoard(position) && ownRiverSide(position, piece.Side) && !blocked
	})
}

func horseMoves(pieces []Piece, piece Piece) []Position {
	offsets := []Position{{X: 1, Y: 2}, {X: 2, Y: 1}, {X: -1, Y: 2}, {X: -2, Y: 1}, {X: 1, Y: -2}, {X: 2, Y: -1}, {X: -1, Y: -2}, {X: -2, Y: -1}}
	moves := []Position{}
	for _, offset := range offsets {
		leg := Position{X: piece.X, Y: piece.Y + offset.Y/2}
		if abs(offset.X) == 2 {
			leg = Position{X: piece.X + offset.X/2, Y: piece.Y}
		}
		position := Position{X: piece.X + offset.X, Y: piece.Y + offset.Y}
		if _, blocked := pieceAt(pieces, leg); insideBoard(position) && !blocked {
			moves = append(moves, position)
		}
	}
	return moves
}

func slidingMoves(pieces []Piece, piece Piece, cannon bool) []Position {
	moves := []Position{}
	for _, direction := range slidingDirections {
		position := Position{X: piece.X + direction.X, Y: piece.Y + direction.Y}
		screenFound := false
		for insideBoard(position) {
			_, occupied := pieceAt(pieces, position)
			if !cannon {
				moves = append(moves, position)
				if occupied {
					break
				}
			} else if !screenFound {
				if occupied {
					screenFound = true
				} else {
					moves = append(moves, position)
				}
			} else if occupied {
				moves = append(moves, position)
				break
			}
			position = Position{X: position.X + direction.X, Y: position.Y + direction.Y}
		}
	}
	return moves
}

func soldierMoves(piece Piece) []Position {
	forward := 1
	if piece.Side == SideRed {
		forward = -1
	}
	moves := []Position{{X: piece.X, Y: piece.Y + forward}}
	if crossedRiver(piece) {
		moves = append(moves, Position{X: piece.X + 1, Y: piece.Y}, Position{X: piece.X - 1, Y: piece.Y})
	}
	return filterPositions(moves, insideBoard)
}

func sideInCheck(pieces []Piece, side Side) bool {
	general, ok := findGeneral(pieces, side)
	if !ok {
		return true
	}
	generalPosition := Position{X: general.X, Y: general.Y}
	for _, piece := range pieces {
		if piece.Side == side {
			continue
		}
		if containsPosition(pseudoMoves(pieces, piece), generalPosition) {
			return true
		}
	}
	return false
}

func hasAnyLegalMove(pieces []Piece, side Side) bool {
	return len(allLegalMoves(pieces, side)) > 0
}

func (m *Manager) chooseAIPosition(room *Room, player *Player) (Piece, Position, bool, string) {
	level := aiplayer.LevelNormal
	if player.AI != nil {
		level = aiplayer.NormalizeLevel(player.AI.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	}
	if level == aiplayer.LevelBeginner {
		moves := allLegalMoves(room.Pieces, player.Side)
		if len(moves) == 0 {
			return Piece{}, Position{}, false, ""
		}
		move := moves[rand.IntN(len(moves))]
		return move.piece, move.to, true, ""
	}
	if level == aiplayer.LevelLLM {
		if piece, to, ok, speech, err := m.decideXiangqiWithLLM(room, player); ok {
			return piece, to, true, speech
		} else if err != nil {
			slog.Warn("xiangqi llm decision failed, falling back",
				"room", room.ID,
				"player", player.ID,
				"playerName", player.Name,
				"error", err,
			)
		}
	}
	piece, to, ok := chooseAIMove(room, player.Side, level)
	return piece, to, ok, ""
}

func (m *Manager) decideXiangqiWithLLM(room *Room, player *Player) (Piece, Position, bool, string, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return Piece{}, Position{}, false, "", errors.New("llm_not_configured")
	}

	moves := allLegalMoves(room.Pieces, player.Side)
	if len(moves) == 0 {
		return Piece{}, Position{}, false, "", nil
	}

	actions := make([]aiplayer.LegalAction, 0, len(moves))
	for _, move := range moves {
		id := xiangqiActionID(move.piece, move.to)
		actions = append(actions, aiplayer.LegalAction{
			ID:          id,
			Label:       fmt.Sprintf("%s 到 %s", formatPiece(move.piece), formatPosition(move.to)),
			Description: xiangqiActionDescription(room, move.piece, move.to),
		})
	}

	personality := ""
	speechStyle := ""
	if player.AI != nil {
		personality = player.AI.Personality
		speechStyle = player.AI.SpeechStyle
	}
	ctx, cancel := context.WithTimeout(context.Background(), aiplayer.DecisionTimeout)
	defer cancel()
	decision, err := m.aiProvider.Decide(ctx, aiplayer.DecisionInput{
		Game:        "xiangqi",
		Level:       aiplayer.LevelLLM,
		SessionID:   player.ID,
		PlayerName:  player.Name,
		Personality: personality,
		SpeechStyle: speechStyle,
		State: map[string]any{
			"side":         player.Side,
			"pieces":       room.Pieces,
			"moves":        recentXiangqiMoves(room),
			"check":        room.CheckSide,
			"recentSpeech": recentSpeeches(room),
		},
		Actions: actions,
	})
	if err != nil {
		return Piece{}, Position{}, false, "", err
	}

	for _, move := range moves {
		if xiangqiActionID(move.piece, move.to) == decision.ActionID {
			return move.piece, move.to, true, decision.Speech, nil
		}
	}
	return Piece{}, Position{}, false, "", errors.New("llm_illegal_action")
}

func chooseAIMove(room *Room, side Side, level aiplayer.Level) (Piece, Position, bool) {
	moves := allLegalMoves(room.Pieces, side)
	if len(moves) == 0 {
		return Piece{}, Position{}, false
	}
	bestIndex := 0
	bestScore := math.Inf(-1)
	for index, move := range moves {
		score := scoreMove(room.Pieces, move.piece, move.to)
		if level == aiplayer.LevelMaster {
			score += mobilityScore(room.Pieces, move.piece, move.to)
		}
		if score > bestScore {
			bestIndex = index
			bestScore = score
		}
	}
	return moves[bestIndex].piece, moves[bestIndex].to, true
}

func mobilityScore(pieces []Piece, piece Piece, to Position) float64 {
	nextPieces := applyMove(pieces, piece, to)
	score := float64(len(allLegalMoves(nextPieces, piece.Side))) * 0.08
	score -= float64(len(allLegalMoves(nextPieces, oppositeSide(piece.Side)))) * 0.05
	return score
}

func xiangqiActionID(piece Piece, to Position) string {
	return fmt.Sprintf("move:%s:%d:%d", piece.ID, to.X, to.Y)
}

func xiangqiActionDescription(room *Room, piece Piece, to Position) string {
	description := "普通移动"
	if captured, ok := pieceAt(room.Pieces, to); ok {
		description = fmt.Sprintf("吃掉%s", formatPiece(captured))
	}
	nextPieces := applyMove(room.Pieces, piece, to)
	opponent := oppositeSide(piece.Side)
	if sideInCheck(nextPieces, opponent) {
		description += "，形成将军"
	}
	if !hasAnyLegalMove(nextPieces, opponent) {
		description += "，可将死"
	}
	return description
}

func recentXiangqiMoves(room *Room) []Move {
	if len(room.Moves) <= 8 {
		return append([]Move{}, room.Moves...)
	}
	return append([]Move{}, room.Moves[len(room.Moves)-8:]...)
}

func scoreMove(pieces []Piece, piece Piece, to Position) float64 {
	score := rand.Float64() * 0.1
	if captured, ok := pieceAt(pieces, to); ok {
		score += pieceValue(captured.Type) * 10
		if captured.Type == PieceGeneral {
			score += 10000
		}
	}
	nextPieces := applyMove(pieces, piece, to)
	opponent := oppositeSide(piece.Side)
	if sideInCheck(nextPieces, opponent) {
		score += 35
	}
	if !hasAnyLegalMove(nextPieces, opponent) {
		score += 5000
	}
	score += float64(4-abs(to.X-4)) * 0.4
	if piece.Side == SideRed {
		score += float64(9-to.Y) * 0.12
	} else {
		score += float64(to.Y) * 0.12
	}
	return score
}

func initialPieces() []Piece {
	pieces := []Piece{}
	pieces = appendBackRank(pieces, SideBlack, 0)
	pieces = appendBackRank(pieces, SideRed, 9)
	pieces = appendPieces(pieces, SideBlack, PieceCannon, 2, []int{1, 7})
	pieces = appendPieces(pieces, SideRed, PieceCannon, 7, []int{1, 7})
	pieces = appendPieces(pieces, SideBlack, PieceSoldier, 3, []int{0, 2, 4, 6, 8})
	pieces = appendPieces(pieces, SideRed, PieceSoldier, 6, []int{0, 2, 4, 6, 8})
	return pieces
}

func appendBackRank(pieces []Piece, side Side, y int) []Piece {
	order := []PieceType{PieceRook, PieceHorse, PieceElephant, PieceAdvisor, PieceGeneral, PieceAdvisor, PieceElephant, PieceHorse, PieceRook}
	for x, pieceType := range order {
		pieces = append(pieces, createPiece(side, pieceType, x, y))
	}
	return pieces
}

func appendPieces(pieces []Piece, side Side, pieceType PieceType, y int, files []int) []Piece {
	for _, x := range files {
		pieces = append(pieces, createPiece(side, pieceType, x, y))
	}
	return pieces
}

func createPiece(side Side, pieceType PieceType, x int, y int) Piece {
	return Piece{ID: fmt.Sprintf("%s-%s-%d-%d", side, pieceType, x, y), Side: side, Type: pieceType, X: x, Y: y}
}

func applyMove(pieces []Piece, piece Piece, to Position) []Piece {
	next := []Piece{}
	for _, item := range pieces {
		if item.ID == piece.ID || samePosition(Position{X: item.X, Y: item.Y}, to) {
			continue
		}
		next = append(next, item)
	}
	piece.X = to.X
	piece.Y = to.Y
	next = append(next, piece)
	return next
}

func pieceAt(pieces []Piece, position Position) (Piece, bool) {
	for _, piece := range pieces {
		if piece.X == position.X && piece.Y == position.Y {
			return piece, true
		}
	}
	return Piece{}, false
}

func findPiece(pieces []Piece, id string) (Piece, bool) {
	for _, piece := range pieces {
		if piece.ID == id {
			return piece, true
		}
	}
	return Piece{}, false
}

func findGeneral(pieces []Piece, side Side) (Piece, bool) {
	for _, piece := range pieces {
		if piece.Side == side && piece.Type == PieceGeneral {
			return piece, true
		}
	}
	return Piece{}, false
}

func filterPositions(positions []Position, keep func(Position) bool) []Position {
	filtered := []Position{}
	for _, position := range positions {
		if keep(position) {
			filtered = append(filtered, position)
		}
	}
	return filtered
}

func piecesBetween(pieces []Piece, from Position, to Position) int {
	stepX := sign(to.X - from.X)
	stepY := sign(to.Y - from.Y)
	position := Position{X: from.X + stepX, Y: from.Y + stepY}
	count := 0
	for !samePosition(position, to) {
		if _, ok := pieceAt(pieces, position); ok {
			count++
		}
		position = Position{X: position.X + stepX, Y: position.Y + stepY}
	}
	return count
}

func nextPlayerIndex(room *Room, side Side) int {
	nextSide := oppositeSide(side)
	for index, player := range room.Players {
		if player.Side == nextSide {
			return index
		}
	}
	return 0
}

func opponentPlayer(room *Room, side Side) *Player {
	nextSide := oppositeSide(side)
	for _, player := range room.Players {
		if player.Side == nextSide {
			return player
		}
	}
	return room.Players[0]
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
	return "XQ" + randomToken(5)
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range length {
		builder.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return builder.String()
}

func containsPosition(positions []Position, target Position) bool {
	for _, position := range positions {
		if samePosition(position, target) {
			return true
		}
	}
	return false
}

func insideBoard(position Position) bool {
	return position.X >= 0 && position.X < BoardWidth && position.Y >= 0 && position.Y < BoardHeight
}

func insidePalace(position Position, side Side) bool {
	return position.X >= 3 && position.X <= 5 && ((side == SideRed && position.Y >= 7 && position.Y <= 9) || (side == SideBlack && position.Y >= 0 && position.Y <= 2))
}

func ownRiverSide(position Position, side Side) bool {
	return side == SideRed && position.Y >= 5 || side == SideBlack && position.Y <= 4
}

func crossedRiver(piece Piece) bool {
	return piece.Side == SideRed && piece.Y <= 4 || piece.Side == SideBlack && piece.Y >= 5
}

func samePosition(first Position, second Position) bool {
	return first.X == second.X && first.Y == second.Y
}

func oppositeSide(side Side) Side {
	if side == SideRed {
		return SideBlack
	}
	return SideRed
}

func pieceValue(pieceType PieceType) float64 {
	values := map[PieceType]float64{
		PieceAdvisor:  2,
		PieceCannon:   4.5,
		PieceElephant: 2,
		PieceGeneral:  1000,
		PieceHorse:    4,
		PieceRook:     9,
		PieceSoldier:  1.5,
	}
	return values[pieceType]
}

func formatMoveMessage(player *Player, move Move) string {
	captureText := ""
	if move.Captured != nil {
		captureText = fmt.Sprintf("，吃掉%s", formatPiece(*move.Captured))
	}
	checkText := ""
	if move.Check {
		checkText = "，将军"
	}
	return fmt.Sprintf("%s %s 从 %s 到 %s%s%s。", player.Name, formatPiece(Piece{Side: move.Side, Type: move.PieceType}), formatPosition(move.From), formatPosition(move.To), captureText, checkText)
}

func formatPiece(piece Piece) string {
	labels := map[Side]map[PieceType]string{
		SideRed:   {PieceGeneral: "帥", PieceAdvisor: "仕", PieceElephant: "相", PieceHorse: "傌", PieceRook: "俥", PieceCannon: "炮", PieceSoldier: "兵"},
		SideBlack: {PieceGeneral: "將", PieceAdvisor: "士", PieceElephant: "象", PieceHorse: "馬", PieceRook: "車", PieceCannon: "砲", PieceSoldier: "卒"},
	}
	return labels[piece.Side][piece.Type]
}

func formatPosition(position Position) string {
	return fmt.Sprintf("%d路%d线", position.X+1, position.Y+1)
}

func sign(value int) int {
	if value > 0 {
		return 1
	}
	if value < 0 {
		return -1
	}
	return 0
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
