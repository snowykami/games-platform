package uno

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/gameactor"
)

func (m *Manager) RunAIOptionalSpeech(roomID string) (PublicRoom, bool, error) {
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return PublicRoom{}, false, nil
	}
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || len(room.Speeches) == 0 {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	if hasPendingAIRequiredAction(room) {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	lastSpeech := room.Speeches[len(room.Speeches)-1]
	if lastSpeech.ID == room.LastAISpeechSourceID {
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	player := nextAISpeechPlayer(room, lastSpeech.PlayerID)
	if player == nil || player.AI == nil {
		room.LastAISpeechSourceID = lastSpeech.ID
		view := publicRoom(room, "")
		room.mu.Unlock()
		return view, false, nil
	}
	room.LastAISpeechSourceID = lastSpeech.ID
	updatedAt := room.UpdatedAt
	playerID := player.ID
	decision, err := m.decideWithAIAgent(room, player, gameactor.AgentOptionalSpeech, map[string]any{
		"phase":        room.Phase,
		"activeColor":  room.ActiveColor,
		"topCard":      discardTopCard(room),
		"handCount":    len(player.Hand),
		"recentSpeech": recentSpeeches(room),
		"speechGuide":  "像 UNO 朋友局里自然接一句，短句即可；如果没必要回应就跳过。",
	}, speechActions())
	if err != nil {
		room.mu.Unlock()
		return PublicRoom{}, false, err
	}
	if decision.ActionID != "speak" || strings.TrimSpace(decision.Speech) == "" {
		room.mu.Unlock()
		return PublicRoom{}, false, nil
	}

	defer room.mu.Unlock()
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
	startedAt := time.Now()

	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	defer room.mu.Unlock()
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
	slog.Info("uno ai turn started",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
	)

	action, speech := m.chooseAIAction(room, player)
	if action.Kind == "draw" {
		drawCount := 1
		if room.PendingDrawCount > 0 {
			drawCount = room.PendingDrawCount
			room.PendingDrawCount = 0
			room.PendingDrawKind = ""
		}
		drawn := drawCards(room, drawCount)
		player.Hand = append(player.Hand, drawn...)
		player.HandCount = len(player.Hand)
		message := fmt.Sprintf("%s 摸了 %d 张牌。", player.Name, len(drawn))
		room.Log = append(room.Log, createLog(message))
		recordDrawAction(room, player, player, len(drawn), message)
		advanceTurn(room)
	} else {
		playCard(room, player, action.CardIndex, action.Color)
	}
	recordSpeech(room, player, speech)

	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	shouldContinue := room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
	slog.Info("uno ai turn completed",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"level", level,
		"action", action.Kind,
		"duration", time.Since(startedAt),
		"continue", shouldContinue,
	)
	return publicRoom(room, ""), shouldContinue, nil
}

func hasPendingAIRequiredAction(room *Room) bool {
	return room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI
}

func (m *Manager) Tick(now time.Time) TickResult {
	now = now.UTC()
	m.mu.RLock()
	roomIDs := make([]string, 0, len(m.rooms))
	for roomID := range m.rooms {
		roomIDs = append(roomIDs, roomID)
	}
	m.mu.RUnlock()

	result := TickResult{}
	for _, roomID := range roomIDs {
		err := m.RunRoomCommand(context.Background(), roomID, gameactor.EventTurnDeadlineReached, gameactor.LaneRule, func() error {
			room, err := m.room(roomID)
			if err != nil {
				return nil
			}
			room.mu.Lock()
			defer room.mu.Unlock()
			destroy := shouldDestroyOfflineRoom(room, now)
			if destroy {
				result.DestroyedRoomIDs = append(result.DestroyedRoomIDs, roomID)
				return nil
			}

			if room.Phase == PhasePlaying && len(room.Players) > 0 {
				result.BroadcastRoomIDs = append(result.BroadcastRoomIDs, roomID)
				if applyTimeoutTurn(room, now) {
					result.ScheduleAIRoomIDs = append(result.ScheduleAIRoomIDs, roomID)
				}
				if room.Phase == PhasePlaying && len(room.Players) > 0 && room.Players[room.CurrentPlayerIndex].IsAI {
					result.ScheduleAIRoomIDs = append(result.ScheduleAIRoomIDs, roomID)
				}
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, gameactor.ErrActorClosed) {
				continue
			}
			m.mu.RLock()
			if _, ok := m.rooms[roomID]; ok {
				result.ScheduleAIRoomIDs = append(result.ScheduleAIRoomIDs, roomID)
			}
			m.mu.RUnlock()
		}
	}

	if len(result.DestroyedRoomIDs) > 0 {
		m.mu.Lock()
		for _, roomID := range result.DestroyedRoomIDs {
			delete(m.rooms, roomID)
			m.removeRoomAgents(roomID)
			m.RemoveRoom(roomID)
		}
		m.mu.Unlock()
	}

	return result
}
