package socialdeduction

type PublicRoomOptions struct {
	GodViewAvailable bool
	GodView          bool
}

func (m *Manager) publicRoom(room *Room, viewerUserID string) PublicRoom {
	return m.publicRoomWithOptions(room, viewerUserID, PublicRoomOptions{})
}

func (m *Manager) publicRoomWithOptions(room *Room, viewerUserID string, options PublicRoomOptions) PublicRoom {
	viewer := findPlayerByUserID(room, viewerUserID)
	godView := options.GodViewAvailable && options.GodView
	players := make([]PublicPlayer, 0, len(room.Players))
	for _, player := range room.Players {
		visible := godView || roleVisible(room, viewer, player)
		publicPlayer := PublicPlayer{
			ID:             player.ID,
			UserID:         player.UserID,
			Name:           player.Name,
			Seat:           player.Seat,
			RoomRole:       player.RoomRole,
			Kind:           player.Kind,
			IsAI:           player.IsAI,
			Connected:      player.Connected,
			DisconnectedAt: player.DisconnectedAt,
			AI:             player.AI,
			Alive:          player.Alive,
			VisibleToYou:   visible,
			Note:           playerNote(room, viewer, player.ID),
		}
		if visible {
			publicPlayer.Role = player.Role
			publicPlayer.Alignment = player.Alignment
		}
		players = append(players, publicPlayer)
	}

	logs := room.Log
	if len(logs) > 12 {
		logs = logs[len(logs)-12:]
	}
	youPlayerID := ""
	if viewer != nil {
		youPlayerID = viewer.ID
	}

	return PublicRoom{
		ID:               room.ID,
		Game:             room.Game,
		HostUserID:       room.HostUserID,
		HostPlayerID:     playerIDForUser(room, room.HostUserID),
		Phase:            room.Phase,
		Players:          players,
		YouPlayerID:      youPlayerID,
		MinPlayers:       m.minPlayers(),
		MaxPlayers:       m.maxPlayers(),
		Werewolf:         werewolfViewForViewer(room, viewer),
		Avalon:           avalonViewForViewer(room),
		Undercover:       undercoverViewForViewer(room, viewer),
		Winner:           room.Winner,
		WinnerMessage:    room.WinnerMessage,
		GodViewAvailable: options.GodViewAvailable,
		GodViewEnabled:   godView,
		Log:              append([]LogEntry{}, logs...),
		Speeches:         append([]SpeechEntry{}, room.Speeches...),
		ActionSeq:        room.ActionSeq,
		RecentActions:    append([]PublicAction{}, room.RecentActions...),
	}
}

func playerIDForUser(room *Room, userID string) string {
	if player := findPlayerByUserID(room, userID); player != nil {
		return player.ID
	}
	return ""
}

func roleVisible(room *Room, viewer *Player, target *Player) bool {
	if room.Phase == PhaseFinished {
		return true
	}
	if viewer == nil || target.Role == "" {
		return false
	}
	if viewer.ID == target.ID {
		return true
	}
	if room.Game == GameWerewolf {
		return viewer.Role == RoleWerewolf && target.Role == RoleWerewolf
	}
	if room.Game == GameAvalon {
		if viewer.Alignment == AlignmentEvil && target.Alignment == AlignmentEvil {
			return true
		}
		return viewer.Role == RoleMerlin && target.Alignment == AlignmentEvil
	}
	if room.Game == GameUndercover {
		return viewer.ID == target.ID
	}
	return false
}

func werewolfViewForViewer(room *Room, viewer *Player) WerewolfView {
	view := WerewolfView{
		Day:             room.Werewolf.Day,
		RoleConfig:      room.Werewolf.RoleConfig,
		RolePresets:     werewolfRolePresets(len(room.Players)),
		SeerChecks:      seerChecksForViewer(room, viewer),
		Votes:           cloneWerewolfVotes(room.Werewolf.Votes),
		DaySpeakers:     cloneBoolMap(room.Werewolf.DaySpeakers),
		LastNight:       room.Werewolf.LastNight,
		HunterPendingID: room.Werewolf.HunterPendingID,
		RevealedIdiots:  cloneBoolMap(room.Werewolf.RevealedIdiots),
	}
	if viewer != nil && viewer.Role == RoleWitch {
		view.WitchVictimID = currentWerewolfKillTarget(room)
		view.WitchAntidoteUsed = room.Werewolf.WitchAntidoteUsed
		view.WitchPoisonUsed = room.Werewolf.WitchPoisonUsed
	}
	if viewer != nil && room.Phase == PhaseWerewolfNight {
		_, view.NightSubmitted = room.Werewolf.NightActions[viewer.ID]
	}
	return view
}

func avalonViewForViewer(room *Room) AvalonView {
	teamVotes := cloneBoolMap(room.Avalon.TeamVotes)
	if room.Phase == PhaseAvalonVote {
		teamVotes = map[string]bool{}
	}
	return AvalonView{
		Round:         room.Avalon.Round,
		LeaderID:      room.Avalon.LeaderID,
		Team:          append([]string{}, room.Avalon.Team...),
		TeamVotes:     teamVotes,
		TeamVoteCount: len(room.Avalon.TeamVotes),
		QuestResults:  append([]AvalonQuestResult{}, room.Avalon.QuestResults...),
		RejectedTeams: room.Avalon.RejectedTeams,
		RequiredTeam:  room.Avalon.RequiredTeam,
		RequiredFails: room.Avalon.RequiredFails,
		Successes:     room.Avalon.Successes,
		Fails:         room.Avalon.Fails,
	}
}
