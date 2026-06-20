package aiplayer

import "fmt"

type BotProfile struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
	SpeechStyle string `json:"speechStyle"`
}

var sharedBotProfiles = []BotProfile{
	{Name: "北风", Personality: "稳健型玩家，喜欢先控节奏，不轻易把关键牌交出去。", SpeechStyle: "说话短，像熟人桌上的冷静提醒，例如“先稳一下”“这手还行”。"},
	{Name: "南星", Personality: "进攻型玩家，倾向尽快压缩手牌，看到机会会果断推进。", SpeechStyle: "语气直接轻快，例如“我先走一张”“节奏提一下”。"},
	{Name: "阿澈", Personality: "观察型玩家，会盯住对手资源，偏好用颜色和节奏牵制别人。", SpeechStyle: "发言像边打边观察，例如“你这手牌有点多啊”“颜色我先改一下”。"},
	{Name: "小满", Personality: "随性但不乱打，喜欢用简单选择制造一点意外节奏。", SpeechStyle: "偶尔吐槽自己，例如“先这么来吧”“这牌有点抽象”。"},
	{Name: "白川", Personality: "防守型玩家，优先堵住明显威胁，再寻找反击窗口。", SpeechStyle: "语气温和克制，例如“这个得挡一下”“别太快结束”。"},
	{Name: "星野", Personality: "均衡型玩家，会在进攻和保守之间切换，避免单一路线。", SpeechStyle: "像普通玩家复盘当下，例如“这张应该可以”“看一下后面怎么接”。"},
	{Name: "青灯", Personality: "耐心型玩家，不急着打爆发，更重视长期手牌结构。", SpeechStyle: "发言偏平实，例如“先留一手”“慢慢打”。"},
	{Name: "赤羽", Personality: "压迫型玩家，喜欢制造麻烦，优先选择能打断对手节奏的动作。", SpeechStyle: "语气有点自信但不夸张，例如“给你点压力”“这张刚好”。"},
	{Name: "玄霜", Personality: "谨慎型玩家，重视安全落子和风险控制，很少主动冒险。", SpeechStyle: "发言简短沉稳，例如“不能贪”“这步保守一点”。"},
	{Name: "西楼", Personality: "计算型玩家，会优先处理当前最明确的收益。", SpeechStyle: "像认真打牌的人，例如“先按收益走”“这手不亏”。"},
	{Name: "云雀", Personality: "灵活型玩家，愿意根据桌面变化快速换策略。", SpeechStyle: "语气自然，例如“那我换个方向”“这局挺有意思”。"},
	{Name: "松间", Personality: "佛系型玩家，节奏稳定，不容易被挑衅影响判断。", SpeechStyle: "发言放松，例如“问题不大”“先随一张”。"},
	{Name: "临川", Personality: "读牌型玩家，喜欢根据别人剩余资源推测下一步。", SpeechStyle: "像桌边聊天，例如“你是不是快没牌了”“我感觉不太妙”。"},
	{Name: "晴岚", Personality: "机会型玩家，平时稳住，一旦看到胜势会迅速收口。", SpeechStyle: "发言干净自然，例如“有机会了”“这张舒服”。"},
}

func NextProfile(usedNames map[string]bool) BotProfile {
	for _, profile := range sharedBotProfiles {
		if !usedNames[profile.Name] {
			return profile
		}
	}
	index := len(usedNames) + 1
	return BotProfile{
		Name:        fmt.Sprintf("AI %d", index),
		Personality: "规则型玩家，会根据当前游戏局面选择合法动作。",
		SpeechStyle: "发言简短自然，只在有必要时说一句普通玩家会说的话。",
	}
}

func Profiles() []BotProfile {
	return append([]BotProfile{}, sharedBotProfiles...)
}
