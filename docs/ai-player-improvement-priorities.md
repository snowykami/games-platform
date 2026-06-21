# AI 玩家后端改进优先级

## 背景

当前 AI 玩家后端已经具备较清晰的 Agent 化基础：

- `gameactor.RoomCommandRegistry` 提供按房间串行的命令入口，降低多人和 AI 同时写房间状态的风险。
- `gameactor.RoomAIScheduler` 负责同房间 AI 正式动作和可选发言的互斥调度。
- `aiagent.Controller`、`aiagent.Agent` 和 `aiagent.Registry` 已经让每个 AI 玩家拥有独立 goroutine、session、persona 和短期 memory。
- 社交推理游戏已经开始做座位别名、身份可见性过滤、规则版本和发言版本分离。

整体方向正确，但当前实现仍处在“半 Actor 化”阶段：房间状态权威性主要依赖各游戏 manager、room lock、stale 校验和调度约定共同保证，还没有完全收敛到文档中设想的 `GameAdapter -> RoomActor -> PlayerIntent` 模型。

## 优先级排序

### P0：收敛成真正的 GameAdapter -> RoomActor 模型

当前问题：

- `RoomCommandRegistry` 已经保证同房间命令串行，但规则入口仍分散在各游戏 manager。
- 各游戏仍保留 `RunAIAction`、`RunAIOptionalSpeech` 这类调度语义。
- AI intent 的应用、合法动作生成、公开/私有视图投影和广播策略还没有统一抽象。

建议：

- 落地统一 `GameAdapter`：

```go
type GameAdapter interface {
    PublicState(viewerID string) any
    PrivateState(playerID string) any
    CurrentRequiredActor() (playerID string, ok bool)
    LegalActions(playerID string) []aiplayer.LegalAction
    ApplyIntent(intent gameactor.PlayerIntent) (ApplyResult, error)
    BuildAgentContext(playerID string, event gameactor.AgentEvent) map[string]any
}
```

- 让 RoomActor 成为唯一状态变更入口。
- 各游戏 manager 逐步只负责房间生命周期和适配器装配。
- 新能力不再进入旧式 scheduler，而是进入 adapter/actor 管线。

预期收益：

- 状态写入边界更清晰。
- 新游戏接入成本更低。
- AI、人类、定时器和系统 fallback 走同一套规则入口。

### P0：统一 stale 判定机制

当前问题：

- Uno 使用 `Phase + ActionSeq + UpdatedAt`。
- 社交推理使用 rule/speech 版本。
- 棋类和麻将各有自己的状态重检方式。
- 公共包里已有 `RoomVersion`、`ActiveAgentRequest`、`LegalActionHash`，但尚未成为所有游戏的统一机制。

建议：

- 所有 AI 请求统一保存：

```go
type ActiveAgentRequest struct {
    RequestID       string
    PlayerID        string
    Kind            gameactor.AgentEventType
    Phase           string
    RuleVersion     int64
    SpeechVersion   int64
    LegalActionHash string
    Deadline        time.Time
}
```

- required action 只因规则版本、阶段、行动者、合法动作集合变化而过期。
- optional speech 只因发言版本、阶段、行动者或冷却策略变化而过期。
- stale intent 统一记录结构化日志，不应用到房间。

预期收益：

- 避免“别人发言导致 AI 正式动作过期”这类误伤。
- 避免不同游戏 stale 语义漂移。
- 更容易定位 AI 决策被丢弃的真实原因。

### P0：增加 LLM 全局限流和 per-room budget

当前问题：

- 每个 AI Agent 可以独立发起 LLM 请求。
- 多 AI 房间、多个房间同时运行时，缺少全局并发和成本保护。
- optional speech 与 required action 的优先级差异还没有进入统一 governor。

建议：

- 新增 `LLMGovernor`：

```go
type LLMGovernor struct {
    MaxConcurrent    int
    PerRoomLimit     int
    PerAgentCooldown time.Duration
}
```

- required action 优先，optional speech 可被限流后直接 skip。
- 每个房间同时最多 1 到 2 个 LLM 请求。
- 每个 Agent 同时只允许一个 active LLM 请求。
- 记录限流原因：`global_limit`、`room_limit`、`agent_cooldown`、`optional_suppressed`。

预期收益：

- 控制成本。
- 避免高并发 LLM 拖慢房间。
- AI 多的桌面不会同时抢话或堆积请求。

### P1：删除或压缩旧式 RunAIAction / RunAIOptionalSpeech 概念

当前问题：

- 多个游戏仍通过 `RunAIAction` 和 `RunAIOptionalSpeech` 表达 AI 调度入口。
- 社交推理中仍标注了 legacy scheduler。
- 命名虽然已经 actor 化，但模型上仍容易让维护者误以为 AI 是“被动函数调用”。

建议：

- 逐游戏迁移到 `SubmitAgentEvent -> PlayerIntent -> ApplyIntent`。
- 保留短期兼容时，明确标注迁移目标和删除条件。
- 完成一个游戏后删除该游戏旧路径，避免双轨长期并存。

预期收益：

- 降低架构心智负担。
- 减少调度分支。
- AI 玩家更符合“持续存在的玩家”模型。

### P1：加强 AI context 隐私测试

当前问题：

- 社交推理已经做了座位别名和角色可见性过滤。
- 但隐藏信息泄漏风险很高，需要自动化测试长期守住边界。

建议：

- 为狼人杀、阿瓦隆、谁是卧底分别增加 AI context 序列化测试。
- 测试不同 viewer 下不得出现：
  - `isAI`
  - `kind`
  - `userId`
  - `aiProfile`
  - `connected`
  - 内部 player ID
  - 不可见角色
  - 不可见阵营
  - 不可见词语或夜晚结果
- 将 context 序列化成 JSON 后做 deny-list 和可见性断言。

预期收益：

- 防止服务端 AI 因拿到完整房间状态而作弊。
- 降低后续新增字段时误泄漏的风险。
- 社交推理游戏更可信。

### P1：升级 Agent memory

当前问题：

- 通用 Agent memory 主要是最近事件字符串列表。
- 社交推理另有 session memory 和 private notes，但结构还偏轻量。
- AI 难以稳定维护长期怀疑、承诺、投票理由和公开说过的话。

建议：

```go
type AgentMemoryStore struct {
    ShortTermEvents []MemoryEvent
    Summary         string
    PlayerNotes     map[string]string
    PublicClaims    map[string]string
    Commitments     []string
    CurrentPlan     string
    Suspicion       map[string]int
    Trust           map[string]int
}
```

- 最近 10 到 30 条事件保留原文。
- 阶段结束时压缩 summary。
- 玩家备注和公开承诺保留结构化字段。
- 谁是卧底记录自己上一轮描述，避免前后矛盾。
- 狼人杀和阿瓦隆记录怀疑链、投票理由和阵营推断。

预期收益：

- AI 发言更连贯。
- 社交推理 AI 更像真实玩家。
- 减少重复、模板化和自相矛盾发言。

### P1：给 optional speech 加注意力过滤

当前问题：

- 可选发言主要由最近发言触发。
- 多 AI 房间里容易出现所有 AI 都想回应的情况。
- 当前冷却和密度控制还不够统一。

建议：

```go
type AgentAttention struct {
    FocusPlayerIDs []string
    FocusTopics    []string
    SilenceUntil   time.Time
    LastReactedSeq int64
    Interest       map[string]int
}
```

- 触发 optional speech 前先判断：
  - 是否提到自己。
  - 是否影响自己的胜负目标。
  - 是否来自重点关注玩家。
  - 是否处于关键阶段。
  - 是否距离上次发言太近。
  - 当前桌面发言是否过密。
- 每个桌面事件最多触发 1 到 2 个 AI 发言。
- 低相关事件只更新记忆，不调用完整 LLM。

预期收益：

- 桌面更安静自然。
- 降低 LLM 成本。
- AI 发言更像有注意力和个性的玩家。

### P2：增强 Agent 生命周期与泄漏测试

当前问题：

- Registry 已支持移除单个 Agent、移除房间 Agent 和关闭 Controller。
- 但需要更多并发和销毁场景测试。

建议测试：

- 房间销毁后 Agent registry 清空。
- AI 被房主移除后 Agent 退出。
- 正在等待 LLM 时房间关闭，decision 不回写。
- 正在等待 LLM 时玩家被移除，decision 不回写。
- 高频创建/销毁房间后 goroutine 数量不持续上涨。
- `AgentShutdown` 事件和 `Close` 路径都可安全退出。

预期收益：

- 降低 goroutine 泄漏风险。
- 房间生命周期更可靠。
- 长时间运行服务更稳定。

### P2：增加可观测性指标

当前问题：

- 结构化日志已经覆盖 LLM 成功、失败、stale 等关键路径。
- 但缺少面向线上观测的 metrics。

建议增加指标：

- LLM 请求数、成功数、失败数、超时数。
- fallback 次数。
- stale discard 次数和原因分布。
- optional speech speak/skip 次数。
- Agent 创建/关闭数量。
- 每房间 Agent 数。
- Agent inbox 长度。
- LLM governor 当前并发。
- per-room LLM budget 使用量。

预期收益：

- 更容易调参。
- 更容易发现成本异常。
- 更容易定位 AI 卡住、太吵或不说话的问题。

### P2：统一 fallback 行为和日志

当前问题：

- 各游戏都有自己的规则 fallback。
- LLM 不可用、超时、非法 action、stale、无合法动作等场景的日志和行为还不完全统一。

建议：

```go
type AIFallbackReason string

const (
    FallbackLLMDisabled   AIFallbackReason = "llm_disabled"
    FallbackLLMTimeout    AIFallbackReason = "llm_timeout"
    FallbackIllegalAction AIFallbackReason = "illegal_action"
    FallbackStaleIntent   AIFallbackReason = "stale_intent"
    FallbackNoLegalAction AIFallbackReason = "no_legal_action"
)
```

- fallback 必须产生结构化日志。
- required action fallback 可改变规则状态。
- optional speech fallback 默认 skip。
- 用户可见日志只描述游戏行为，不暴露 LLM 或系统细节。

预期收益：

- 行为一致。
- 排查简单。
- 不同游戏的 AI 体验更统一。

## 建议实施顺序

1. 先做 `GameAdapter -> RoomActor` 的最小闭环，选择一个游戏试点。
2. 同步把该游戏 stale 判定改为 `RoomVersion + ActiveAgentRequest + LegalActionHash`。
3. 加 `LLMGovernor`，先覆盖 required action 和 optional speech 两类请求。
4. 为社交推理 AI context 补隐私测试。
5. 在社交推理游戏中升级 memory 和 attention。
6. 逐游戏删除旧式 AI 调度入口。

前三项是当前收益最高的改动，会直接提升架构稳定性、成本可控性和后续扩展能力。
