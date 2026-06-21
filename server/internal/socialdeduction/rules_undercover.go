package socialdeduction

import "fmt"

func startUndercover(room *Room) {
	pair := chooseUndercoverPair(room.Undercover.PresetID)
	players := shuffledPlayers(room.Players)
	undercoverCount := undercoverCountForPlayers(len(players))
	blankCount := 0
	if room.Undercover.IncludeBlank && len(players) >= 6 {
		blankCount = 1
	}
	for index, player := range players {
		switch {
		case index < undercoverCount:
			player.Role = RoleUndercover
			player.Alignment = AlignmentEvil
		case index < undercoverCount+blankCount:
			player.Role = RoleBlank
			player.Alignment = AlignmentNeutral
		default:
			player.Role = RoleCivilian
			player.Alignment = AlignmentGood
		}
	}
	room.Phase = PhaseUndercoverDescribe
	room.Undercover.Round = 1
	room.Undercover.WordPair = pair
	room.Undercover.Presets = nil
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]UndercoverVoteIntent{}
	room.Undercover.CurrentSpeakerID = firstLivingPlayerID(room)
	room.Undercover.LastEliminatedID = ""
	room.Log = append(room.Log, createLog(fmt.Sprintf("谁是卧底开始，题库：%s。请依次描述自己的词。", undercoverPresetName(room.Undercover.PresetID))))
	recordAction(room, PublicAction{Type: "start", Message: "谁是卧底开始，进入描述阶段。"})
}

func advanceUndercoverSpeaker(room *Room) {
	next := nextUndescribedLivingPlayer(room)
	if next != nil {
		room.Undercover.CurrentSpeakerID = next.ID
		return
	}
	room.Phase = PhaseUndercoverVote
	room.Undercover.CurrentSpeakerID = ""
	room.Undercover.Votes = map[string]UndercoverVoteIntent{}
	room.Log = append(room.Log, createLog("本轮描述结束，开始投票。"))
	recordAction(room, PublicAction{Type: "undercover_vote_started", Message: "开始投票。"})
}

func resolveUndercoverVote(room *Room) {
	confirmedVotes := confirmedUndercoverVotes(room)
	if len(confirmedVotes) < livingCount(room) {
		return
	}
	targetID, tied := mostVotedUndercoverTarget(confirmedVotes)
	if tied || targetID == "" {
		room.Log = append(room.Log, createLog("本轮投票平票，无人出局。"))
		recordAction(room, PublicAction{Type: "undercover_vote_tied", Message: "投票平票，无人出局。"})
		startNextUndercoverRound(room)
		return
	}
	target := findPlayerByID(room, targetID)
	if target != nil {
		target.Alive = false
		room.Undercover.LastEliminatedID = target.ID
		message := fmt.Sprintf("%s 被投票出局。", target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "undercover_eliminate", TargetID: target.ID, Message: message})
	}
	if checkUndercoverWin(room) {
		return
	}
	startNextUndercoverRound(room)
}

func confirmedUndercoverVotes(room *Room) map[string]string {
	votes := map[string]string{}
	for actorID, vote := range room.Undercover.Votes {
		if vote.Confirmed && vote.TargetID != "" {
			votes[actorID] = vote.TargetID
		}
	}
	return votes
}

func startNextUndercoverRound(room *Room) {
	room.Undercover.Round++
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]UndercoverVoteIntent{}
	room.Undercover.CurrentSpeakerID = firstLivingPlayerID(room)
	room.Phase = PhaseUndercoverDescribe
	room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 轮描述开始。", room.Undercover.Round)))
	recordAction(room, PublicAction{Type: "undercover_round_started", Message: "下一轮描述开始。"})
}

func checkUndercoverWin(room *Room) bool {
	civilians := 0
	undercover := 0
	blank := 0
	living := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		living++
		switch player.Role {
		case RoleUndercover:
			undercover++
		case RoleBlank:
			blank++
		default:
			civilians++
		}
	}
	if undercover == 0 && blank == 0 {
		finish(room, AlignmentGood, "所有卧底阵营出局，平民获胜。")
		return true
	}
	if blank > 0 && undercover == 0 && living <= 2 {
		finish(room, AlignmentNeutral, "白板留到最后，白板获胜。")
		return true
	}
	if undercover+blank >= civilians || living <= 3 {
		finish(room, AlignmentEvil, "卧底阵营隐藏到最后，卧底获胜。")
		return true
	}
	return false
}
