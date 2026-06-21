package uno

import (
	"fmt"
	"log/slog"
	"time"
)

func refreshTurnDeadline(room *Room, now time.Time) {
	if room.Phase != PhasePlaying || len(room.Players) == 0 {
		room.TurnDeadline = nil
		return
	}
	deadline := now.UTC().Add(turnTimeout)
	room.TurnDeadline = &deadline
}

func applyTimeoutTurn(room *Room, now time.Time) bool {
	if room.Phase != PhasePlaying || len(room.Players) == 0 || room.TurnDeadline == nil || now.Before(*room.TurnDeadline) {
		return false
	}
	if room.CurrentPlayerIndex >= len(room.Players) {
		room.CurrentPlayerIndex = 0
	}

	player := room.Players[room.CurrentPlayerIndex]
	actions := legalAIActions(room, player)
	if len(actions) == 0 {
		actions = []aiTurnAction{{Kind: "draw"}}
	}
	action := actions[0]
	timeoutMessage := fmt.Sprintf("%s 超时，服务端自动行动。", player.Name)
	room.Log = append(room.Log, createLog(timeoutMessage))
	recordEffectAction(room, player, player, timeoutMessage)

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
		player.NeedsUNO = false
		message := fmt.Sprintf("%s 超时后自动摸了 %d 张牌。", player.Name, len(drawn))
		room.Log = append(room.Log, createLog(message))
		recordDrawAction(room, player, player, len(drawn), message)
		advanceTurn(room)
	} else {
		playCard(room, player, action.CardIndex, action.Color)
	}

	room.UpdatedAt = now
	refreshTurnDeadline(room, now)
	slog.Info("uno turn timeout auto action",
		"room", room.ID,
		"player", player.ID,
		"playerName", player.Name,
		"action", action.Kind,
	)
	return true
}

func shouldDestroyOfflineRoom(room *Room, now time.Time) bool {
	if room.Phase != PhaseLobby && room.Phase != PhasePlaying {
		return false
	}
	updateAllHumansOfflineSince(room, now)
	return room.AllHumansOfflineSince != nil && now.Sub(*room.AllHumansOfflineSince) >= offlineRoomTTL
}

func updateAllHumansOfflineSince(room *Room, now time.Time) {
	for _, player := range room.Players {
		if player.IsAI {
			continue
		}
		if player.Connected {
			room.AllHumansOfflineSince = nil
			return
		}
	}
	if room.AllHumansOfflineSince == nil {
		startedAt := now.UTC()
		room.AllHumansOfflineSince = &startedAt
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func turnRemainingSeconds(deadline *time.Time) int {
	if deadline == nil {
		return 0
	}
	remaining := time.Until(*deadline)
	if remaining <= 0 {
		return 0
	}
	return int((remaining + time.Second - time.Nanosecond) / time.Second)
}
