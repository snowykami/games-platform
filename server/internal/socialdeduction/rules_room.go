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
	room.RecentActions = nil
	room.Werewolf = WerewolfState{RoleConfig: roleConfig, RolePresets: werewolfRolePresets(len(room.Players)), NightActions: map[string]string{}, SeerChecks: map[string]Alignment{}, Votes: map[string]WerewolfVoteIntent{}, RevealedIdiots: map[string]bool{}, Day: 1}
	room.Avalon = AvalonState{TeamVotes: map[string]bool{}, QuestCards: map[string]string{}, Round: 1}
	room.Undercover = UndercoverState{PresetID: undercoverConfig.PresetID, IncludeBlank: undercoverConfig.IncludeBlank, Presets: undercoverPresets(), Described: map[string]bool{}, Votes: map[string]UndercoverVoteIntent{}, Round: 1}
	for index, player := range room.Players {
		player.Seat = index
		player.Alive = true
		player.Role = ""
		player.Alignment = ""
	}
}
