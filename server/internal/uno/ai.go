package uno

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

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

	return m.decideWithAIAgent(room, player, gameactor.AgentRequiredAction, map[string]any{
		"activeColor":      room.ActiveColor,
		"direction":        room.Direction,
		"pendingDrawCount": room.PendingDrawCount,
		"topCard":          discardTopCard(room),
		"hand":             player.Hand,
		"opponents":        publicOpponentCounts(room, player.ID),
		"recentSpeech":     recentSpeeches(room),
		"speechGuide":      "UNO 发言像普通朋友局：短句、自然、可以吐槽牌不好或提醒颜色，不要中二台词，不要解释规则。",
	}, legalActions)
}

func (m *Manager) decideWithAIAgent(room *Room, player *Player, eventType gameactor.AgentEventType, state map[string]any, actions []aiplayer.LegalAction) (aiplayer.Decision, error) {
	if m.aiController == nil || !m.aiController.Enabled() {
		return aiplayer.Decision{}, aiagent.ErrLLMNotConfigured
	}
	expectedPhase := room.Phase
	expectedActionSeq := room.ActionSeq
	expectedUpdatedAt := room.UpdatedAt
	return m.aiController.Decide(aiagent.DecisionRequest{
		RoomID:        room.ID,
		PlayerID:      player.ID,
		RequestPrefix: "uno",
		SessionID:     "uno:" + room.ID + ":" + player.ID,
		Phase:         string(room.Phase),
		Type:          eventType,
		Profile: aiagent.Profile{
			Name:        player.Name,
			Personality: player.AI.Personality,
			SpeechStyle: player.AI.SpeechStyle,
		},
		State:   state,
		Actions: actions,
		Unlock:  room.mu.Unlock,
		Lock:    room.mu.Lock,
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
