package mahjong

import (
	"errors"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func (m *Manager) chooseAIDiscard(room *Room, player *Player) Tile {
	level := aiplayer.NormalizeLevel("", false)
	if player.AI != nil {
		level = aiplayer.NormalizeLevel(player.AI.Level, m.aiProvider != nil && m.aiProvider.Enabled())
	}
	if level == aiplayer.LevelLLM {
		if tile, err := m.decideDiscardWithLLM(room, player); err == nil {
			return tile
		}
	}
	return chooseAIDiscard(player)
}

func (m *Manager) decideDiscardWithLLM(room *Room, player *Player) (Tile, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return Tile{}, errors.New("llm_not_configured")
	}
	actions := make([]aiplayer.LegalAction, 0, len(player.Hand))
	for _, tile := range player.Hand {
		actions = append(actions, aiplayer.LegalAction{
			ID:          "discard:" + tile.ID,
			Label:       "打出 " + formatTile(tile),
			Description: "从手牌中打出这张牌。",
		})
	}
	decision, err := m.decideWithAIAgent(room, player, gameactor.AgentRequiredAction, map[string]any{
		"wind":         player.Wind,
		"roundWind":    room.RoundWind,
		"hand":         append([]Tile{}, player.Hand...),
		"melds":        append([]Meld{}, player.Melds...),
		"wallCount":    len(room.Wall),
		"discards":     publicDiscards(room),
		"recentSpeech": recentSpeeches(room),
		"speechGuide":  "麻将发言自然短句，不要透露完整手牌；可以简单说牌型还差点或先安全打。",
	}, actions)
	if err != nil {
		return Tile{}, err
	}
	for _, tile := range player.Hand {
		if decision.ActionID == "discard:"+tile.ID {
			if strings.TrimSpace(decision.Speech) != "" {
				recordSpeech(room, player, decision.Speech)
			}
			return tile, nil
		}
	}
	return Tile{}, errors.New("llm_illegal_action")
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
		RequestPrefix: "mahjong",
		SessionID:     "mahjong:" + room.ID + ":" + player.ID,
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

func publicDiscards(room *Room) []map[string]any {
	discards := []map[string]any{}
	for _, player := range room.Players {
		for _, tile := range player.Discards {
			discards = append(discards, map[string]any{
				"playerId": player.ID,
				"tile":     tile,
			})
		}
	}
	return discards
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
