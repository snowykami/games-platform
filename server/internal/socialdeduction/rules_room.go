package socialdeduction

func resetRoom(room *Room) {
	roleConfig := room.Werewolf.RoleConfig
	undercoverConfig := room.Undercover
	if undercoverConfig.PresetID == "" {
		undercoverConfig.PresetID = defaultUndercoverPresetID()
	}
	if room.Game == GameWerewolf && roleConfig.Counts.total() == 0 {
		roleConfig = defaultWerewolfConfig(len(room.Players))
	}
	room.Winner = ""
	room.WinnerMessage = ""
	room.Log = nil
	room.Speeches = nil
	room.LastAISpeechSourceID = ""
	room.ActionSeq = 0
	room.RecentActions = nil
	room.AIDebugTraces = nil
	clearAIPlayerNotes(room)
	room.Werewolf = WerewolfState{RoleConfig: roleConfig, RolePresets: werewolfRolePresets(len(room.Players)), NightActions: map[string]string{}, WolfSpeeches: nil, SeerChecks: map[string]Alignment{}, Votes: map[string]WerewolfVoteIntent{}, DaySpeakers: map[string]bool{}, RevealedIdiots: map[string]bool{}, Day: 1}
	room.Avalon = AvalonState{TeamVotes: map[string]bool{}, QuestCards: map[string]string{}, Round: 1}
	room.Undercover = UndercoverState{PresetID: undercoverConfig.PresetID, IncludeBlank: undercoverConfig.IncludeBlank, Presets: undercoverPresets(), Described: map[string]bool{}, Votes: map[string]UndercoverVoteIntent{}, Round: 1}
	assignRandomSeats(room)
	for _, player := range room.Players {
		player.Alive = true
		player.Role = ""
		player.Alignment = ""
	}
}

func clearAIPlayerNotes(room *Room) {
	for _, player := range room.Players {
		if player.IsAI {
			delete(room.PlayerNotes, player.ID)
		}
	}
}
