package mahjong

import (
	"fmt"
	"slices"
)

func resetGame(room *Room) {
	wall := shuffle(createWall())
	room.Wall = wall
	room.DeadWall = append([]Tile{}, room.Wall[len(room.Wall)-14:]...)
	room.Wall = room.Wall[:len(room.Wall)-14]
	room.CurrentPlayerIndex = 0
	room.DealerIndex = 0
	room.RoundWind = WindEast
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.WinnerID = ""
	room.WinResult = WinResult{}
	room.RecentActions = nil
	for index, player := range room.Players {
		player.Wind = winds[index]
		player.Hand = nil
		player.Melds = nil
		player.Discards = nil
		count := 13
		if index == 0 {
			count = 14
		}
		player.Hand = sortTiles(shiftTiles(&room.Wall, count))
	}
}

func drawForCurrent(room *Room, player *Player) {
	if len(room.Wall) == 0 {
		room.Phase = PhaseFinished
		room.Log = append(room.Log, createLog("流局：牌墙已经摸完。"))
		return
	}
	tile := shiftTiles(&room.Wall, 1)[0]
	player.Hand = sortTiles(append(player.Hand, tile))
	room.HasDrawn = true
	room.LastDiscard = nil
	message := fmt.Sprintf("%s 摸牌。", player.Name)
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionDraw, ActorID: player.ID, ActorName: player.Name, Message: message})
}

func discardTile(room *Room, player *Player, tileID string) bool {
	tileIndex := slices.IndexFunc(player.Hand, func(tile Tile) bool { return tile.ID == tileID })
	if tileIndex < 0 {
		return false
	}
	tile := player.Hand[tileIndex]
	player.Hand = append(player.Hand[:tileIndex], player.Hand[tileIndex+1:]...)
	player.Discards = append(player.Discards, tile)
	room.HasDrawn = false
	room.LastDiscard = &LastDiscard{Tile: tile, PlayerID: player.ID}
	message := fmt.Sprintf("%s 打出 %s。", player.Name, formatTile(tile))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionDiscard, ActorID: player.ID, ActorName: player.Name, Tile: &tile, Message: message})
	openClaimWindow(room)
	return true
}

func openClaimWindow(room *Room) {
	options := createClaimOptions(room)
	botHuIndex := slices.IndexFunc(options, func(option ClaimOption) bool {
		player := findPlayerByID(room, option.PlayerID)
		return option.Kind == ClaimHu && player != nil && player.IsAI
	})
	if botHuIndex >= 0 {
		applyClaim(room, options[botHuIndex])
		return
	}

	humanOptions := slices.DeleteFunc(options, func(option ClaimOption) bool {
		player := findPlayerByID(room, option.PlayerID)
		return player == nil || player.IsAI
	})
	if len(humanOptions) > 0 {
		room.ClaimOptions = humanOptions
		room.Phase = PhaseClaiming
		room.Log = append(room.Log, createLog("有人可以声明吃、碰或胡。"))
		return
	}

	advanceAfterClaims(room)
}

func applyClaim(room *Room, claim ClaimOption) {
	player := findPlayerByID(room, claim.PlayerID)
	if player == nil || room.LastDiscard == nil {
		return
	}
	if claim.Kind == ClaimHu && claim.WinResult.CanWin {
		message := fmt.Sprintf("%s 荣和 %s，%d 番。", player.Name, formatTile(claim.Tile), claim.WinResult.Fan)
		finishWin(room, player, claim.WinResult, message)
		return
	}

	discarder := findPlayerByID(room, room.LastDiscard.PlayerID)
	if discarder != nil {
		discarder.Discards = slices.DeleteFunc(discarder.Discards, func(tile Tile) bool { return tile.ID == claim.Tile.ID })
	}
	used := map[string]bool{}
	for _, tile := range claim.TilesFromHand {
		used[tile.ID] = true
	}
	player.Hand = slices.DeleteFunc(player.Hand, func(tile Tile) bool { return used[tile.ID] })
	meldKind := MeldPung
	label := "碰"
	if claim.Kind == ClaimChi {
		meldKind = MeldChow
		label = "吃"
	}
	player.Melds = append(player.Melds, Meld{
		ID:           "meld_" + randomToken(10),
		Kind:         meldKind,
		Tiles:        sortTiles(append(append([]Tile{}, claim.TilesFromHand...), claim.Tile)),
		FromPlayerID: room.LastDiscard.PlayerID,
		Exposed:      true,
	})
	room.CurrentPlayerIndex = playerIndex(room, player.ID)
	room.HasDrawn = true
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.Phase = PhasePlaying
	message := fmt.Sprintf("%s %s了 %s。", player.Name, label, formatTile(claim.Tile))
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionClaim, ActorID: player.ID, ActorName: player.Name, Tile: &claim.Tile, Message: message})
}

func finishWin(room *Room, player *Player, result WinResult, message string) {
	room.Phase = PhaseFinished
	room.WinnerID = player.ID
	room.WinResult = result
	room.ClaimOptions = nil
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: ActionWin, ActorID: player.ID, ActorName: player.Name, Message: message})
}

func advanceAfterClaims(room *Room) {
	if room.LastDiscard == nil {
		return
	}
	room.CurrentPlayerIndex = nextPlayerIndex(room, room.LastDiscard.PlayerID)
	room.HasDrawn = false
	room.LastDiscard = nil
	room.ClaimOptions = nil
	room.Phase = PhasePlaying
}

func createClaimOptions(room *Room) []ClaimOption {
	if room.LastDiscard == nil {
		return nil
	}
	discard := room.LastDiscard
	discarderIndex := playerIndex(room, discard.PlayerID)
	nextIndex := (discarderIndex + 1) % len(room.Players)
	options := []ClaimOption{}
	for index, player := range room.Players {
		if player.ID == discard.PlayerID {
			continue
		}
		winTiles := sortTiles(append(append([]Tile{}, player.Hand...), discard.Tile))
		winResult := evaluateWin(winTiles, player.Melds, false, player.Wind, room.RoundWind, room.RuleSet)
		if winResult.CanWin {
			options = append(options, ClaimOption{
				ID:        fmt.Sprintf("hu_%s_%s", player.ID, discard.Tile.ID),
				PlayerID:  player.ID,
				Kind:      ClaimHu,
				Tile:      discard.Tile,
				WinResult: winResult,
			})
		}
		sameTiles := filterTiles(player.Hand, func(tile Tile) bool { return tile.Code == discard.Tile.Code })
		if len(sameTiles) >= 2 {
			options = append(options, ClaimOption{
				ID:            fmt.Sprintf("peng_%s_%s", player.ID, discard.Tile.ID),
				PlayerID:      player.ID,
				Kind:          ClaimPeng,
				Tile:          discard.Tile,
				TilesFromHand: sameTiles[:2],
			})
		}
		if index == nextIndex {
			options = append(options, chiOptions(player, discard.Tile)...)
		}
	}
	return options
}

func chiOptions(player *Player, tile Tile) []ClaimOption {
	if tile.Rank == 0 || !isSuited(tile.Code) {
		return nil
	}
	windows := [][]int{{tile.Rank - 2, tile.Rank - 1}, {tile.Rank - 1, tile.Rank + 1}, {tile.Rank + 1, tile.Rank + 2}}
	options := []ClaimOption{}
	for _, ranks := range windows {
		if ranks[0] < 1 || ranks[1] > 9 {
			continue
		}
		first, okFirst := firstTileByCode(player.Hand, fmt.Sprintf("%s%d", tile.Code[:1], ranks[0]))
		second, okSecond := firstTileByCode(player.Hand, fmt.Sprintf("%s%d", tile.Code[:1], ranks[1]))
		if !okFirst || !okSecond {
			continue
		}
		tiles := []Tile{first, second}
		options = append(options, ClaimOption{
			ID:            fmt.Sprintf("chi_%s_%s_%s_%s", player.ID, tile.ID, first.ID, second.ID),
			PlayerID:      player.ID,
			Kind:          ClaimChi,
			Tile:          tile,
			TilesFromHand: tiles,
		})
	}
	return options
}
