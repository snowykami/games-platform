package socialdeduction

import (
	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func (m *Manager) canUseLLM(player *Player) bool {
	return player.AI != nil && player.AI.Level == string(aiplayer.LevelLLM) && m.aiProvider != nil && m.aiProvider.Enabled()
}

func werewolfAIState(room *Room, actor *Player) map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	visibleAllies := []string{}
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    aiPlayerRef(room, player),
			"seat":  aiPlayerNumber(room, player),
			"alive": player.Alive,
		}
		if roleVisible(room, actor, player) {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
			if player.ID != actor.ID && player.Alignment == actor.Alignment {
				visibleAllies = append(visibleAllies, aiPlayerRef(room, player))
			}
		}
		players = append(players, entry)
	}
	state := map[string]any{
		"phase":           room.Phase,
		"day":             room.Werewolf.Day,
		"yourRole":        actor.Role,
		"yourAlignment":   actor.Alignment,
		"players":         players,
		"visibleAllies":   visibleAllies,
		"lastNight":       aliasPlayerNamesInText(room, room.Werewolf.LastNight),
		"publicFacts":     werewolfPublicFacts(room),
		"votes":           aliasWerewolfVoteIntents(room, room.Werewolf.Votes),
		"recentSpeech":    aiSpeechForWerewolf(room),
		"revealedIdiots":  aliasBoolMap(room, room.Werewolf.RevealedIdiots),
		"hunterPendingId": aliasOptionalPlayerID(room, room.Werewolf.HunterPendingID),
	}
	if actor.Role == RoleSeer {
		state["seerChecks"] = aliasAlignmentMap(room, room.Werewolf.SeerChecks)
	}
	if actor.Role == RoleWitch {
		state["witch"] = map[string]any{
			"victimId":     aliasOptionalPlayerID(room, currentWerewolfKillTarget(room)),
			"antidoteUsed": room.Werewolf.WitchAntidoteUsed,
			"poisonUsed":   room.Werewolf.WitchPoisonUsed,
		}
	}
	if actor.Role == RoleWerewolf {
		_, consensus := werewolfConsensusAction(room)
		state["wolfChat"] = aiWerewolfWolfSpeeches(room)
		state["wolfChoices"] = aliasWerewolfNightActions(room)
		state["wolfConsensusReached"] = consensus
	}
	return state
}

func avalonAIState(room *Room, actor *Player, phase string) map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    aiPlayerRef(room, player),
			"seat":  aiPlayerNumber(room, player),
			"alive": player.Alive,
		}
		if roleVisible(room, actor, player) {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
		}
		players = append(players, entry)
	}
	teamVotes := cloneBoolMap(room.Avalon.TeamVotes)
	if phase == "team_vote" {
		teamVotes = map[string]bool{}
	}
	state := map[string]any{
		"phase":         phase,
		"round":         room.Avalon.Round,
		"yourRole":      actor.Role,
		"yourAlignment": actor.Alignment,
		"players":       players,
		"leaderId":      aliasOptionalPlayerID(room, room.Avalon.LeaderID),
		"team":          aliasStringSlice(room, room.Avalon.Team),
		"teamVotes":     aliasBoolMap(room, teamVotes),
		"teamVoteCount": len(room.Avalon.TeamVotes),
		"questResults":  append([]AvalonQuestResult{}, room.Avalon.QuestResults...),
		"successes":     room.Avalon.Successes,
		"fails":         room.Avalon.Fails,
		"recentSpeech":  aiSpeeches(room),
	}
	if actor.Role == RolePercival {
		state["percivalMarks"] = aliasStringSlice(room, avalonPercivalMarks(room, actor))
	}
	return state
}

func undercoverAIState(room *Room, player *Player, phase string) map[string]any {
	word := undercoverWordForPlayer(room, player)
	return map[string]any{
		"phase":                 phase,
		"round":                 room.Undercover.Round,
		"yourRole":              player.Role,
		"yourWord":              word,
		"speechPolicy":          "描述阶段必须把 speech 写成最终要说出口的话，不能照抄 action label。只能给间接线索，绝不能直接说出、拼写、引用或复述 yourWord；空白牌也不能声称自己知道具体词。禁止空话套话，例如“生活里常见”“具体场景”“特点不能说太细”“先看大家怎么描述”。像真人一样给一个具体但不泄词的侧面线索。",
		"badSpeechExamples":     []string{"它在生活里挺常见", "它一般会出现在具体场景里", "它的特点不能说得太细", "我先说个比较宽的范围", "我会先看大家怎么描述"},
		"forbiddenPublicSpeech": forbiddenPublicSpeech(word),
		"players":               publicPlayersForAI(room, player),
		"described":             aliasBoolMap(room, room.Undercover.Described),
		"votes":                 aliasUndercoverVoteIntents(room, room.Undercover.Votes),
		"recentSpeech":          aiSpeeches(room),
	}
}

func publicPlayersForAI(room *Room, actor *Player) []map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    aiPlayerRef(room, player),
			"seat":  aiPlayerNumber(room, player),
			"alive": player.Alive,
		}
		if roleVisible(room, actor, player) {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
		}
		players = append(players, entry)
	}
	return players
}

type socialAISession struct {
	Game        GameKind
	RoomID      string
	PlayerID    string
	PrivateRole Role
	PrivateWord string
	Memory      []string
}

func (m *Manager) ensureAISession(room *Room, player *Player) *socialAISession {
	key := socialAISessionKey(room.ID, player.ID)
	session := m.aiSessions[key]
	if session == nil {
		session = &socialAISession{Game: room.Game, RoomID: room.ID, PlayerID: player.ID}
		m.aiSessions[key] = session
	}
	session.PrivateRole = player.Role
	session.PrivateWord = undercoverWordForPlayer(room, player)
	return session
}
