package gomoku

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

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
	decision, err := m.decideWithAIAgent(room, player, gameactor.AgentRequiredAction, map[string]any{
		"stone":        player.Stone,
		"boardSize":    BoardSize,
		"moves":        room.Moves,
		"recentSpeech": recentSpeeches(room),
	}, actions)
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
		RequestPrefix: "gomoku",
		SessionID:     "gomoku:" + room.ID + ":" + player.ID,
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
