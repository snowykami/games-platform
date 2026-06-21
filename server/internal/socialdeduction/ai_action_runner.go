package socialdeduction

import (
	"fmt"
	"slices"
	"strings"
)

func (m *Manager) runAIAction(room *Room, player *Player) bool {
	switch room.Phase {
	case PhaseWerewolfNight:
		if !canActAtNight(player) {
			return false
		}
		_, wolfConsensus := werewolfConsensusAction(room)
		if _, ok := room.Werewolf.NightActions[player.ID]; ok {
			if player.Role != RoleWerewolf || wolfConsensus || !allLivingWerewolvesActed(room) {
				return false
			}
		}
		actionID, speech := m.chooseWerewolfNightAction(room, player)
		if actionID == "" {
			return false
		}
		if _, err := applyWerewolfNightAction(room, player, actionID); err != nil {
			return false
		}
		if player.Role == RoleWerewolf && speech != "" {
			recordWerewolfWolfSpeech(room, player, speech)
		}
		m.advanceWerewolfNight(room)
		return true
	case PhaseWerewolfVote:
		if vote, ok := room.Werewolf.Votes[player.ID]; ok && vote.Confirmed {
			return false
		}
		if room.Werewolf.RevealedIdiots[player.ID] {
			return false
		}
		target, speech := m.chooseWerewolfVote(room, player)
		if target == nil {
			return false
		}
		room.Werewolf.Votes[player.ID] = WerewolfVoteIntent{TargetID: target.ID, Confirmed: true}
		if isGenericWerewolfVoteSpeech(speech) {
			speech = fallbackWerewolfVoteSpeech(room, player, target)
		}
		recordSpeech(room, player, speech)
		m.resolveWerewolfVote(room)
		return true
	case PhaseWerewolfHunter:
		if room.Werewolf.HunterPendingID != player.ID {
			return false
		}
		target, speech, ok := m.chooseHunterShot(room, player)
		if !ok {
			return false
		}
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		targetID := ""
		if target != nil {
			targetID = target.ID
		}
		return resolveHunterShot(room, targetID) == nil
	case PhaseAvalonTeam:
		if room.Avalon.LeaderID != player.ID {
			return false
		}
		team, speech := m.chooseAvalonTeam(room, player)
		if len(team) != room.Avalon.RequiredTeam {
			return false
		}
		room.Avalon.Team = team
		room.Avalon.TeamVotes = map[string]bool{}
		room.Phase = PhaseAvalonVote
		if speech == "" {
			speech = "这队先试一下。"
		}
		recordSpeech(room, player, speech)
		return true
	case PhaseAvalonVote:
		if _, ok := room.Avalon.TeamVotes[player.ID]; ok {
			return false
		}
		approve, _, ok := m.chooseAvalonTeamVote(room, player)
		if !ok {
			return false
		}
		room.Avalon.TeamVotes[player.ID] = approve
		m.resolveAvalonTeamVote(room)
		return true
	case PhaseAvalonQuest:
		if !slices.Contains(room.Avalon.Team, player.ID) {
			return false
		}
		if _, ok := room.Avalon.QuestCards[player.ID]; ok {
			return false
		}
		card, _ := m.chooseAvalonQuestCard(room, player)
		if card == "" {
			return false
		}
		room.Avalon.QuestCards[player.ID] = card
		m.resolveAvalonQuest(room)
		return true
	case PhaseAssassination:
		if player.Role != RoleAssassin {
			return false
		}
		target, speech := m.chooseAvalonAssassination(room, player)
		if target == nil {
			return false
		}
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		if target.Role == RoleMerlin {
			finish(room, AlignmentEvil, fmt.Sprintf("%s 刺中了梅林，邪恶阵营逆转获胜。", player.Name))
		} else {
			finish(room, AlignmentGood, fmt.Sprintf("%s 没有找到梅林，正义阵营获胜。", player.Name))
		}
		return true
	case PhaseUndercoverDescribe:
		if room.Undercover.CurrentSpeakerID != player.ID {
			return false
		}
		text := m.chooseUndercoverDescription(room, player)
		if text == "" {
			return false
		}
		recordSpeech(room, player, text)
		room.Undercover.Described[player.ID] = true
		recordAction(room, PublicAction{Type: "undercover_describe", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 完成了描述。", player.Name)})
		advanceUndercoverSpeaker(room)
		return true
	case PhaseUndercoverVote:
		if vote, ok := room.Undercover.Votes[player.ID]; ok && vote.Confirmed {
			return false
		}
		target, speech := m.chooseUndercoverVote(room, player)
		if target == nil {
			return false
		}
		room.Undercover.Votes[player.ID] = UndercoverVoteIntent{TargetID: target.ID, Confirmed: true}
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		recordAction(room, PublicAction{Type: "undercover_vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已确认投票。", player.Name)})
		resolveUndercoverVote(room)
		return true
	default:
		return false
	}
}

func isGenericWerewolfVoteSpeech(speech string) bool {
	normalized := strings.TrimSpace(strings.Trim(speech, "。.!！?？~～ "))
	if normalized == "" {
		return true
	}
	generic := map[string]bool{
		"我投这里":    true,
		"我票这里":    true,
		"我先投这里":   true,
		"我先票这里":   true,
		"我投这个":    true,
		"我票这个":    true,
		"我先票这个位置": true,
		"我先投这个位置": true,
		"先票这个位置":  true,
	}
	return generic[normalized]
}

func fallbackWerewolfVoteSpeech(room *Room, actor *Player, target *Player) string {
	targetRef := fmt.Sprintf("%d号", aiPlayerNumber(room, target))
	templates := []string{
		"我先压%s，看看票型怎么走。",
		"%s刚才那段不太自然，先挂一票。",
		"这轮我更不放心%s，先给压力。",
		"%s的位置有点滑，我先投这边。",
		"先票%s，明天再看谁跟得最顺。",
		"%s没有把逻辑补清楚，我先站这里。",
	}
	index := (aiPlayerNumber(room, actor) + room.Werewolf.Day) % len(templates)
	return fmt.Sprintf(templates[index], targetRef)
}
