package games

import "github.com/snowykami/games-platform/server/internal/i18n"

type Definition struct {
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	MinPlayers     int      `json:"minPlayers"`
	MaxPlayers     int      `json:"maxPlayers"`
	SupportsOnline bool     `json:"supportsOnline"`
	SupportsLocal  bool     `json:"supportsLocal"`
	Status         string   `json:"status"`
	Tags           []string `json:"tags"`
}

func List() []Definition {
	return ListForLocale(i18n.LocaleEN)
}

func ListForLocale(locale i18n.Locale) []Definition {
	return []Definition{
		{
			Slug:           "uno",
			Title:          gameText(locale, "uno.title"),
			Description:    gameText(locale, "uno.description"),
			MinPlayers:     2,
			MaxPlayers:     4,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "available",
			Tags:           gameTags(locale, "uno.tags"),
		},
		{
			Slug:           "gomoku",
			Title:          gameText(locale, "gomoku.title"),
			Description:    gameText(locale, "gomoku.description"),
			MinPlayers:     2,
			MaxPlayers:     2,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "available",
			Tags:           gameTags(locale, "gomoku.tags"),
		},
		{
			Slug:           "xiangqi",
			Title:          gameText(locale, "xiangqi.title"),
			Description:    gameText(locale, "xiangqi.description"),
			MinPlayers:     2,
			MaxPlayers:     2,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "available",
			Tags:           gameTags(locale, "xiangqi.tags"),
		},
		{
			Slug:           "mahjong",
			Title:          gameText(locale, "mahjong.title"),
			Description:    gameText(locale, "mahjong.description"),
			MinPlayers:     4,
			MaxPlayers:     4,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "available",
			Tags:           gameTags(locale, "mahjong.tags"),
		},
	}
}

func gameText(locale i18n.Locale, key string) string {
	text := map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {
			"uno.title":           "Uno",
			"uno.description":     "Lightweight Uno with server rooms, turn flow, rule validation, and extensible card rules.",
			"gomoku.title":        "Gomoku",
			"gomoku.description":  "Classic five-in-a-row with room links, live sync, and rule-based AI.",
			"xiangqi.title":       "Xiangqi",
			"xiangqi.description": "Server-authoritative Chinese chess rooms with AI, check detection, match results, and move records.",
			"mahjong.title":       "Mahjong",
			"mahjong.description": "Server-room Chinese Official Mahjong with live sync, hidden hands, AI seats, and an extensible ruleset.",
		},
		i18n.LocaleZH: {
			"uno.title":           "Uno",
			"uno.description":     "轻量 Uno 房间，验证卡牌规则、回合流转、服务端校验和扩展玩法。",
			"gomoku.title":        "五子棋",
			"gomoku.description":  "经典双人五连棋，支持链接开房、观战同步与规则 AI 对弈。",
			"xiangqi.title":       "象棋",
			"xiangqi.description": "服务端象棋房间，支持链接开局、AI 对局、将军判定、胜负和棋谱。",
			"mahjong.title":       "麻将",
			"mahjong.description": "服务端国标麻将房间，支持链接开局、隐藏手牌、AI 补位和可扩展规则集。",
		},
	}
	if value := text[locale][key]; value != "" {
		return value
	}
	return text[i18n.LocaleEN][key]
}

func gameTags(locale i18n.Locale, key string) []string {
	tags := map[i18n.Locale]map[string][]string{
		i18n.LocaleEN: {
			"uno.tags":     {"Cards", "Turn-based", "Prototype"},
			"gomoku.tags":  {"Board", "Two-player", "Five-in-row"},
			"xiangqi.tags": {"Board", "Strategy", "AI"},
			"mahjong.tags": {"Table", "Official", "Four-player"},
		},
		i18n.LocaleZH: {
			"uno.tags":     {"卡牌", "回合制", "首个原型"},
			"gomoku.tags":  {"棋类", "双人", "五连"},
			"xiangqi.tags": {"棋类", "策略", "AI"},
			"mahjong.tags": {"牌桌", "国标", "四人"},
		},
	}
	if value := tags[locale][key]; len(value) > 0 {
		return value
	}
	return tags[i18n.LocaleEN][key]
}
