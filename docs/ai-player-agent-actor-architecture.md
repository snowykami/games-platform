# AI 玩家 Agent Actor 重构计划

## 目标

当前 AI 玩家由房间调度器按需调用，例如 `RunNextAI` 执行正式动作，`RunAISpeech` 执行主动发言。这个模型简单，但 AI 更像一段被动函数，不像持续存在的玩家。它也容易遇到两个问题：

- 正式动作和主动发言并发触发，互相更新房间版本，导致 stale decision。
- 社交推理游戏缺少长期个人状态，AI 容易忘记自己刚才说过什么、怀疑谁、为什么投票。

新架构目标是让每个 AI 玩家表现为独立玩家：

- 每个 AI 玩家有自己的 goroutine、记忆、persona、LLM 会话和节奏。
- AI 可以响应事件，也可以在轮到自己时主动决策。
- 房间仍然保持服务端权威，AI 只能提交意图，不能直接改权威状态。
- 支持不同游戏复用同一套 AI Agent 管线。

本项目当前尚未正式上线，因此本计划按 **重构优先** 执行，不以保留旧内部接口为目标。可以破坏现有 manager / hub / AI 调度内部结构，只要最终 HTTP/WebSocket 对前端的行为清晰、实现更简单、更一致即可。避免为了短期兼容保留双套调度逻辑，避免让旧 `RunNextAI` / `RunAISpeech` 和新 Actor 模型长期并存。

## 核心结论

推荐采用 **Room Actor + AI Player Agent** 模型，而不是让每个 AI goroutine 直接修改房间。

```text
Human / WebSocket / HTTP
        |
        v
RoomActor goroutine  <---->  AIPlayerAgent goroutine x N
        |
        v
authoritative room state + broadcast
```

关键边界：

- `RoomActor` 是唯一能修改房间状态的组件。
- `AIPlayerAgent` 是玩家脑子，只能观察事件、维护记忆、提交 `PlayerIntent`。
- `GameAdapter` 负责把具体游戏规则暴露给 RoomActor 和 Agent。
- LLM 是不可信外部依赖，所有返回必须重新走合法动作校验。

## 重构原则

- 优先清理旧架构，而不是包兼容层。迁移一个游戏时，应将该游戏切到 RoomActor 调度，并删除该游戏旧的 AI 调度入口。
- 允许调整内部 Go 类型、manager 方法和 Hub 调度方式。前端 API 可以在必要时同步调整。
- 对外行为以当前产品目标为准，而不是以旧实现为准。
- 每个阶段结束时保持可运行、可测试，不留下半迁移状态。
- 不为旧 AI 调度新增功能；新能力只进入 Agent Actor 架构。
- 新架构中的公共抽象必须服务真实复用，避免过早做复杂框架。
- 重构时同步治理项目结构和文件行数。不要把 Actor、新 AI context、HTTP、Hub、规则、前端交互继续堆进既有大文件；前后端都应按职责拆分，迁移到新结构后删除旧路径。

### 结构治理目标

Actor 重构不是只换调度模型，也要把当前逐渐膨胀的文件拆开。目标是让一个文件只承载一个清晰职责，后续新增游戏不会继续复制粘贴大块 manager/page 代码。

后端建议拆分方向：

- `manager.go` 只保留房间生命周期和少量协调逻辑。
- `room_actor.go` / `actor.go` 放 RoomActor 主循环、事件分发和关闭流程。
- `agent.go` / `memory.go` / `prompts.go` 放 AI Agent、记忆和 prompt 构建。
- `rules_*.go` 放纯规则、阶段推进、胜负判定和合法动作生成。
- `view_*.go` 或 `public_view.go` 放 viewer 视图、私有视图和 AI context 投影。
- `ai_*.go` 放每个游戏的 LLM state、action alias、fallback 和 AI 专属测试。
- `http.go` 只保留路由注册、请求解码、响应编码，不写复杂游戏规则。
- `hub.go` 只处理订阅、广播和连接生命周期，不直接做游戏结算。

前端建议拆分方向：

- slug 页面只做路由和数据装配，不直接塞完整游戏 UI。
- `Lobby`、`RoomHeader`、`PlayerList`、`GameTable`、`ActionPanel`、`SpeechBubbleLayer`、`RulesModal` 等按职责拆组件。
- WebSocket/API 状态放 hook，例如 `useGameRoom`、`useSocialRoom`，避免组件里混合网络协议和渲染细节。
- 游戏视觉组件、规则提示组件和通用玩家卡片分离，避免一个页面文件同时承担布局、状态机、翻译文本和交互。
- i18n key、按钮 variant、玩家状态点、发言气泡等共用能力抽到共享模块，避免各游戏重复实现。

文件行数不是机械指标，但应作为重构信号：当单个 Go/TSX 文件明显难以扫读，或同时包含三种以上职责时，应在本阶段拆分。优先拆出稳定边界，避免为了追求小文件制造无意义转发层。

## 设计原则

### 服务端权威

AI 不能直接调用 `applyMove`、`recordSpeech`、`resolveVote` 等修改函数。AI 只能提交：

```go
type PlayerIntent struct {
    RoomID    string
    PlayerID  string
    RequestID string
    Kind      IntentKind
    ActionID  string
    Speech    string
    Reason    string
    Notes     map[string]string
    CreatedAt time.Time
}
```

RoomActor 收到后：

1. 找到当前权威房间状态。
2. 重新生成该玩家当前合法动作。
3. 验证 `ActionID` 仍然合法。
4. 校验阶段、身份、轮次、目标、隐藏信息边界。
5. 应用状态变更。
6. 记录日志并广播每个订阅者自己的视图。

### 每个 AI 一个长期 Agent

每个 AI 玩家一个 goroutine：

```go
type AIPlayerAgent struct {
    roomID   string
    playerID string
    profile  AIProfile
    memory   AgentMemory
    inbox    chan AgentEvent
    outbox   chan PlayerIntent
    llm      LLMProvider
}
```

Agent 负责：

- 记住自己的 persona、口吻、长期怀疑对象、私人备注。
- 观察桌面事件，例如发言、投票、出牌、阶段变更。
- 在必选动作时做正式决策。
- 在可选时机选择是否发言。
- 避免重复、机械、无意义发言。

### RoomActor 串行处理事件

房间状态由一个 goroutine 串行处理：

```go
type RoomActor struct {
    roomID      string
    game        GameAdapter
    inbox       chan RoomEvent
    agents      map[string]*AIPlayerAgentHandle
    subscribers map[string]Subscriber
    version     int64
}
```

RoomActor 负责：

- HTTP / WebSocket 玩家输入。
- AI intent。
- 房间定时器，例如 Uno 回合超时。
- 玩家在线状态。
- AI required action 调度。
- AI optional speech 调度。
- 广播和房间销毁。

这样可以避免“多个 goroutine 同时写房间”的问题。

## 事件模型

### RoomEvent

所有外部输入进入 RoomActor 都是 RoomEvent：

```go
type RoomEvent struct {
    ID        string
    RoomID    string
    PlayerID  string
    Type      RoomEventType
    Payload   any
    CreatedAt time.Time
}
```

典型类型：

- `HumanIntentSubmitted`
- `AIIntentSubmitted`
- `PlayerSpeechSubmitted`
- `PlayerConnected`
- `PlayerDisconnected`
- `TurnDeadlineReached`
- `RoomIdleTimeoutReached`
- `AgentRequestTimedOut`
- `RoomClosed`

### AgentEvent

RoomActor 发送给 AI Agent 的事件：

```go
type AgentEvent struct {
    ID           string
    RoomID       string
    PlayerID     string
    Type         AgentEventType
    RoomVersion  int64
    PublicState  any
    PrivateState any
    RecentEvents []RoomEvent
    LegalActions []LegalAction
    Deadline     time.Time
}
```

典型类型：

- `Observe`
- `RequiredAction`
- `OptionalSpeech`
- `PhaseChanged`
- `PlayerSpeechObserved`
- `PrivateInfoChanged`
- `Shutdown`

### RequestID 和 RoomVersion

每次 AI 请求都带 `RequestID` 和 `RoomVersion`。

Agent 返回 intent 时，RoomActor 需要检查：

- request 是否仍然有效。
- room version 是否匹配。
- 当前阶段是否仍接受该玩家动作。
- action 是否仍在合法动作列表中。

过期 intent 不应用到房间，但要打结构化日志：

```text
ai intent discarded
room=...
player=...
requestID=...
expectedVersion=...
currentVersion=...
expectedPhase=...
currentPhase=...
reason=stale_version|phase_changed|illegal_action|request_cancelled
```

## GameAdapter 抽象

每个游戏需要提供适配器，让 RoomActor 不知道具体规则细节。

```go
type GameAdapter interface {
    Game() string

    PublicState(viewerID string) any
    PrivateState(playerID string) any

    CurrentRequiredActor() (playerID string, ok bool)
    RequiredActionDeadline() (time.Time, bool)
    LegalActions(playerID string) []LegalAction

    ApplyIntent(intent PlayerIntent) (ApplyResult, error)
    ShouldNotifyAgent(playerID string, event RoomEvent) bool
    BuildAgentContext(playerID string, event AgentEvent) AgentDecisionInput
}
```

`ApplyIntent` 必须是唯一规则入口：

```go
type ApplyResult struct {
    Changed        bool
    Broadcast      bool
    LogEntries     []LogEntry
    RecentActions  []PublicAction
    NextAgentHints []AgentHint
}
```

## AI Agent 生命周期

### 创建

房主添加 AI 时：

1. RoomActor 创建 AI player。
2. 创建 `AIPlayerAgent`。
3. 发送初始 `PrivateInfoChanged` 事件。
4. Agent 初始化 persona 和 memory。

### 运行

Agent 主循环：

```go
func (a *AIPlayerAgent) Run(ctx context.Context) {
    for {
        select {
        case event := <-a.inbox:
            a.handleEvent(ctx, event)
        case <-a.idleTicker.C:
            a.maybeThinkOrSpeak(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

### 关闭

以下场景必须关闭 Agent：

- AI 被房主移除。
- 房间结束并销毁。
- 所有真人离线超时并销毁房间。
- 服务端 shutdown。

关闭时要：

- cancel context。
- drain channel 或让发送侧检查 room lifecycle。
- 从 RoomActor agent registry 移除。

## Agent 决策模式

### 必选动作

RoomActor 发现当前 required actor 是 AI：

```text
RoomActor -> AIPlayerAgent: RequiredAction
AIPlayerAgent -> LLM/tool strategy
AIPlayerAgent -> RoomActor: PlayerIntent
RoomActor validates and applies
```

规则：

- 必选动作有 deadline，默认使用全局 LLM timeout 30 秒。
- deadline 到达仍未返回时，RoomActor 使用 fallback。
- fallback 必须写入日志，例如“北风超时，系统自动行动”。
- Agent 正在处理必选动作时，不处理可选发言。

### 注意力和意图

仅有独立 goroutine 不等于像真人。真人会忽略大部分无关事件，只对与自己目标、身份、风险、当前承诺相关的事件反应。Agent 需要先做轻量注意力判断，再决定是否调用 LLM。

建议每个 Agent 维护：

```go
type AgentAttention struct {
    FocusPlayerIDs []string
    FocusTopics    []string
    SilenceUntil   time.Time
    LastReactedSeq int64
    Interest       map[string]int
}

type AgentIntentState struct {
    CurrentPlan   string
    CommittedTo   map[string]string
    Suspicion     map[string]int
    Trust         map[string]int
    Mood          string
    RiskTolerance int
}
```

事件进入 Agent 后先判断：

- 是否提到了自己。
- 是否影响自己的胜负目标。
- 是否是当前阶段关键事件。
- 是否来自自己重点关注的玩家。
- 是否距离上次发言太近。

通过过滤后才调用完整 LLM。没通过时只更新记忆，不发言。这样能避免所有 AI 对每句话都抢着回应。

### 可选发言

有玩家发言或关键动作后，RoomActor 可通知部分 AI：

```text
RoomActor -> AIPlayerAgent: OptionalSpeech
AIPlayerAgent -> speak | skip
RoomActor validates speech and broadcasts
```

规则：

- 可选发言不能阻塞游戏主流程。
- 每个 AI 有冷却时间，例如 8 到 15 秒。
- 每个桌面事件最多触发 1 到 2 个 AI 发言。
- 如果存在任何 AI required action pending，跳过 optional speech。
- 发言只修改 speech/log，不允许产生游戏动作。
- 发言需要通过“真人感”过滤：过于泛泛、重复、解释规则、复读局面、暴露隐藏信息都应被拒绝或降级为 skip。

### 私有频道和团队频道

真实玩家不只有公共发言。部分游戏需要私有频道或团队频道，否则 AI 无法模拟队友讨论。

```go
type SpeechChannel string

const (
    ChannelPublic SpeechChannel = "public"
    ChannelPrivate SpeechChannel = "private"
    ChannelTeam SpeechChannel = "team"
    ChannelSystem SpeechChannel = "system"
)
```

适用：

- 狼人杀夜晚：狼人频道，只有狼人玩家和狼人 Agent 可见。
- 阿瓦隆邪恶阵营：邪恶阵营初始认人信息进入私有记忆。
- 谁是卧底：通常只有公共频道。
- Uno / 棋类 / 麻将：通常只有公共桌面发言。

RoomActor 广播时必须按 viewer 过滤频道。Agent context 也只能包含它可见的频道内容。

### 反应延迟和真人节奏

Agent 不应总是在 LLM 返回后立即行动。需要在 deadline 内模拟自然思考时间。

```go
type AgentTimingProfile struct {
    MinThink       time.Duration
    MaxThink       time.Duration
    SpeechCooldown time.Duration
    ConfirmDelay   time.Duration
}
```

建议：

- Uno：动作较快，1 到 3 秒。
- 五子棋/象棋：关键局面 2 到 6 秒。
- 麻将：摸牌/打牌较快，吃碰胡稍停顿。
- 社交推理：发言和投票确认更慢，允许观察别人发言。

RoomActor 仍然保留硬 deadline，Agent 只是在 deadline 内模拟自然节奏。

### 社交推理游戏

狼人杀、阿瓦隆、谁是卧底是 Agent 架构收益最高的游戏。

Agent 应维护：

```go
type AgentMemory struct {
    Persona       string
    SpeechStyle   string
    PrivateFacts  map[string]string
    PlayerNotes   map[string]string
    RecentEvents  []MemoryEvent
    RecentSpeech  []SpeechMemory
    Commitments   []string
    Suspicions    map[string]int
}
```

典型效果：

- 狼人能记住队友和夜晚目标。
- 预言家能记住验人结果。
- 谁是卧底能记住自己上一轮描述，避免前后矛盾。
- 阿瓦隆邪恶阵营能记住谁像梅林。
- AI 能在投票确认前根据新发言改变倾向。

社交推理额外要求：

- 支持伪装、误导和试探，但不能泄露系统提示或隐藏信息。
- 发言应是玩家视角，少说规则，少总结全局事实。
- 识别“别人攻击自己”和“别人要求自己表态”。
- 记录自己公开说过的内容，避免自相矛盾。
- 投票确认前可以改变选择，但应记录改变原因。

### 回合制、棋类和麻将

Uno、象棋、五子棋、麻将不应完全照搬社交推理 Agent。这些游戏需要更强的动作策略和更克制的语言表现。

- Uno：重点是手牌管理、颜色控制、阻止快 Uno 的玩家、特殊牌时机。发言短促，像桌面吐槽。
- 象棋：重点是局面评估、威胁处理、将军/被将军、吃子价值。发言少而像棋友短评。
- 五子棋：重点是连子威胁、堵四、造活三。发言围绕落点压力。
- 麻将：重点是牌型进张、危险牌、是否吃碰、听牌倾向。发言不能泄露隐藏手牌。

GameAdapter 应能声明 Agent context 类型：

```go
type AgentContextKind string

const (
    ContextTactical AgentContextKind = "tactical"
    ContextSocial   AgentContextKind = "social"
    ContextHybrid   AgentContextKind = "hybrid"
)
```

不同游戏使用不同 prompt profile，不能所有游戏共用同一个系统提示。

## 投票类机制

讨论型投票建议统一为“选择 + 确认”模型。

```go
type VoteIntent struct {
    TargetID  string
    Confirmed bool
    UpdatedAt time.Time
}
```

RoomActor 规则：

- 选择目标不结算。
- 改目标会取消确认。
- 所有 eligible voters confirmed 后才结算。
- AI Agent 可以先 select，再等待其他人发言，再 confirm。
- 当前非 Agent 架构阶段，AI 可暂时 select + confirm 一步完成。

适用：

- 狼人杀白天放逐。
- 谁是卧底投票。
- 未来其他社交推理投票。

不适用或需隐藏：

- 阿瓦隆队伍投票，提交前应隐藏 approve/reject 内容，只公开已提交人数。

## LLM 调用策略

### Prompt 分层

建议拆成四层：

1. 平台安全层：只能选合法动作、不能泄露私密信息。
2. 游戏规则层：当前游戏的目标、阶段、隐藏信息约束。
3. Persona 层：玩家性格、口吻、长期习惯。
4. 临场上下文层：当前局面、最近发言、合法动作。

### LLM Schema 设计

LLM 输入和 function calling 输出都应优先扁平化。复杂嵌套对象、深层 map、数组套对象再套对象，都会显著增加模型返回错误 JSON、漏字段、把对象 stringify 的概率。

原则：

- function calling 参数保持浅层结构。
- 复杂状态在服务端压缩成 summary 字符串、扁平数组或标量字段。
- legal actions 使用扁平列表，每个 action 只包含 `id`、`label`、`description`。
- player、card、tile、piece 等对象传给 LLM 前转换为扁平 DTO。
- 避免让 LLM 返回复杂嵌套结构。尤其不要要求它返回数组套对象、map 套对象。
- 如果必须返回 map，例如 `notes`，只允许一层 `map[string]string`，并在服务端兼容 stringified JSON。

推荐输入形态：

```json
{
  "game": "werewolf",
  "phase": "vote",
  "playerId": "seat_1",
  "playerName": "白川",
  "role": "villager",
  "alignment": "good",
  "round": 1,
  "publicSummary": "昨夜 snowykami 出局，他白天声称小满可疑。",
  "privateSummary": "你是平民，没有夜晚信息。",
  "recentSpeech": [
    "北风: 我先挂白川。",
    "snowykami: 我验了小满偏坏。"
  ],
  "players": [
    "seat_1|白川|alive|self",
    "seat_2|北风|alive|unknown",
    "seat_3|小满|alive|unknown"
  ],
  "legalActions": [
    {"id":"vote:seat_2","label":"投票给座位2","description":"座位2，存活玩家"},
    {"id":"vote:seat_3","label":"投票给座位3","description":"座位3，存活玩家"}
  ]
}
```

避免输入形态：

```json
{
  "room": {
    "players": [
      {
        "id": "p1",
        "private": {
          "role": {"kind":"villager","alignment":"good"}
        },
        "history": {
          "votes": [{"round":1,"target":{"id":"p2"}}]
        }
      }
    ]
  }
}
```

推荐输出仍保持：

```json
{
  "actionId": "vote:seat_3",
  "reason": "snowykami 的遗言更值得先验证。",
  "speech": "我先跟小满这条线。",
  "notes": {
    "seat_3": "被遗言指认，优先观察"
  }
}
```

Schema 变复杂时，优先在服务端拆成多个简单决策，而不是让一次 LLM 返回大而复杂的 JSON。

### AI 可见信息最小化

AI Agent 必须像真实玩家一样只看到当前玩家能看到的信息。尤其是社交推理游戏，不能因为 AI 在服务端运行就把完整房间状态、完整玩家对象或对手身份直接塞进 LLM 上下文。

强制规则：

- LLM context 不得包含 `kind`、`isAI`、`userId`、`aiProfile`、`connected`、`session` 等会暴露真人/AI 或账号身份的字段。
- 其他玩家统一使用座位别名，例如 `seat_1`、`seat_2`。不要把内部 `player.ID`、用户 ID、AI 用户 ID 暴露给 LLM。
- 玩家列表只给必要字段：座位别名、显示名、座位号、存活状态、公开状态。角色和阵营只在规则允许该 viewer 可见时出现。
- legal action 的 `id` 也应使用同一套座位别名，RoomActor 或 GameAdapter 在应用 intent 前再映射回内部 player ID。
- 发言、投票、队伍、夜晚结果、私人备注、记忆都要经过同一套别名化和可见性过滤。
- 不向 AI 明示某个玩家是真人还是 AI。展示名可以保留，因为真实桌面也能看到名字；但不要提供额外的 AI badge、模型名、persona 或人机类型。
- AI 自己的 persona、私有角色、私有词、私有记忆只进入该 AI 自己的 `PrivateState(playerID)`，不能广播给其他 Agent。
- 游戏结束后是否公开全部身份由游戏规则决定；即使结束公开，也应保持不泄露账号身份和人机类型。

社交推理游戏的 `GameAdapter.BuildAgentContext` 应以 `roleVisible(room, viewer, target)` 或等价规则作为唯一身份可见性来源。狼人杀、谁是卧底、阿瓦隆等游戏需要单独测试：

- 狼人杀平民看不到其他玩家角色；狼人只看到狼人队友；预言家只看到自己的验人结果；女巫只看到规则允许的夜晚信息。
- 谁是卧底只看到自己的词和身份，不看到其他人的词或真实身份。
- 阿瓦隆邪恶阵营看到邪恶阵营；梅林看到邪恶阵营；普通好人看不到隐藏身份。
- 所有 AI context 序列化后都不包含 `isAI`、`kind`、`userId`、内部玩家 ID 或不可见角色。

### 多级思考策略

并不是每次都应调用最贵的完整 LLM 决策。建议分层：

- 规则 fallback：超时或 LLM 不可用时使用。
- 轻量反应：判断是否需要说话，可用较短 prompt。
- 完整决策：必选动作、关键投票、关键社交发言。
- 记忆总结：阶段结束后压缩历史，避免上下文无限增长。

### 记忆压缩

Agent 长期运行会积累大量事件，必须压缩：

```go
type AgentMemoryStore struct {
    ShortTerm    []MemoryEvent
    Summary      string
    PlayerNotes  map[string]string
    PublicClaims map[string]string
}
```

策略：

- 最近 10 到 30 条事件保留原文。
- 阶段结束后生成 summary。
- 玩家备注保留结构化信息。
- 废弃过期局面，但保留关键身份发言、承诺和投票变化。

### 成本和限流

每个 AI 一个 Agent 会显著增加 LLM 并发，需要全局限流：

```go
type LLMGovernor struct {
    MaxConcurrent    int
    PerRoomLimit     int
    PerAgentCooldown time.Duration
}
```

建议：

- 每个房间同时最多 1 到 2 个 LLM required action。
- optional speech 使用更严格限流。
- 同一 Agent 同时只能有一个 LLM 请求。
- 超时和失败要快速 fallback，不阻塞房间。
- 日志中记录是 required action 还是 optional speech，方便排查成本。

### Function Calling

Agent 仍然只通过 function calling 返回：

```json
{
  "actionId": "vote:player_id",
  "reason": "简短理由",
  "speech": "可选桌面发言",
  "notes": {
    "player_id": "私人备注"
  }
}
```

约束：

- `notes` 必须接受 object，也兼容 stringified object，但坏 notes 不应导致 action 失败。
- `speech` 必须经过泄密和模板话过滤。
- `actionId` 必须重新校验。
- LLM 响应失败、解析失败、非法 action 都必须结构化记录。

## 并发模型

### Channel 建议

```go
const roomEventBuffer = 128
const agentEventBuffer = 32
const agentIntentBuffer = 32
```

RoomActor inbox 可以适当大；Agent inbox 不宜太大，避免 AI 落后于房间太多。

### 背压策略

- RoomActor inbox 满：HTTP 返回 503 或 WebSocket 返回错误。
- Agent inbox 满：丢弃 optional speech 事件，但不能丢 required action。
- Agent outbox 满：记录错误并取消该 Agent 当前请求。

### 取消策略

RoomActor 每次发 required action 时保存 active request：

```go
type ActiveAgentRequest struct {
    RequestID string
    PlayerID  string
    Kind      AgentEventType
    Version   int64
    Deadline  time.Time
    Cancel    context.CancelFunc
}
```

当阶段变化、玩家死亡、房间关闭时取消旧 request。

## 日志和可观测性

必须记录：

- Agent 创建/关闭。
- Required action 开始/成功/失败/超时/fallback。
- Optional speech speak/skip/被限流。
- Intent 被丢弃的具体原因。
- LLM provider 状态码、响应体片段、decode error、tool arguments。

推荐字段：

```text
room
game
player
playerName
agentID
requestID
eventType
roomVersion
phase
actionID
speechLength
reasonLength
duration
error
```

## 重构实施计划

### 阶段 1：抽公共包

新增：

```text
server/internal/aiagent/
  types.go
  agent.go
  memory.go
  prompts.go
  scheduler.go
```

定义事件、intent、memory、Agent runner、RoomActor 基础类型和通用日志字段。这个阶段可以不接管具体游戏，但要明确新模型的最终入口，不再为旧 `ScheduleAI` 扩展能力。

同时建立前后端拆分基线：

- 记录当前超大文件和职责混杂文件，作为迁移时优先拆分对象。
- 先抽出共享 Actor / Agent 类型，再迁移具体游戏，避免各游戏各写一套半成品。
- 先抽出前端共享房间 hook、玩家卡片、发言气泡层、规则弹窗等通用组件，再让各游戏页面变薄。
- 每迁移一个游戏，都应减少对应旧 manager/page 文件职责，而不是只新增文件。

### 阶段 2：谁是卧底试点

原因：

- 最依赖真实发言和长期记忆。
- 规则相对集中。
- 最容易验证“像真人独立玩家”的效果。

目标：

- 每个 AI 有独立 session。
- 描述阶段由 required action 触发。
- 投票阶段支持 select + confirm。
- optional speech 不影响 required action。
- 删除谁是卧底旧 `RunNextAI` / `RunAISpeech` 路径，避免同一游戏双轨调度。
- 拆出谁是卧底规则、视图投影、AI context 和前端房间组件，避免社交推理大文件继续膨胀。

### 阶段 3：狼人杀迁移

重点：

- 夜晚身份动作。
- 白天讨论。
- 投票预选和确认。
- 狼队夜晚出刀使用私有选择 + 确认机制，支持多个 AI 狼人先选、改选、确认。
- 猎人临终动作。
- 预言家、女巫等私有信息隔离。
- 删除狼人杀旧调度路径。
- 拆出狼人杀夜晚规则、白天投票、身份视图、AI prompt/action alias 和前端投票面板。

### 阶段 4：Uno 迁移

重点：

- 回合 deadline。
- 自动行动 fallback。
- AI 发言和出牌节奏。
- 手牌私有信息。
- 删除 Uno 旧 AI 调度路径。
- 拆出 Uno 规则、调度器、AI 出牌策略、牌面组件、手牌区和移动端压缩视图。

### 阶段 5：象棋、五子棋、麻将迁移

重点：

- 保持现有规则 AI fallback。
- LLM AI 作为 Agent 策略之一。
- 统一 player status、speech、logging。
- 每迁移一个游戏，就删除该游戏旧 AI 调度路径。
- 逐步拆分各游戏前端页面和后端 manager，保留规则、视图、AI、HTTP/Hub 的清晰边界。

### 阶段 6：删除全局旧调度

所有游戏迁移完成后删除共享层面的旧调度概念：

- `RunNextAI`
- `RunAISpeech`
- Hub 内的重复 `ScheduleAI` / `ScheduleAISpeech`

改为 RoomActor 统一调度。

## 非兼容策略

项目未上线，默认不做长期兼容。迁移期间只允许短时间分游戏过渡，不允许同一个游戏长期保留两套 AI 调度。

允许：

- 内部 Go API 破坏性调整。
- 房间状态 JSON 为了表达新机制而调整，例如投票从字符串变成 `{targetId, confirmed}`。
- 前端同步修改调用方式。
- 测试重写为新行为。

不建议：

- 为旧调度写适配层。
- 用 feature flag 长期维持旧/新两套 AI 行为。
- 为了少改前端而保留不合理的后端状态结构。
- 保留“旧调度可用但没人维护”的死路径。

如果需要降低风险，可以按游戏分支迁移，但每个游戏完成迁移后必须删除该游戏旧路径。

## 风险和应对

### goroutine 泄漏

风险：房间销毁后 Agent 未退出。

应对：

- 每个房间一个 root context。
- RoomActor 关闭时 cancel 所有 Agent。
- 测试房间创建/销毁后 goroutine 数量趋势。

### 事件积压

风险：LLM 慢，Agent inbox 堆积旧事件。

应对：

- optional speech 可丢弃。
- Agent 只保留最近 N 条事件。
- required action 使用 requestID 和 deadline。

### 隐藏信息泄漏

风险：Agent context 拼装错误，把其他玩家私密信息传给 LLM。

应对：

- GameAdapter 提供 `PrivateState(playerID)`。
- 为每个社交游戏写不同 viewer 的 AI context 测试。
- 禁止把完整 Room 直接传给 AI。

### AI 过度发言

风险：每个 Agent 都想回应，桌面很吵。

应对：

- 每事件最多触发 1 到 2 个 AI。
- 每 AI 发言冷却。
- 低相关事件直接跳过。
- 允许 RoomActor 根据桌面密度抑制 optional speech。

### 结算不一致

风险：AI 返回时状态已变。

应对：

- RoomActor 重新校验所有 intent。
- stale intent 只记录，不应用。
- required action 超时走 fallback。

## 验证清单

- 多 AI 连续 required action 不会互相阻塞房间。
- optional speech 不会打断 required action。
- stale intent 有明确日志。
- 房间销毁后 Agent 全部退出。
- AI 被房主移除后 Agent 退出。
- 玩家断线/重连不影响 Agent 状态。
- 每个社交 AI 只看到自己该知道的身份、词、夜晚结果和私人备注。
- LLM 超时后 fallback 生效。
- 前端仍通过同一 WebSocket 收到广播。

## 预期体验

完成后，AI 玩家会更像真人：

- 有自己的记忆和前后连贯发言。
- 会根据其他人发言改变怀疑和投票。
- 不会每次都说模板话。
- 不会在不该说话时插队影响游戏推进。
- 在回合制游戏里有思考间隔、发言节奏和行动 fallback。
