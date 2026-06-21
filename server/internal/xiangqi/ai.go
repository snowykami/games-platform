package xiangqi

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"

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
