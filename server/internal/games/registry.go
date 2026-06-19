package games

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
	return []Definition{
		{
			Slug:           "uno",
			Title:          "Uno",
			Description:    "轻量本地 Uno 原型，先验证卡牌规则、回合流转和页面结构。",
			MinPlayers:     2,
			MaxPlayers:     4,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "available",
			Tags:           []string{"卡牌", "回合制", "首个原型"},
		},
		{
			Slug:           "gomoku",
			Title:          "五子棋",
			Description:    "经典双人棋类，适合作为首个完整联机规则验证目标。",
			MinPlayers:     2,
			MaxPlayers:     2,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "coming-soon",
			Tags:           []string{"棋类", "双人"},
		},
		{
			Slug:           "xiangqi",
			Title:          "象棋",
			Description:    "中国象棋，后续通过独立适配器接入。",
			MinPlayers:     2,
			MaxPlayers:     2,
			SupportsOnline: true,
			SupportsLocal:  true,
			Status:         "coming-soon",
			Tags:           []string{"棋类", "策略"},
		},
		{
			Slug:           "mahjong",
			Title:          "麻将",
			Description:    "多人牌桌游戏，规则复杂度较高，后续分阶段实现。",
			MinPlayers:     4,
			MaxPlayers:     4,
			SupportsOnline: true,
			SupportsLocal:  false,
			Status:         "coming-soon",
			Tags:           []string{"牌桌", "多人"},
		},
	}
}
