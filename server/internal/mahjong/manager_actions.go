package mahjong

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < minPlayers {
		return PublicRoom{}, errors.New("need_four_players")
	}

	resetGame(room)
	room.Phase = PhasePlaying
	room.HasDrawn = true
	room.Log = append(room.Log, createLog("东风起局，庄家先打。国标麻将 8 番起胡。"))
	recordAction(room, PublicAction{Type: ActionStart, ActorID: room.Players[0].ID, ActorName: room.Players[0].Name, Message: "麻将开始。"})
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Draw(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HasDrawn {
		return PublicRoom{}, errors.New("already_drawn")
	}
	drawForCurrent(room, player)
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Discard(roomID string, actorID string, tileID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !room.HasDrawn {
		return PublicRoom{}, errors.New("must_draw_first")
	}
	if !discardTile(room, player, tileID) {
		return PublicRoom{}, errors.New("tile_not_found")
	}
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) SelfDraw(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.currentHuman(roomID, actorID)
	if err != nil {
		return PublicRoom{}, err
	}
	if !room.HasDrawn {
		return PublicRoom{}, errors.New("must_draw_first")
	}
	result := evaluateWin(player.Hand, player.Melds, true, player.Wind, room.RoundWind, room.RuleSet)
	if !result.CanWin {
		return PublicRoom{}, errors.New("mahjong_win_unavailable")
	}
	finishWin(room, player, result, fmt.Sprintf("%s 自摸，%d 番。", player.Name, result.Fan))
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) Claim(roomID string, actorID string, claimID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if room.Phase != PhaseClaiming {
		return PublicRoom{}, errors.New("claim_unavailable")
	}
	claimIndex := slices.IndexFunc(room.ClaimOptions, func(option ClaimOption) bool {
		return option.ID == claimID && option.PlayerID == player.ID
	})
	if claimIndex < 0 {
		return PublicRoom{}, errors.New("claim_unavailable")
	}
	applyClaim(room, room.ClaimOptions[claimIndex])
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}

func (m *Manager) SkipClaims(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if room.Phase != PhaseClaiming {
		return PublicRoom{}, errors.New("claim_unavailable")
	}

	room.ClaimOptions = slices.DeleteFunc(room.ClaimOptions, func(option ClaimOption) bool {
		return option.PlayerID == player.ID
	})
	if len(room.ClaimOptions) == 0 {
		advanceAfterClaims(room)
	} else {
		room.Log = append(room.Log, createLog("暂不声明。"))
	}
	room.UpdatedAt = time.Now().UTC()
	return publicRoom(room, actorID), nil
}
