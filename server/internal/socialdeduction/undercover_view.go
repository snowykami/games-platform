package socialdeduction

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func applyDefaultUndercoverConfig(room *Room) {
	if len(room.Undercover.DomainIDs) == 0 {
		room.Undercover.DomainIDs = []string{defaultUndercoverPresetID()}
	}
	room.Undercover.DomainIDs = normalizeUndercoverDomainIDs(room.Undercover.DomainIDs)
	room.Undercover.PresetID = room.Undercover.DomainIDs[0]
	room.Undercover.Presets = undercoverPresets()
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]UndercoverVoteIntent{}
}

func defaultUndercoverPresetID() string {
	return "computing"
}

func undercoverPairsForDomain(id string) []UndercoverWordPair {
	for _, source := range undercoverDomainSources() {
		if source.ID == id {
			return allUndercoverPairsForDomain(source)
		}
	}
	return nil
}

func allUndercoverPairsForDomain(source undercoverDomainSource) []UndercoverWordPair {
	pairs := []UndercoverWordPair{}
	for groupIndex, group := range source.Groups {
		for civilianIndex, civilianWord := range group {
			for undercoverIndex, undercoverWord := range group {
				if civilianIndex == undercoverIndex {
					continue
				}
				pairs = append(pairs, UndercoverWordPair{
					ID:             fmt.Sprintf("%s-%03d-%02d-%02d", source.ID, groupIndex+1, civilianIndex+1, undercoverIndex+1),
					CivilianWord:   civilianWord.Text,
					UndercoverWord: undercoverWord.Text,
					Category:       source.Name,
					CivilianHint:   civilianWord.Hint,
					UndercoverHint: undercoverWord.Hint,
				})
			}
		}
	}
	return pairs
}

func undercoverGroupsForDomain(id string) []undercoverWordGroup {
	for _, source := range undercoverDomainSources() {
		if source.ID != id {
			continue
		}
		groups := make([]undercoverWordGroup, 0, len(source.Groups))
		for index, words := range source.Groups {
			groups = append(groups, undercoverWordGroup{
				DomainID:   source.ID,
				Category:   source.Name,
				GroupIndex: index + 1,
				Words:      words,
			})
		}
		return groups
	}
	return nil
}

func undercoverGroupCountForDomain(id string) int {
	return len(undercoverGroupsForDomain(id))
}

func undercoverPresets() []UndercoverPreset {
	presets := make([]UndercoverPreset, 0, len(undercoverDomainSources()))
	for _, source := range undercoverDomainSources() {
		presets = append(presets, UndercoverPreset{
			ID:          source.ID,
			Name:        source.Name,
			Description: source.Description,
			PairCount:   undercoverGroupCountForDomain(source.ID),
		})
	}
	return presets
}

func undercoverPresetExists(id string) bool {
	return undercoverDomainExists(id)
}

func undercoverDomainExists(id string) bool {
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

func undercoverDomainNames(ids []string) string {
	return strings.Join(undercoverDomainNameList(ids), "、")
}

func undercoverDomainNameList(ids []string) []string {
	names := []string{}
	for _, id := range normalizeUndercoverDomainIDs(ids) {
		names = append(names, undercoverPresetName(id))
	}
	return names
}

func normalizeUndercoverDomainIDs(ids []string) []string {
	seen := map[string]bool{}
	next := []string{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] || !undercoverDomainExists(id) {
			continue
		}
		seen[id] = true
		next = append(next, id)
	}
	if len(next) == 0 {
		return []string{defaultUndercoverPresetID()}
	}
	return next
}

func chooseUndercoverPair(domainIDs []string) UndercoverWordPair {
	groups := []undercoverWordGroup{}
	for _, id := range normalizeUndercoverDomainIDs(domainIDs) {
		groups = append(groups, undercoverGroupsForDomain(id)...)
	}
	if len(groups) == 0 {
		groups = undercoverGroupsForDomain(defaultUndercoverPresetID())
	}
	return chooseUndercoverPairFromGroup(groups[rand.IntN(len(groups))])
}

func chooseUndercoverPairFromGroup(group undercoverWordGroup) UndercoverWordPair {
	civilianIndex := rand.IntN(len(group.Words))
	undercoverIndex := rand.IntN(len(group.Words) - 1)
	if undercoverIndex >= civilianIndex {
		undercoverIndex++
	}
	return UndercoverWordPair{
		ID:             fmt.Sprintf("%s-%03d-%02d-%02d", group.DomainID, group.GroupIndex, civilianIndex+1, undercoverIndex+1),
		CivilianWord:   group.Words[civilianIndex].Text,
		UndercoverWord: group.Words[undercoverIndex].Text,
		Category:       group.Category,
		CivilianHint:   group.Words[civilianIndex].Hint,
		UndercoverHint: group.Words[undercoverIndex].Hint,
	}
}

func undercoverTotalGroupCount() int {
	total := 0
	for _, source := range undercoverDomainSources() {
		total += undercoverGroupCountForDomain(source.ID)
	}
	return total
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

func undercoverWordHintForPlayer(room *Room, player *Player) string {
	switch player.Role {
	case RoleUndercover:
		return room.Undercover.WordPair.UndercoverHint
	case RoleBlank:
		return ""
	default:
		return room.Undercover.WordPair.CivilianHint
	}
}

func undercoverViewForViewer(room *Room, viewer *Player) UndercoverView {
	view := UndercoverView{
		Round:            room.Undercover.Round,
		PresetID:         room.Undercover.PresetID,
		DomainIDs:        append([]string{}, room.Undercover.DomainIDs...),
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
		if viewer != nil {
			view.YourWord = undercoverWordForPlayer(room, viewer)
		}
		return view
	}
	if viewer != nil {
		view.WordPair = UndercoverWordPair{ID: room.Undercover.WordPair.ID, Category: room.Undercover.WordPair.Category}
		view.YourWord = undercoverWordForPlayer(room, viewer)
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
				Label:       fmt.Sprintf("投票给 %d号 %s", target.Seat+1, target.Name),
				Description: fmt.Sprintf("座位 %d 的存活玩家，AI action id 会映射为 seat_%d。", target.Seat+1, target.Seat+1),
			})
		}
	}
	return actions
}
