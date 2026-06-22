package xiangqi

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sort"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

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

	decision, err := m.decideWithAIAgent(room, player, gameactor.AgentRequiredAction, map[string]any{
		"side":         player.Side,
		"pieces":       room.Pieces,
		"moves":        recentXiangqiMoves(room),
		"check":        room.CheckSide,
		"recentSpeech": recentSpeeches(room),
	}, actions)
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

func (m *Manager) decideWithAIAgent(room *Room, player *Player, eventType gameactor.AgentEventType, state map[string]any, actions []aiplayer.LegalAction) (aiplayer.Decision, error) {
	if m.aiController == nil || !m.aiController.Enabled() {
		return aiplayer.Decision{}, aiagent.ErrLLMNotConfigured
	}
	expectedPhase := room.Phase
	expectedActionSeq := room.ActionSeq
	expectedUpdatedAt := room.UpdatedAt
	personality := ""
	speechStyle := ""
	if player.AI != nil {
		personality = player.AI.Personality
		speechStyle = player.AI.SpeechStyle
	}
	return m.aiController.Decide(aiagent.DecisionRequest{
		RoomID:        room.ID,
		PlayerID:      player.ID,
		RequestPrefix: "xiangqi",
		SessionID:     "xiangqi:" + room.ID + ":" + player.ID,
		Phase:         string(room.Phase),
		Type:          eventType,
		Profile: aiagent.Profile{
			Name:        player.Name,
			Personality: personality,
			SpeechStyle: speechStyle,
		},
		State:   state,
		Actions: actions,
		Unlock:  m.mu.Unlock,
		Lock:    m.mu.Lock,
		Stale: func(_ aiplayer.Decision) error {
			if room.Phase != expectedPhase || room.ActionSeq != expectedActionSeq {
				return errors.New("ai_agent_decision_stale")
			}
			if eventType == gameactor.AgentOptionalSpeech && !room.UpdatedAt.Equal(expectedUpdatedAt) {
				return errors.New("ai_agent_speech_stale")
			}
			return nil
		},
	})
}

func (m *Manager) removeAIAgent(roomID string, playerID string) {
	if m.aiController != nil {
		m.aiController.Remove(roomID, playerID)
	}
}

func (m *Manager) removeRoomAgents(roomID string) {
	if m.aiController != nil {
		m.aiController.RemoveRoom(roomID)
	}
}

func chooseAIMove(room *Room, side Side, level aiplayer.Level) (Piece, Position, bool) {
	moves := allLegalMoves(room.Pieces, side)
	if len(moves) == 0 {
		return Piece{}, Position{}, false
	}
	depth := 2
	if level == aiplayer.LevelMaster {
		depth = 3
		if len(room.Pieces) <= 18 {
			depth = 4
		}
	}
	moves = orderedXiangqiMoves(room.Pieces, moves, side)
	search := newXiangqiSearch()
	bestMove := moves[0]
	bestScore := math.Inf(-1)
	alpha := math.Inf(-1)
	beta := math.Inf(1)
	for _, move := range moves {
		nextPieces := applyMove(room.Pieces, move.piece, move.to)
		score := -search.negamax(nextPieces, oppositeSide(side), depth-1, -beta, -alpha)
		if score > bestScore {
			bestMove = move
			bestScore = score
		}
		if score > alpha {
			alpha = score
		}
	}
	return bestMove.piece, bestMove.to, true
}

const xiangqiMateScore = 1_000_000

const (
	xiangqiTTExact = iota
	xiangqiTTLower
	xiangqiTTUpper
)

type xiangqiSearch struct {
	table map[string]xiangqiTTEntry
}

type xiangqiTTEntry struct {
	Depth int
	Score float64
	Flag  int
}

func newXiangqiSearch() *xiangqiSearch {
	return &xiangqiSearch{table: map[string]xiangqiTTEntry{}}
}

func (search *xiangqiSearch) negamax(pieces []Piece, side Side, depth int, alpha float64, beta float64) float64 {
	moves := allLegalMoves(pieces, side)
	if len(moves) == 0 {
		if sideInCheck(pieces, side) {
			return -xiangqiMateScore
		}
		return 0
	}
	alphaOriginal := alpha
	key := xiangqiPositionKey(pieces, side)
	if entry, ok := search.table[key]; ok && entry.Depth >= depth {
		switch entry.Flag {
		case xiangqiTTExact:
			return entry.Score
		case xiangqiTTLower:
			if entry.Score > alpha {
				alpha = entry.Score
			}
		case xiangqiTTUpper:
			if entry.Score < beta {
				beta = entry.Score
			}
		}
		if alpha >= beta {
			return entry.Score
		}
	}
	if depth <= 0 {
		return search.quiescence(pieces, side, alpha, beta, 4)
	}

	best := math.Inf(-1)
	for _, move := range orderedXiangqiMoves(pieces, moves, side) {
		nextPieces := applyMove(pieces, move.piece, move.to)
		score := -search.negamax(nextPieces, oppositeSide(side), depth-1, -beta, -alpha)
		if score > best {
			best = score
		}
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break
		}
	}
	search.store(key, depth, best, alphaOriginal, beta)
	return best
}

func (search *xiangqiSearch) quiescence(pieces []Piece, side Side, alpha float64, beta float64, depth int) float64 {
	standPat := evaluateXiangqiSide(pieces, side) - evaluateXiangqiSide(pieces, oppositeSide(side))
	if depth <= 0 {
		return standPat
	}
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	for _, move := range orderedXiangqiMoves(pieces, xiangqiCaptureMoves(pieces, side), side) {
		nextPieces := applyMove(pieces, move.piece, move.to)
		score := -search.quiescence(nextPieces, oppositeSide(side), -beta, -alpha, depth-1)
		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}
	return alpha
}

func (search *xiangqiSearch) store(key string, depth int, score float64, alpha float64, beta float64) {
	flag := xiangqiTTExact
	if score <= alpha {
		flag = xiangqiTTUpper
	} else if score >= beta {
		flag = xiangqiTTLower
	}
	search.table[key] = xiangqiTTEntry{Depth: depth, Score: score, Flag: flag}
}

func xiangqiCaptureMoves(pieces []Piece, side Side) []legalMove {
	moves := []legalMove{}
	for _, move := range allLegalMoves(pieces, side) {
		if captured, ok := pieceAt(pieces, move.to); ok && captured.Side != side {
			moves = append(moves, move)
		}
	}
	return moves
}

func xiangqiPositionKey(pieces []Piece, side Side) string {
	items := append([]Piece{}, pieces...)
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ID < items[j].ID
	})
	var builder strings.Builder
	builder.WriteString(string(side))
	for _, piece := range items {
		builder.WriteByte('|')
		builder.WriteString(piece.ID)
		builder.WriteByte('@')
		builder.WriteByte(byte('0' + piece.X))
		builder.WriteByte(',')
		builder.WriteByte(byte('0' + piece.Y))
	}
	return builder.String()
}

func orderedXiangqiMoves(pieces []Piece, moves []legalMove, side Side) []legalMove {
	ordered := append([]legalMove{}, moves...)
	sort.SliceStable(ordered, func(i int, j int) bool {
		return xiangqiMoveOrderScore(pieces, ordered[i], side) > xiangqiMoveOrderScore(pieces, ordered[j], side)
	})
	return ordered
}

func xiangqiMoveOrderScore(pieces []Piece, move legalMove, side Side) float64 {
	score := 0.0
	if captured, ok := pieceAt(pieces, move.to); ok {
		score += pieceValue(captured.Type)*100 - pieceValue(move.piece.Type)
		if captured.Type == PieceGeneral {
			score += xiangqiMateScore
		}
	}
	nextPieces := applyMove(pieces, move.piece, move.to)
	if sideInCheck(nextPieces, oppositeSide(side)) {
		score += 500
	}
	score += evaluatePiecePosition(move.piece, move.to) * 0.05
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
	score := rand.Float64() * 0.01
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

func evaluateXiangqiSide(pieces []Piece, side Side) float64 {
	score := 0.0
	for _, piece := range pieces {
		if piece.Side != side {
			continue
		}
		score += pieceValue(piece.Type) * 100
		score += evaluatePiecePosition(piece, Position{X: piece.X, Y: piece.Y})
		score += float64(len(pseudoMoves(pieces, piece))) * mobilityWeight(piece.Type)
	}
	if sideInCheck(pieces, oppositeSide(side)) {
		score += 35
	}
	if sideInCheck(pieces, side) {
		score -= 55
	}
	return score
}

func evaluatePiecePosition(piece Piece, position Position) float64 {
	center := float64(4 - abs(position.X-4))
	advance := float64(sideAdvance(piece.Side, position.Y))
	switch piece.Type {
	case PieceSoldier:
		score := advance * 10
		if soldierCrossedRiver(piece.Side, position.Y) {
			score += 80 + center*8
		}
		return score
	case PieceHorse:
		return center*8 + advance*2
	case PieceCannon:
		return center*7 + advance*1.5
	case PieceRook:
		return center*4 + advance*2
	case PieceGeneral:
		return -float64(abs(position.X-4))*8 - float64(abs(position.Y-generalHomeY(piece.Side)))*4
	case PieceAdvisor, PieceElephant:
		return 12
	default:
		return 0
	}
}

func mobilityWeight(pieceType PieceType) float64 {
	switch pieceType {
	case PieceRook:
		return 3
	case PieceCannon:
		return 2.5
	case PieceHorse:
		return 2.2
	case PieceSoldier:
		return 1
	default:
		return 0.6
	}
}

func sideAdvance(side Side, y int) int {
	if side == SideRed {
		return 9 - y
	}
	return y
}

func soldierCrossedRiver(side Side, y int) bool {
	return side == SideRed && y <= 4 || side == SideBlack && y >= 5
}

func generalHomeY(side Side) int {
	if side == SideRed {
		return 9
	}
	return 0
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
