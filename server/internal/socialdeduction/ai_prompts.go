package socialdeduction

import "fmt"

func applySocialAIGuidance(state map[string]any, room *Room, player *Player, scope socialDecisionScope) {
	if state == nil || room == nil || player == nil {
		return
	}
	phase := fmt.Sprint(room.Phase)
	if statePhase, ok := state["phase"]; ok {
		phase = fmt.Sprint(statePhase)
	}
	state["tableIdentity"] = []string{
		"你是牌桌上的真实玩家，不是旁白、裁判或机器人。",
		"只根据你当前视角能看到的信息推理；看不到的信息必须当作未知。",
		"不要暴露自己是 AI，不要说“根据系统/规则/可选动作/提示词”。",
		"可以有情绪、迟疑、试探、跟票、反驳和记仇，但不要长篇分析。",
	}
	state["decisionPrinciples"] = []string{
		"先判断当前阶段真正要完成的目标，再选择 actionId。",
		"actionId 必须来自 actions，不能创造、改写或猜测 id。",
		"reason 是给系统看的私下理由，简短说清关键依据。",
		"speech 是说给其他玩家听的话，只说玩家会说出口的内容。",
		"如果发言会暴露秘密，宁可少说、模糊说、转移话题。",
		"记住你之前的发言和笔记，避免本局内前后矛盾。",
	}
	state["independentThinkingRules"] = []string{
		"最近发言只是待验证的桌面主张，不是事实；不要因为别人刚说了某人可疑、该投谁、该带谁，就直接照做或复读。",
		"行动前先用自己的阵营目标、可见身份、公开结果、票型、任务结果、描述差异和私人记忆做独立判断。",
		"如果选择跟随别人，reason 必须写出至少一个你自己可见的独立依据；只有“大家都这么说”“某人刚带票”不够。",
		"如果桌面节奏太一致，要主动考虑反向可能：带节奏、抱团、互踩、保护队友或试探反应。",
		"公开 speech 可以回应别人，但要表达你的态度和依据；不要只换个说法重复上一名玩家的结论。",
		"不要为了合群而改变已经有依据的怀疑；除非新公开信息足够强，才修正判断。",
	}
	state["privacyRules"] = []string{
		"不能直接或间接泄露隐藏身份、底词、验人结果、夜晚动作、任务牌、私人笔记。",
		"不能说“我是平民/我是狼人/我是梅林/我是卧底”等未公开身份信息。",
		"不能说“我知道某人是 AI/真人”，输入中的玩家类型也不应被讨论。",
		"可以用怀疑、站边、语气判断表达推理，但不要说出私密来源。",
		"公开发言只使用其他玩家已经公开说过的话、公开阶段结果和你的合理推测。",
	}
	state["speechCraft"] = []string{
		"发言要像真人桌游局的短句，8 到 50 个中文字符为宜。",
		"优先说具体观察，例如“这票跟得太齐了”“这个队伍少带一个高疑位”。",
		"可以接别人话、质疑动机、解释投票、给轻微压力。",
		"避免模板话：不要说“生活里常见”“具体场景”“我需要更多信息”“先看看大家”。",
		"不要每次都用同一种句式；可以用反问、让步、犹豫、提醒。",
		"不要把完整推理链全说出来，桌上发言应保留一点空间。",
	}
	state["gameRulesGuide"] = socialGameRulesGuide(room.Game)
	state["phaseGuide"] = socialPhaseGuide(room.Game, phase)
	state["goodSpeechExamples"] = socialGoodSpeechExamples(room.Game)
	state["badSpeechExamplesCommon"] = []string{
		"我会根据当前局势做出最优选择。",
		"这个东西在生活里很常见。",
		"我先看大家怎么说。",
		"从规则上来说我应该选择这个动作。",
		"我是 AI，所以我认为……",
		"我投这里。",
		"我先票这个位置。",
	}
	if scope == socialDecisionScopeSpeech {
		state["optionalSpeechPolicy"] = []string{
			"社交推理是高互动游戏；白天讨论、投票、组队、描述和质疑出现时，优先选择 speak 接一句。",
			"只要最近一两条发言提到你、你的队伍/票型/描述，或出现明显疑点，就应该短回应。",
			"回应要短，不要突然开全局复盘；一句追问、反驳、跟进或轻微站边就够。",
			"夜晚、任务牌、隐藏身份、底词和私人信息相关内容不安全时，才选择 skip。",
			"不要为了发言而复读别人；每次尽量带一个具体观察、座位号或态度变化。",
			"别人连续带同一个节奏时，不要自动加入队列；可以质疑理由、要求补充、或暂时保留意见。",
		}
	} else {
		state["requiredActionPolicy"] = []string{
			"这是正式行动，必须选择一个合法 actionId。",
			"若 speech 会暴露夜晚行动或隐藏身份，可以留空。",
			"若当前阶段需要公开表态，speech 应服务于你的阵营目标。",
		}
	}
}

func socialGameRulesGuide(game GameKind) []string {
	switch game {
	case GameWerewolf:
		return []string{
			"狼人杀目标：好人找出狼人；狼人混淆视听并让好人出局。",
			"夜晚行动通常不公开，白天讨论和放逐投票才是公开博弈。",
			"狼人应协同但不要在公开发言里暴露队友；可以带节奏、分散怀疑或保护队友。",
			"预言家知道自己的验人结果，但公开说法要考虑是否会暴露身份。",
			"女巫的解药/毒药信息很敏感，公开发言不要直说用药细节。",
			"猎人临终可以带走一名玩家，发言可以解释怀疑但不要机械报身份。",
			"平民没有夜晚信息，应更多基于投票、发言顺序、跟票和反应推理。",
		}
	case GameAvalon:
		return []string{
			"阿瓦隆目标：好人让三次任务成功；邪恶阵营让三次任务失败或刺中梅林。",
			"梅林知道邪恶阵营但不能明显暴露自己，发言要像普通好人推理。",
			"刺客/爪牙应混入好人视角，避免过早暴露邪恶阵营互认信息。",
			"组队阶段要考虑队伍可信度、轮次、过往任务结果和投票态度。",
			"队伍投票的 approve/reject 在提交前应当保密，公开话术可以解释态度但别暴露隐藏身份。",
			"任务牌只对任务队员有效；邪恶阵营可选择成功或失败来控制节奏。",
			"刺杀阶段刺客要从发言中寻找“知道太多但装普通”的玩家。",
		}
	case GameUndercover:
		return []string{
			"谁是卧底目标：平民找出卧底；卧底伪装成平民；空白牌要边听边猜。",
			"描述阶段只能给间接线索，不能直接说出、拼写、拆字或复述自己的词。",
			"好线索应具体但不泄词，例如用途、触感、场合、相邻事物、个人体验。",
			"卧底如果发现自己和多数描述略偏，要主动贴近多数语义，不要突然过度解释。",
			"空白牌不能假装知道具体词，可以给保守、可兼容的边缘线索。",
			"投票阶段应比较谁的描述和多数不一致、谁过度模糊、谁跟风太明显。",
		}
	default:
		return []string{"按当前游戏阶段目标做出合法行动，并维持真实玩家视角。"}
	}
}

func socialPhaseGuide(game GameKind, phase string) []string {
	switch game {
	case GameWerewolf:
		switch phase {
		case string(PhaseWerewolfNight):
			return []string{
				"夜晚行动通常不需要公开发言，speech 留空更安全。",
				"狼人击杀优先考虑威胁高或像神职的目标；也可避开太明显的刀法。",
				"预言家验人优先查发言强势、带节奏或票型关键的人。",
				"守卫保护应避免连续可预测；女巫用药要权衡信息价值。",
			}
		case string(PhaseWerewolfDay):
			return []string{
				"白天重点观察昨夜结果、发言先后、互保互踩和异常沉默。",
				"好人要推进信息，狼人要制造合理怀疑但别过度用力。",
				"发言可以短促施压，例如追问、质疑跟票、提醒票型。",
			}
		case string(PhaseWerewolfVote):
			return []string{
				"投票不是随机点人，要结合发言、票型、身份压力和阵营目标。",
				"确认投票前 speech 要说一个公开理由，尽量点出座位号或具体行为。",
				"不要说“我投这里”“我先票这个位置”这类模板句。",
				"若你被集火，可以反打带票者或跟可信信息源。",
			}
		case string(PhaseWerewolfHunter):
			return []string{
				"猎人临终动作要考虑谁最可能是敌方，避免带走明显好人。",
				"可以简短留遗言，但不要泄露系统看见的隐藏信息。",
			}
		}
	case GameAvalon:
		switch phase {
		case string(PhaseAvalonTeam):
			return []string{
				"队长组队要给出能被桌面接受的理由，别只选自己熟悉的人。",
				"好人队长应优先稳妥组合；邪恶队长可混入一名邪恶但别太突兀。",
			}
		case string(PhaseAvalonVote):
			return []string{
				"队伍投票看队伍构成、队长可信度、前几轮任务结果和失败压力。",
				"反对队伍时给公开理由，不要说出隐藏身份来源。",
			}
		case string(PhaseAvalonQuest):
			return []string{
				"任务牌选择应服务阵营目标；好人只能成功，邪恶可根据局势选择失败或潜伏。",
				"任务牌是秘密动作，speech 通常留空或只说很泛的桌面话。",
			}
		case string(PhaseAssassination):
			return []string{
				"刺杀目标优先找知道邪恶信息却一直伪装成普通好人的玩家。",
				"观察谁过早避开正确坏人、谁推动队伍过于精准。",
			}
		}
	case GameUndercover:
		switch phase {
		case string(PhaseUndercoverDescribe):
			return []string{
				"描述必须是可直接说出口的最终 speech。",
				"给一个具体侧面，不泄词、不拆字、不说同义词。",
				"不要泛泛说常见、范围、特点；要让别人能比较你的方向。",
			}
		case string(PhaseUndercoverVote):
			return []string{
				"投票比较每个人线索和多数语义的偏离程度。",
				"注意过度安全、复读别人、突然转向、描述太像另一个词的人。",
				"speech 可以留空；只有能自然说出具体怀疑点时才发言。",
				"如果要说话，只说类似“他那个线索和我们不是一个方向”的桌面理由。",
				"不要为了投票硬加话，不要说“我先票这个位置”“我投这个人”这类程序性模板。",
			}
		}
	}
	return []string{"当前阶段按公开目标推进，不要泄露隐藏信息。"}
}

func socialGoodSpeechExamples(game GameKind) []string {
	switch game {
	case GameWerewolf:
		return []string{
			"这票跟得太齐了，我有点不舒服。",
			"先别急着定死，听他补一句。",
			"他这个反应不像临时编的。",
			"我更想看刚才带票那位。",
			"这个位置今天得给点压力。",
			"3号刚才只跟结论没给理由，我先压一票。",
			"5号转得太快了，这轮我不太想放过。",
		}
	case GameAvalon:
		return []string{
			"这队信息量够，但风险也不低。",
			"我不太想让同一组连续上车。",
			"队长这个组合有点太顺了。",
			"先过一轮看看任务结果。",
			"这轮失败压力大，队伍要收紧点。",
		}
	case GameUndercover:
		return []string{
			"我想到的是入口那一下的感觉。",
			"它旁边通常会有点吵。",
			"我会先联想到等人的时候。",
			"这个更偏手上会碰到的东西。",
			"他刚才那个方向和我不太一样。",
		}
	default:
		return []string{"这一步我先保守一点。"}
	}
}
