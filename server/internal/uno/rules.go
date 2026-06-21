package uno

import (
	"fmt"
	"slices"
	"time"
)

func playCard(room *Room, player *Player, cardIndex int, color Color) {
	card := player.Hand[cardIndex]
	player.Hand = slices.Delete(player.Hand, cardIndex, cardIndex+1)
	player.HandCount = len(player.Hand)
	player.NeedsUNO = false
	room.DiscardPile = append(room.DiscardPile, card)
	if card.Color == ColorWild {
		room.ActiveColor = color
	} else {
		room.ActiveColor = card.Color
	}
	message := fmt.Sprintf("%s 打出了 %s。", player.Name, formatCard(card))
	room.Log = append(room.Log, createLog(message))
	recordPlayAction(room, player, card, message)

	if len(player.Hand) == 0 {
		room.Phase = PhaseFinished
		room.WinnerID = player.ID
		winMessage := fmt.Sprintf("%s 获胜。", player.Name)
		room.Log = append(room.Log, createLog(winMessage))
		recordAction(room, PublicAction{
			Type:      ActionWin,
			ActorID:   player.ID,
			ActorName: player.Name,
			TargetID:  player.ID,
			Message:   winMessage,
		})
		return
	}
	if len(player.Hand) == 1 {
		player.NeedsUNO = !player.IsAI
		if player.IsAI {
			room.Log = append(room.Log, createLog(fmt.Sprintf("%s 喊了 UNO。", player.Name)))
		}
	}

	applyEffect(room, card)
}

func applyEffect(room *Room, card Card) {
	if applySevenZero(room, card) {
		advanceTurn(room)
		return
	}

	switch card.Kind {
	case KindSkip:
		target := room.Players[nextIndex(room)]
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], target, fmt.Sprintf("%s 被跳过。", target.Name))
		advanceTurn(room)
		advanceTurn(room)
	case KindReverse:
		room.Direction *= -1
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "回合方向反转。")
		if len(room.Players) == 2 {
			advanceTurn(room)
			advanceTurn(room)
			return
		}
		advanceTurn(room)
	case KindDrawTwo:
		applyDrawPenalty(room, card, 2)
	case KindWildDrawFour:
		applyDrawPenalty(room, card, 4)
	case KindWildDrawSix:
		applyDrawPenalty(room, card, 6)
	case KindWildDrawTen:
		applyDrawPenalty(room, card, 10)
	case KindFlip:
		room.FlipSide = !room.FlipSide
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "牌桌翻面。")
		advanceTurn(room)
	default:
		advanceTurn(room)
	}
}

func applyDrawPenalty(room *Room, card Card, count int) {
	if room.Rules.Stacking || room.Rules.NoMercy {
		room.PendingDrawCount += count
		room.PendingDrawKind = card.Kind
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[nextIndex(room)], fmt.Sprintf("罚牌累积到 %d 张。", room.PendingDrawCount))
		advanceTurn(room)
		return
	}

	next := nextIndex(room)
	drawn := drawCards(room, count)
	room.Players[next].Hand = append(room.Players[next].Hand, drawn...)
	room.Players[next].HandCount = len(room.Players[next].Hand)
	recordDrawAction(room, room.Players[room.CurrentPlayerIndex], room.Players[next], len(drawn), fmt.Sprintf("%s 摸了 %d 张牌。", room.Players[next].Name, len(drawn)))
	advanceTurn(room)
	advanceTurn(room)
}

func applySevenZero(room *Room, card Card) bool {
	if !room.Rules.SevenZero || card.Kind != KindNumber || card.Value == nil {
		return false
	}
	if *card.Value == 7 && len(room.Players) > 1 {
		current := room.Players[room.CurrentPlayerIndex]
		target := room.Players[nextIndex(room)]
		current.Hand, target.Hand = target.Hand, current.Hand
		current.HandCount = len(current.Hand)
		target.HandCount = len(target.Hand)
		recordEffectAction(room, current, target, fmt.Sprintf("%s 与 %s 交换手牌。", current.Name, target.Name))
		return true
	}
	if *card.Value == 0 && len(room.Players) > 1 {
		rotateHands(room)
		recordEffectAction(room, room.Players[room.CurrentPlayerIndex], room.Players[room.CurrentPlayerIndex], "所有玩家按当前方向轮换手牌。")
		return true
	}
	return false
}

func isPlayable(card Card, room *Room) bool {
	if len(room.DiscardPile) == 0 {
		return true
	}
	if room.Rules.AllWild {
		return true
	}
	if room.PendingDrawCount > 0 {
		return canStackDraw(card, room)
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	if card.Color == ColorWild || card.Color == room.ActiveColor {
		return true
	}
	if card.Kind == KindNumber {
		return top.Kind == KindNumber && card.Value != nil && top.Value != nil && *card.Value == *top.Value
	}
	return card.Kind == top.Kind
}

func canStackDraw(card Card, room *Room) bool {
	if !(room.Rules.Stacking || room.Rules.NoMercy) {
		return false
	}
	if room.Rules.NoMercy {
		return drawPenalty(card.Kind) > 0
	}
	return card.Kind == room.PendingDrawKind && drawPenalty(card.Kind) > 0
}

func canJumpIn(card Card, player *Player, room *Room) bool {
	if !room.Rules.JumpIn || len(room.DiscardPile) == 0 || room.PendingDrawCount > 0 {
		return false
	}
	if room.CurrentPlayerIndex < len(room.Players) && room.Players[room.CurrentPlayerIndex].ID == player.ID {
		return false
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	return sameFace(card, top)
}

func playableCardsForPlayer(player *Player, room *Room) []string {
	playableCardIDs := []string{}
	for _, card := range player.Hand {
		if isPlayable(card, room) || canJumpIn(card, player, room) {
			playableCardIDs = append(playableCardIDs, card.ID)
		}
	}
	return playableCardIDs
}

func drawCards(room *Room, count int) []Card {
	cards := []Card{}
	for range count {
		if len(room.DrawPile) == 0 {
			recycleDiscards(room)
		}
		if len(room.DrawPile) == 0 {
			return cards
		}
		card := room.DrawPile[len(room.DrawPile)-1]
		room.DrawPile = room.DrawPile[:len(room.DrawPile)-1]
		cards = append(cards, card)
	}
	return cards
}

func recycleDiscards(room *Room) {
	if len(room.DiscardPile) <= 1 {
		return
	}
	top := room.DiscardPile[len(room.DiscardPile)-1]
	recycled := shuffle(room.DiscardPile[:len(room.DiscardPile)-1])
	room.DiscardPile = []Card{top}
	room.DrawPile = append(room.DrawPile, recycled...)
}

func rotateHands(room *Room) {
	hands := make([][]Card, len(room.Players))
	for index, player := range room.Players {
		hands[index] = player.Hand
	}
	for index, player := range room.Players {
		source := (index - room.Direction + len(room.Players)) % len(room.Players)
		player.Hand = hands[source]
		player.HandCount = len(player.Hand)
	}
}

func advanceTurn(room *Room) {
	room.CurrentPlayerIndex = nextIndex(room)
	refreshTurnDeadline(room, time.Now().UTC())
}

func nextIndex(room *Room) int {
	total := len(room.Players)
	return (room.CurrentPlayerIndex + room.Direction + total) % total
}
