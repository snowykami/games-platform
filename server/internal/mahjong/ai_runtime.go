package mahjong

import (
	"fmt"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func (m *Manager) RunAIOptionalSpeech(roomID string) (PublicRoom, bool, error) {
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
	updatedAt := room.UpdatedAt
	playerID := player.ID
	decision, err := m.decideWithAIAgent(room, player, gameactor.AgentOptionalSpeech, map[string]any{
		"phase":        room.Phase,
		"wind":         player.Wind,
		"handCount":    len(player.Hand),
		"wallCount":    len(room.Wall),
		"recentSpeech": recentSpeeches(room),
		"speechGuide":  "像麻将桌上的自然短句，可以闲聊或轻微评价牌局，不要透露隐藏手牌。",
	}, speechActions())
	if err != nil {
		m.mu.Unlock()
		return PublicRoom{}, false, err
	}
	if decision.ActionID != "speak" || strings.TrimSpace(decision.Speech) == "" {
		m.mu.Unlock()
		return PublicRoom{}, false, nil
	}

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

func (m *Manager) RunAIAction(roomID string) (PublicRoom, bool, error) {
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

	tile := m.chooseAIDiscard(room, player)
	discardTile(room, player, tile.ID)
	room.UpdatedAt = time.Now().UTC()
	shouldContinue := room.Phase == PhasePlaying && room.Players[room.CurrentPlayerIndex].IsAI
	return publicRoom(room, ""), shouldContinue, nil
}

func hasPendingAIRequiredAction(room *Room) bool {
	return room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
}
