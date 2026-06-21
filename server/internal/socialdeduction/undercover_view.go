package socialdeduction

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func applyDefaultUndercoverConfig(room *Room) {
	if room.Undercover.PresetID == "" {
		room.Undercover.PresetID = defaultUndercoverPresetID()
	}
	room.Undercover.Presets = undercoverPresets()
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]UndercoverVoteIntent{}
}

func defaultUndercoverPresetID() string {
	return "daily"
}

func undercoverPresets() []UndercoverPreset {
	return []UndercoverPreset{
		{ID: "daily", Name: "日常生活", Description: "生活里常见但容易混淆的词。", Pairs: []UndercoverWordPair{
			{ID: "daily-1", CivilianWord: "咖啡", UndercoverWord: "奶茶", Category: "饮品"},
			{ID: "daily-2", CivilianWord: "公交车", UndercoverWord: "地铁", Category: "交通"},
			{ID: "daily-3", CivilianWord: "雨伞", UndercoverWord: "遮阳伞", Category: "物品"},
			{ID: "daily-4", CivilianWord: "键盘", UndercoverWord: "钢琴", Category: "物品"},
			{ID: "daily-5", CivilianWord: "火锅", UndercoverWord: "麻辣烫", Category: "食物"},
			{ID: "daily-6", CivilianWord: "电影院", UndercoverWord: "剧院", Category: "地点"},
		}},
		{ID: "internet", Name: "网络热词", Description: "更适合熟人局的互联网语境题库。", Pairs: []UndercoverWordPair{
			{ID: "internet-1", CivilianWord: "弹幕", UndercoverWord: "评论区", Category: "网络"},
			{ID: "internet-2", CivilianWord: "直播", UndercoverWord: "短视频", Category: "网络"},
			{ID: "internet-3", CivilianWord: "表情包", UndercoverWord: "贴纸", Category: "网络"},
			{ID: "internet-4", CivilianWord: "摸鱼", UndercoverWord: "摆烂", Category: "网络"},
			{ID: "internet-5", CivilianWord: "热搜", UndercoverWord: "推荐页", Category: "网络"},
		}},
		{ID: "anime", Name: "轻二次元", Description: "偏 ACG 的非 IP 词库，不依赖具体版权角色。", Pairs: []UndercoverWordPair{
			{ID: "anime-1", CivilianWord: "魔法少女", UndercoverWord: "变身英雄", Category: "幻想"},
			{ID: "anime-2", CivilianWord: "社团活动", UndercoverWord: "校园祭", Category: "校园"},
			{ID: "anime-3", CivilianWord: "机甲", UndercoverWord: "机器人", Category: "科幻"},
			{ID: "anime-4", CivilianWord: "异世界", UndercoverWord: "平行宇宙", Category: "幻想"},
			{ID: "anime-5", CivilianWord: "必杀技", UndercoverWord: "连招", Category: "战斗"},
		}},
		{ID: "ai-curated", Name: "AI 推荐", Description: "按 AI 参与感设计的更抽象题库。", Pairs: []UndercoverWordPair{
			{ID: "ai-1", CivilianWord: "灵感", UndercoverWord: "直觉", Category: "抽象"},
			{ID: "ai-2", CivilianWord: "记忆", UndercoverWord: "回忆", Category: "抽象"},
			{ID: "ai-3", CivilianWord: "计划", UndercoverWord: "策略", Category: "抽象"},
			{ID: "ai-4", CivilianWord: "规则", UndercoverWord: "约定", Category: "抽象"},
			{ID: "ai-5", CivilianWord: "推理", UndercoverWord: "猜测", Category: "抽象"},
		}},
	}
}

func undercoverPresetExists(id string) bool {
	for _, preset := range undercoverPresets() {
		if preset.ID == id {
			return true
		}
	}
	return false
}

func undercoverPresetName(id string) string {
	for _, preset := range undercoverPresets() {
		if preset.ID == id {
			return preset.Name
		}
	}
	return undercoverPresets()[0].Name
}

func chooseUndercoverPair(presetID string) UndercoverWordPair {
	for _, preset := range undercoverPresets() {
		if preset.ID == presetID && len(preset.Pairs) > 0 {
			return preset.Pairs[rand.IntN(len(preset.Pairs))]
		}
	}
	preset := undercoverPresets()[0]
	return preset.Pairs[rand.IntN(len(preset.Pairs))]
}

func undercoverCountForPlayers(count int) int {
	if count >= 7 {
		return 2
	}
	return 1
}

func firstLivingPlayerID(room *Room) string {
	for _, player := range playersBySeat(room) {
		if player.Alive {
			return player.ID
		}
	}
	return ""
}

func nextUndescribedLivingPlayer(room *Room) *Player {
	for _, player := range playersBySeat(room) {
		if player.Alive && !room.Undercover.Described[player.ID] {
			return player
		}
	}
	return nil
}

func firstSeatPlayerID(room *Room) string {
	players := playersBySeat(room)
	if len(players) == 0 {
		return ""
	}
	return players[0].ID
}

func mostVotedUndercoverTarget(votes map[string]string) (string, bool) {
	counts := map[string]int{}
	bestID := ""
	bestCount := 0
	tied := false
	for _, targetID := range votes {
		counts[targetID]++
		switch {
		case counts[targetID] > bestCount:
			bestID = targetID
			bestCount = counts[targetID]
			tied = false
		case counts[targetID] == bestCount:
			tied = true
		}
	}
	return bestID, tied
}

func undercoverWordForPlayer(room *Room, player *Player) string {
	switch player.Role {
	case RoleUndercover:
		return room.Undercover.WordPair.UndercoverWord
	case RoleBlank:
		return ""
	default:
		return room.Undercover.WordPair.CivilianWord
	}
}

func undercoverViewForViewer(room *Room, viewer *Player) UndercoverView {
	view := UndercoverView{
		Round:            room.Undercover.Round,
		PresetID:         room.Undercover.PresetID,
		IncludeBlank:     room.Undercover.IncludeBlank,
		CurrentSpeakerID: room.Undercover.CurrentSpeakerID,
		Described:        cloneBoolMap(room.Undercover.Described),
		Votes:            cloneUndercoverVotes(room.Undercover.Votes),
		LastEliminatedID: room.Undercover.LastEliminatedID,
	}
	if room.Phase == PhaseLobby {
		view.Presets = undercoverPresets()
		return view
	}
	if room.Phase == PhaseFinished {
		view.WordPair = room.Undercover.WordPair
		return view
	}
	if viewer != nil {
		view.WordPair = UndercoverWordPair{ID: room.Undercover.WordPair.ID, Category: room.Undercover.WordPair.Category}
		if viewer.Role == RoleUndercover {
			view.WordPair.UndercoverWord = room.Undercover.WordPair.UndercoverWord
		} else if viewer.Role == RoleCivilian {
			view.WordPair.CivilianWord = room.Undercover.WordPair.CivilianWord
		}
	}
	return view
}

func undercoverDescriptionActions(room *Room, player *Player) []aiplayer.LegalAction {
	word := undercoverWordForPlayer(room, player)
	if word == "" {
		return []aiplayer.LegalAction{
			{ID: "say:blank-follow", Label: "空白牌：跟随已有线索", Description: "根据最近发言接一个不露怯的侧面说法。speech 必须是最终发言，不能泛泛说常见、场景或特点。"},
			{ID: "say:blank-tone", Label: "空白牌：用语气试探", Description: "用谨慎语气给模糊但像真人的线索，不声称知道具体词。speech 必须自然短句。"},
			{ID: "say:blank-soft", Label: "空白牌：保守绕开核心", Description: "绕开核心名词，说一个安全的边缘联想。speech 不能说“我先看大家怎么描述”。"},
		}
	}
	return []aiplayer.LegalAction{
		{ID: "say:use", Label: "从用途或接触方式给线索", Description: "给一个关于使用方式、接触方式或参与动作的侧面线索。speech 必须是最终发言，不得说出底词，不得使用空话。"},
		{ID: "say:association", Label: "从相邻事物给线索", Description: "说它旁边常伴随的类别、动作或氛围，但不能点名底词。speech 必须像真人发言。"},
		{ID: "say:feeling", Label: "从感觉或语境给线索", Description: "给一个带个人感受的侧面线索，不能只说常见、场景、特点。speech 必须短而具体。"},
	}
}

func validUndercoverDescription(text string, word string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	runes := []rune(text)
	if len(runes) > 48 {
		text = string(runes[:48])
	}
	lowerText := strings.ToLower(text)
	for _, phrase := range []string{"生活里挺常见", "生活里很常见", "具体场景", "特点不能说得太细", "比较宽的范围", "看大家怎么描述", "常见但不好说"} {
		if strings.Contains(lowerText, strings.ToLower(phrase)) {
			return "", false
		}
	}
	word = strings.TrimSpace(word)
	if word != "" && strings.Contains(text, word) {
		return "", false
	}
	return text, true
}

func fallbackUndercoverDescription(actionID string) string {
	switch actionID {
	case "say:use":
		return "我想到的是它被用起来的样子。"
	case "say:association":
		return "我会先从它旁边的东西联想。"
	case "say:feeling":
		return "我对它的第一感觉比较明确。"
	case "say:blank-follow":
		return "我先顺着前面的方向说。"
	case "say:blank-tone":
		return "这个我不敢说太满。"
	default:
		return "我先给个边缘一点的线索。"
	}
}

func forbiddenPublicSpeech(word string) []string {
	word = strings.TrimSpace(word)
	if word == "" {
		return []string{}
	}
	return []string{word}
}

func undercoverVoteActions(room *Room, player *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if target.Alive && target.ID != player.ID {
			actions = append(actions, aiplayer.LegalAction{
				ID:          "vote:" + target.ID,
				Label:       target.Name,
				Description: fmt.Sprintf("投票给 %s", target.Name),
			})
		}
	}
	return actions
}
