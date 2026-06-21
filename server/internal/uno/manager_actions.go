package uno

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need_two_players")
	}

	deck := shuffle(createDeck(room.Rules))
	room.DrawPile = deck
	room.DiscardPile = nil
	room.Direction = 1
	room.CurrentPlayerIndex = 0
	room.WinnerID = ""
	room.PendingDrawCount = 0
	room.PendingDrawKind = ""
	room.FlipSide = false
	room.TurnDeadline = nil
	room.Phase = PhasePlaying
	room.Log = append(room.Log, createLog("游戏开始。"))
	for _, player := range room.Players {
		player.Hand = drawCards(room, 7)
		player.HandCount = len(player.Hand)
		player.NeedsUNO = false
	}
	for len(room.DrawPile) > 0 {
		card := drawCards(room, 1)[0]
		if card.Color != ColorWild || room.Rules.AllWild {
			room.DiscardPile = append(room.DiscardPile, card)
			room.ActiveColor = startingColor(card)
			break
		}
		room.DrawPile = append(room.DrawPile, card)
	}

	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	return publicRoom(room, actorID), nil
}

func (m *Manager) Play(roomID string, actorID string, cardID string, color Color) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()

	player, err := playingPlayer(room, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

	cardIndex := slices.IndexFunc(player.Hand, func(card Card) bool {
		return card.ID == cardID
	})
	if cardIndex < 0 {
		return PublicRoom{}, errors.New("card_not_found")
	}

	card := player.Hand[cardIndex]
	isCurrentPlayer := room.CurrentPlayerIndex < len(room.Players) && room.Players[room.CurrentPlayerIndex].ID == player.ID
	if isCurrentPlayer && !isPlayable(card, room) {
		return PublicRoom{}, errors.New("card_not_playable")
	}
	if !isCurrentPlayer && !canJumpIn(card, player, room) {
		return PublicRoom{}, errors.New("card_not_playable")
	}
	if card.Color == ColorWild && !isRealColor(color) {
		return PublicRoom{}, errors.New("wild_color_required")
	}
	if room.CurrentPlayerIndex >= len(room.Players) || room.Players[room.CurrentPlayerIndex].ID != player.ID {
		room.CurrentPlayerIndex = playerIndex(room, player.ID)
	}

	playCard(room, player, cardIndex, color)
	room.UpdatedAt = time.Now().UTC()
	refreshTurnDeadline(room, room.UpdatedAt)
	return publicRoom(room, actorID), nil
}

func (m *Manager) Draw(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()

	player, err := currentHuman(room, actorID)
	if err != nil {
		return PublicRoom{}, err
	}

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
	message := fmt.Sprintf("%s 摸了 %d 张牌。", player.Name, len(drawn))
	room.Log = append(room.Log, createLog(message))
	recordDrawAction(room, player, player, len(drawn), message)
	advanceTurn(room)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) CallUNO(roomID string, actorID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	player := findPlayerByUserID(room, actorID)
	if player == nil || len(player.Hand) != 1 || !player.NeedsUNO {
		return PublicRoom{}, errors.New("uno_call_unavailable")
	}

	player.NeedsUNO = false
	message := fmt.Sprintf("%s 喊了 UNO。", player.Name)
	room.Log = append(room.Log, createLog(message))
	recordEffectAction(room, player, player, message)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) CatchUNO(roomID string, actorID string, targetID string) (PublicRoom, error) {
	room, err := m.lockRoom(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	defer room.mu.Unlock()
	actor := findPlayerByUserID(room, actorID)
	target := findPlayerByID(room, targetID)
	if actor == nil || target == nil || !target.NeedsUNO {
		return PublicRoom{}, errors.New("uno_catch_unavailable")
	}

	drawn := drawCards(room, 2)
	target.Hand = append(target.Hand, drawn...)
	target.HandCount = len(target.Hand)
	target.NeedsUNO = false
	message := fmt.Sprintf("%s 抓到 %s 没喊 UNO，%s 摸了 %d 张牌。", actor.Name, target.Name, target.Name, len(drawn))
	room.Log = append(room.Log, createLog(message))
	recordDrawAction(room, actor, target, len(drawn), message)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}
