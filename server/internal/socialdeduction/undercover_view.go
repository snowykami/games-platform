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

const undercoverPairsPerDomain = 500

type undercoverDomainSource struct {
	ID          string
	Name        string
	Description string
	Groups      [][]string
}

func undercoverDomainSources() []undercoverDomainSource {
	return []undercoverDomainSource{
		{
			ID:          "computing",
			Name:        "计算机与网络",
			Description: "协议、后端、前端、数据库、工程实践等技术词。",
			Groups: [][]string{
				{"TCP", "UDP", "HTTP", "HTTPS", "WebSocket", "gRPC", "DNS", "TLS", "QUIC", "IP", "ICMP", "NAT", "CDN", "代理", "网关", "负载均衡"},
				{"进程", "线程", "协程", "锁", "信号量", "队列", "事件循环", "调度器", "上下文", "内存池", "堆", "栈", "句柄", "缓存行", "死锁", "竞态"},
				{"React", "Vue", "Vite", "TypeScript", "Tailwind", "组件", "状态管理", "路由", "SSR", "Hydration", "虚拟列表", "表单校验", "响应式", "Hook", "Store", "构建产物"},
				{"PostgreSQL", "Redis", "索引", "事务", "锁表", "复制", "分片", "连接池", "慢查询", "执行计划", "主键", "外键", "唯一约束", "物化视图", "迁移", "回滚"},
				{"容器", "镜像", "Kubernetes", "Deployment", "Service", "Ingress", "ConfigMap", "Secret", "Sidecar", "滚动发布", "健康检查", "日志采集", "链路追踪", "告警", "灰度", "回滚策略"},
			},
		},
		{
			ID:          "academic",
			Name:        "学术与校园",
			Description: "论文、实验、课程、学术组织和校园生活。",
			Groups: [][]string{
				{"论文", "摘要", "引言", "综述", "方法", "实验", "数据集", "变量", "假设", "结论", "参考文献", "脚注", "附录", "查重", "盲审", "答辩"},
				{"数学", "物理", "化学", "生物", "经济学", "心理学", "社会学", "语言学", "历史学", "哲学", "统计学", "法学", "教育学", "传播学", "地理学", "天文学"},
				{"讲座", "研讨会", "课题组", "导师", "助教", "学分", "绩点", "选课", "补考", "实验报告", "开题", "中期检查", "会议海报", "口头报告", "奖学金", "交换生"},
				{"图书馆", "自习室", "实验室", "阶梯教室", "操场", "食堂", "宿舍", "校车", "社团", "学生会", "办公室", "档案馆", "报告厅", "机房", "黑板", "白板"},
				{"样本", "问卷", "访谈", "模型", "推导", "证明", "定理", "引理", "公式", "图表", "显著性", "置信区间", "回归", "聚类", "分类", "归纳"},
			},
		},
		{
			ID:          "daily",
			Name:        "日常生活",
			Description: "衣食住行、家居、通勤和生活习惯。",
			Groups: [][]string{
				{"咖啡", "奶茶", "豆浆", "果汁", "汽水", "矿泉水", "红茶", "绿茶", "酸奶", "啤酒", "热巧克力", "柠檬水", "椰汁", "苏打水", "拿铁", "美式"},
				{"火锅", "麻辣烫", "烧烤", "烤肉", "寿司", "披萨", "拉面", "炒饭", "汉堡", "沙拉", "饺子", "包子", "粥", "蛋糕", "冰淇淋", "薯条"},
				{"地铁", "公交车", "出租车", "共享单车", "高铁", "飞机", "轮船", "电梯", "扶梯", "停车场", "斑马线", "红绿灯", "候车厅", "安检口", "站台", "导航"},
				{"雨伞", "遮阳伞", "钥匙", "钱包", "背包", "行李箱", "耳机", "充电器", "水杯", "纸巾", "口罩", "镜子", "梳子", "拖鞋", "台灯", "闹钟"},
				{"厨房", "客厅", "卧室", "阳台", "书房", "冰箱", "洗衣机", "微波炉", "空调", "沙发", "窗帘", "地毯", "衣柜", "书架", "花瓶", "餐桌"},
			},
		},
		{
			ID:          "culture",
			Name:        "文艺与娱乐",
			Description: "影视、音乐、文学、展览和舞台体验。",
			Groups: [][]string{
				{"电影", "电视剧", "纪录片", "短片", "动画", "综艺", "预告片", "片尾曲", "配音", "字幕", "镜头", "剪辑", "导演", "编剧", "演员", "票房"},
				{"小说", "散文", "诗歌", "漫画", "剧本", "传记", "书评", "封面", "章节", "伏笔", "叙事", "对白", "角色", "世界观", "纸书", "电子书"},
				{"钢琴", "吉他", "小提琴", "鼓", "贝斯", "合唱", "独唱", "旋律", "节拍", "和声", "编曲", "录音棚", "演唱会", "音乐节", "耳返", "专辑"},
				{"博物馆", "美术馆", "画展", "摄影展", "装置艺术", "雕塑", "油画", "水彩", "素描", "书法", "海报", "策展", "讲解", "展柜", "门票", "纪念品"},
				{"话剧", "音乐剧", "舞蹈", "相声", "脱口秀", "魔术", "舞台灯", "幕布", "剧场", "观众席", "掌声", "返场", "彩排", "台词", "布景", "道具"},
			},
		},
		{
			ID:          "games",
			Name:        "游戏与桌游",
			Description: "电子游戏、桌游、牌局、竞技和玩家行为。",
			Groups: [][]string{
				{"主线任务", "支线任务", "副本", "Boss", "小怪", "掉落", "装备", "技能树", "等级", "经验值", "成就", "存档", "读档", "传送点", "补给", "复活点"},
				{"象棋", "五子棋", "围棋", "麻将", "Uno", "扑克", "狼人杀", "阿瓦隆", "谁是卧底", "剧本杀", "桌游卡牌", "骰子", "棋盘", "回合", "胜负", "淘汰"},
				{"开局", "中盘", "残局", "先手", "后手", "连招", "走位", "控场", "拉扯", "防守", "进攻", "偷袭", "反打", "资源点", "视野", "节奏"},
				{"射手", "法师", "坦克", "辅助", "刺客", "治疗", "召唤物", "护盾", "暴击", "冷却", "蓝量", "血条", "伤害", "控制", "增益", "减益"},
				{"手柄", "键鼠", "摇杆", "触屏", "匹配", "排位", "天梯", "房间号", "观战", "语音", "战绩", "皮肤", "赛季", "活动", "补丁", "平衡性"},
			},
		},
		{
			ID:          "business",
			Name:        "商业与职场",
			Description: "公司、产品、市场、运营、财务和办公场景。",
			Groups: [][]string{
				{"产品经理", "设计师", "工程师", "运营", "销售", "客服", "财务", "法务", "人事", "实习生", "主管", "总监", "创始人", "顾问", "供应商", "客户"},
				{"需求", "排期", "里程碑", "迭代", "评审", "复盘", "会议纪要", "路线图", "看板", "优先级", "风险", "验收", "交付", "延期", "上线", "回滚"},
				{"品牌", "广告", "渠道", "转化率", "留存", "增长", "用户画像", "竞品", "定价", "活动页", "优惠券", "会员", "私域", "社群", "直播带货", "投放"},
				{"收入", "成本", "利润", "预算", "发票", "报销", "合同", "报价单", "采购", "库存", "现金流", "融资", "估值", "审计", "税务", "流水"},
				{"办公室", "工位", "会议室", "白板笔", "工牌", "打卡", "日报", "周报", "邮件", "飞书", "钉钉", "简历", "面试", "入职", "离职", "团建"},
			},
		},
		{
			ID:          "science",
			Name:        "自然科学",
			Description: "天文、地球、生命、材料和实验观察。",
			Groups: [][]string{
				{"恒星", "行星", "卫星", "彗星", "星云", "黑洞", "银河", "星座", "引力", "轨道", "望远镜", "火箭", "探测器", "宇航服", "空间站", "流星雨"},
				{"细胞", "基因", "蛋白质", "酶", "病毒", "细菌", "抗体", "疫苗", "神经元", "激素", "生态", "种群", "进化", "光合作用", "呼吸作用", "显微镜"},
				{"原子", "分子", "离子", "晶体", "金属", "陶瓷", "塑料", "玻璃", "催化剂", "溶液", "沉淀", "酸碱", "氧化", "还原", "电解", "燃烧"},
				{"岩石", "矿物", "火山", "地震", "板块", "河流", "冰川", "沙漠", "海洋", "季风", "台风", "云层", "降雨", "气压", "温度", "湿度"},
				{"力", "速度", "加速度", "能量", "功率", "电流", "电压", "磁场", "光谱", "折射", "反射", "干涉", "波长", "频率", "热量", "熵"},
			},
		},
		{
			ID:          "travel",
			Name:        "地理与旅行",
			Description: "城市、自然景观、交通住宿和旅行体验。",
			Groups: [][]string{
				{"北京", "上海", "广州", "深圳", "杭州", "成都", "重庆", "西安", "南京", "武汉", "苏州", "厦门", "青岛", "长沙", "昆明", "哈尔滨"},
				{"海滩", "雪山", "草原", "森林", "湖泊", "峡谷", "瀑布", "温泉", "古镇", "夜市", "步行街", "观景台", "灯塔", "码头", "露营地", "滑雪场"},
				{"护照", "签证", "机票", "登机牌", "酒店", "民宿", "青旅", "行程单", "攻略", "地图", "导游", "租车", "换乘", "托运行李", "免税店", "纪念章"},
				{"早市", "小吃街", "咖啡馆", "餐厅", "酒吧", "书店", "剧院", "公园", "广场", "博物馆", "美术馆", "游乐园", "寺庙", "教堂", "城堡", "老街"},
				{"日出", "日落", "雨季", "旱季", "极光", "星空", "云海", "花海", "潮汐", "航拍", "徒步", "骑行", "潜水", "冲浪", "摄影", "路书"},
			},
		},
	}
}

func undercoverPairsForDomain(id string) []UndercoverWordPair {
	for _, source := range undercoverDomainSources() {
		if source.ID == id {
			return generateUndercoverPairs(source)
		}
	}
	return nil
}

func generateUndercoverPairs(source undercoverDomainSource) []UndercoverWordPair {
	pairs := make([]UndercoverWordPair, 0, undercoverPairsPerDomain)
	for groupIndex, group := range source.Groups {
		for left := 0; left < len(group); left++ {
			for right := left + 1; right < len(group); right++ {
				pairs = append(pairs, UndercoverWordPair{
					ID:             fmt.Sprintf("%s-%02d-%03d", source.ID, groupIndex+1, len(pairs)+1),
					CivilianWord:   group[left],
					UndercoverWord: group[right],
					Category:       source.Name,
				})
				if len(pairs) >= undercoverPairsPerDomain {
					return pairs
				}
			}
		}
	}
	return pairs
}

func undercoverPresets() []UndercoverPreset {
	presets := make([]UndercoverPreset, 0, len(undercoverDomainSources()))
	for _, source := range undercoverDomainSources() {
		presets = append(presets, UndercoverPreset{
			ID:          source.ID,
			Name:        source.Name,
			Description: source.Description,
			PairCount:   len(undercoverPairsForDomain(source.ID)),
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
	pairs := []UndercoverWordPair{}
	for _, id := range normalizeUndercoverDomainIDs(domainIDs) {
		pairs = append(pairs, undercoverPairsForDomain(id)...)
	}
	if len(pairs) == 0 {
		pairs = undercoverPairsForDomain(defaultUndercoverPresetID())
	}
	return pairs[rand.IntN(len(pairs))]
}

func undercoverTotalPairCount() int {
	total := 0
	for _, source := range undercoverDomainSources() {
		total += len(undercoverPairsForDomain(source.ID))
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
