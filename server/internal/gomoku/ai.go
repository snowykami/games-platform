package gomoku

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sort"

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
	if len(room.Moves) == 0 {
		return BoardSize / 2, BoardSize / 2
	}

	board := room.Board
	opponent := oppositeStone(stone)
	if point, ok := findWinningPoint(board, stone); ok {
		return point.X, point.Y
	}
	if point, ok := findWinningPoint(board, opponent); ok {
		return point.X, point.Y
	}

	depth := 2
	limit := 10
	if level == aiplayer.LevelMaster {
		depth = 3
		limit = 16
	}
	candidates := gomokuCandidateMoves(board, stone, limit)
	if len(candidates) == 0 {
		return BoardSize / 2, BoardSize / 2
	}

	bestPoint := candidates[0].Point
	bestScore := math.MinInt
	for _, candidate := range candidates {
		next := board
		next[candidate.Y][candidate.X] = stone
		score := gomokuWinScore
		if !boardHasFive(next, candidate.X, candidate.Y, stone) {
			score = -gomokuNegamax(next, opponent, depth-1, math.MinInt/2, math.MaxInt/2)
		}
		score += candidate.Score / 64
		if score > bestScore {
			bestPoint = candidate.Point
			bestScore = score
		}
	}
	return bestPoint.X, bestPoint.Y
}

func findWinningMove(room *Room, stone Stone) (int, int, bool) {
	if point, ok := findWinningPoint(room.Board, stone); ok {
		return point.X, point.Y, true
	}
	return 0, 0, false
}

const gomokuWinScore = 100_000_000

type scoredGomokuPoint struct {
	Point
	Score int
}

func gomokuNegamax(board [BoardSize][BoardSize]Stone, current Stone, depth int, alpha int, beta int) int {
	if depth <= 0 {
		return gomokuEvaluateBoard(board, current) - gomokuEvaluateBoard(board, oppositeStone(current))
	}

	candidates := gomokuCandidateMoves(board, current, 10)
	if len(candidates) == 0 {
		return 0
	}
	best := math.MinInt / 4
	for _, candidate := range candidates {
		next := board
		next[candidate.Y][candidate.X] = current
		score := gomokuWinScore - (3 - depth)
		if !boardHasFive(next, candidate.X, candidate.Y, current) {
			score = -gomokuNegamax(next, oppositeStone(current), depth-1, -beta, -alpha)
		}
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
	return best
}

func gomokuCandidateMoves(board [BoardSize][BoardSize]Stone, stone Stone, limit int) []scoredGomokuPoint {
	if boardStoneCount(board) == 0 {
		return []scoredGomokuPoint{{Point: Point{X: BoardSize / 2, Y: BoardSize / 2}, Score: 0}}
	}

	opponent := oppositeStone(stone)
	candidates := []scoredGomokuPoint{}
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if board[y][x] != "" || !hasNearbyStone(board, x, y, 2) {
				continue
			}
			offense := gomokuPointThreat(board, x, y, stone)
			defense := gomokuPointThreat(board, x, y, opponent)
			center := (BoardSize - abs(x-BoardSize/2) - abs(y-BoardSize/2))
			candidates = append(candidates, scoredGomokuPoint{
				Point: Point{X: x, Y: y},
				Score: offense + defense*9/10 + center,
			})
		}
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return distanceToCenter(candidates[i].Point) < distanceToCenter(candidates[j].Point)
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func gomokuPointThreat(board [BoardSize][BoardSize]Stone, x int, y int, stone Stone) int {
	score := 0
	for _, direction := range directions {
		leftCount, leftOpen := countBoardDirection(board, x, y, -direction.X, -direction.Y, stone)
		rightCount, rightOpen := countBoardDirection(board, x, y, direction.X, direction.Y, stone)
		score += gomokuThreatScore(1+leftCount+rightCount, boolToInt(leftOpen)+boolToInt(rightOpen))
	}
	return score
}

func gomokuEvaluateBoard(board [BoardSize][BoardSize]Stone, stone Stone) int {
	score := 0
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if board[y][x] != stone {
				continue
			}
			for _, direction := range directions {
				previousX := x - direction.X
				previousY := y - direction.Y
				if inBounds(previousX, previousY) && board[previousY][previousX] == stone {
					continue
				}
				count, openEnd := countLineFrom(board, x, y, direction.X, direction.Y, stone)
				backOpen := false
				if inBounds(previousX, previousY) && board[previousY][previousX] == "" {
					backOpen = true
				}
				score += gomokuThreatScore(count, boolToInt(openEnd)+boolToInt(backOpen))
			}
		}
	}
	return score
}

func gomokuThreatScore(count int, openEnds int) int {
	if count >= 5 {
		return gomokuWinScore
	}
	if count == 4 && openEnds == 2 {
		return 1_000_000
	}
	if count == 4 && openEnds == 1 {
		return 120_000
	}
	if count == 3 && openEnds == 2 {
		return 50_000
	}
	if count == 3 && openEnds == 1 {
		return 5_000
	}
	if count == 2 && openEnds == 2 {
		return 2_000
	}
	if count == 2 && openEnds == 1 {
		return 300
	}
	if count == 1 && openEnds == 2 {
		return 80
	}
	if count == 1 && openEnds == 1 {
		return 20
	}
	return 1
}

func findWinningPoint(board [BoardSize][BoardSize]Stone, stone Stone) (Point, bool) {
	candidates := gomokuCandidateMoves(board, stone, BoardSize*BoardSize)
	for _, candidate := range candidates {
		next := board
		next[candidate.Y][candidate.X] = stone
		if boardHasFive(next, candidate.X, candidate.Y, stone) {
			return candidate.Point, true
		}
	}
	return Point{}, false
}

func boardHasFive(board [BoardSize][BoardSize]Stone, x int, y int, stone Stone) bool {
	for _, direction := range directions {
		left, _ := countBoardDirection(board, x, y, -direction.X, -direction.Y, stone)
		right, _ := countBoardDirection(board, x, y, direction.X, direction.Y, stone)
		if 1+left+right >= 5 {
			return true
		}
	}
	return false
}

func countLineFrom(board [BoardSize][BoardSize]Stone, x int, y int, dx int, dy int, stone Stone) (int, bool) {
	count := 0
	nextX := x
	nextY := y
	for inBounds(nextX, nextY) && board[nextY][nextX] == stone {
		count++
		nextX += dx
		nextY += dy
	}
	return count, inBounds(nextX, nextY) && board[nextY][nextX] == ""
}

func countBoardDirection(board [BoardSize][BoardSize]Stone, x int, y int, dx int, dy int, stone Stone) (int, bool) {
	count := 0
	for step := 1; step < 5; step++ {
		nextX := x + dx*step
		nextY := y + dy*step
		if !inBounds(nextX, nextY) {
			return count, false
		}
		if board[nextY][nextX] == "" {
			return count, true
		}
		if board[nextY][nextX] != stone {
			return count, false
		}
		count++
	}
	return count, false
}

func hasNearbyStone(board [BoardSize][BoardSize]Stone, x int, y int, radius int) bool {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nextX := x + dx
			nextY := y + dy
			if inBounds(nextX, nextY) && board[nextY][nextX] != "" {
				return true
			}
		}
	}
	return false
}

func boardStoneCount(board [BoardSize][BoardSize]Stone) int {
	count := 0
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if board[y][x] != "" {
				count++
			}
		}
	}
	return count
}

func oppositeStone(stone Stone) Stone {
	if stone == StoneBlack {
		return StoneWhite
	}
	return StoneBlack
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func distanceToCenter(point Point) int {
	return abs(point.X-BoardSize/2) + abs(point.Y-BoardSize/2)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
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
