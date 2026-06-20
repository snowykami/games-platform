# 游戏平台工程审查 Skill

本文件用于后续新增或重构游戏时参考。目标是避免联机规则、隐藏情报、安全边界和并发模型在不同游戏里重复踩坑。

## 核心原则

- 服务端永远是权威状态。客户端只能提交意图，不能提交可直接信任的游戏结果。
- 公共视图和私有视图必须分离。任何 `PublicRoom`、WebSocket 广播和 AI 输入都要显式说明 viewer。
- 隐藏信息游戏默认最小披露。角色、阵营、手牌、任务牌、投票内容、私人备注、AI 记忆都不能因为“前端不用”而出现在公共 JSON 中。
- 所有真人动作和 AI 动作必须走同一套合法动作校验，不允许 AI 直接改权威状态。
- LLM 是慢且不可信的外部依赖。调用必须有超时，返回 action 必须验证，网络调用不能长期持有房间锁。
- 联机广播不能被慢连接拖死。WebSocket 写入需要超时、错误清理和尽量隔离单个订阅者失败。
- 新增或重构游戏时必须同步治理文件结构和文件行数。不要把规则、视图、AI、HTTP、Hub、前端页面交互继续堆进单个大文件；前后端都要按职责拆分。

## 新游戏上线检查表

### 规则真实性

- 先写清楚人数、阶段、动作、结算、胜负条件和异常情况。
- 与真实桌游/棋牌规则对照，列出有意简化的部分。
- 隐藏动作必须符合真实流程：例如阿瓦隆队伍投票和任务牌应在提交完成后统一揭示；未完成前只能公开“已提交/未提交”。
- 队伍、投票、夜晚行动、听牌/声明等动作要有重复提交、越权提交、阶段错误、目标非法的测试。

### 情报隔离

- `PublicPlayer` 不应默认向其他玩家暴露稳定 `userId`。前端识别自己优先使用 `youPlayerId` 或服务端给 viewer 的标记。
- 每个游戏都要有一组“不同 viewer 看到不同 JSON”的测试：
  - 自己能看到自己的私有信息。
  - 队友只能看到规则允许的信息。
  - 旁观/其他玩家看不到隐藏信息。
  - 游戏结束后是否公开，必须按规则明确。
- 社交推理 AI 输入只能包含该 AI 当前应知道的信息。不要把完整 `Room`、完整投票明细或其他玩家私有备注传给 LLM。
- 社交推理 AI 输入不得暴露真人/AI 类型。LLM context 中禁止出现 `kind`、`isAI`、`userId`、`aiProfile`、`connected` 等账号或人机身份字段；其他玩家统一使用 `seat_1`、`seat_2` 这类座位别名。
- AI 输入里的玩家、发言、投票、队伍、夜晚结果、私人备注和 legal action 都必须使用同一套座位别名，并按 viewer 可见性过滤。服务端在应用 intent 前再把别名映射回内部 player ID。
- 社交推理游戏必须为 AI context 写最小披露测试：狼人杀不能让平民看到他人角色；谁是卧底不能看到他人词或身份；阿瓦隆只能按邪恶阵营和梅林规则披露隐藏身份。
- 私人备注按 `viewerPlayerId -> targetPlayerId` 保存和投影，广播时按订阅者重新生成视图。

### 认证与安全

- 新接口默认挂在 `auth.RequireUser` 之后；管理接口额外使用 `auth.RequireAdmin`。
- 首个管理员引导不能让公开游客登录直接拿 admin。生产环境应使用显式 bootstrap secret、OIDC 白名单或部署期初始化。
- Session cookie 生产环境必须设置 `Secure`，保持 `HttpOnly` 和 `SameSite=Lax/Strict`。
- 登录、发言、添加 AI、LLM 决策、创建/加入房间需要速率限制。
- OIDC state 要有过期时间和清理机制。
- 静态响应建议加基础安全头：`X-Content-Type-Options`、`Referrer-Policy`、`X-Frame-Options` 或 CSP `frame-ancestors`，生产环境加 HSTS。
- 房间号不是权限边界。若房间链接被视为私密入口，房间 id 应使用 crypto random，并考虑加入短期邀请 token。

### 并发与可用性

- 优先使用房间级锁，避免整个游戏 Manager 一个慢房间阻塞所有房间。
- 不要在锁内执行 LLM、HTTP、数据库、Redis 或 WebSocket 写入等慢 I/O。建议流程：
  - 锁内复制决策所需快照和合法动作。
  - 解锁后调用外部服务。
  - 重新加锁后验证阶段、当前玩家和 action 仍然合法，再应用。
- 所有外部调用使用 context timeout。LLM 使用平台共享常量，默认 30 秒；WebSocket 单连接写入建议 1-2 秒。
- AI 调度要防止重复 goroutine、死循环和跨房间阻塞。`ScheduleAI` 应按 room 去重。
- 长连接断开后要清理订阅者；移除玩家时要关闭对应 WebSocket。

### 前端与交互

- 准备页允许页面自然滚动，不能为了单屏把开始按钮或关键操作困在内部滚动区。
- 对局页可尽量单屏，但移动端允许局部滚动。
- 气泡、弹窗、输入面板优先用 portal，避免被游戏舞台 `overflow-hidden` 裁剪。
- 发言气泡是临时反馈，默认 3-5 秒淡出；社交推理可保留日志作为持久记录。
- 玩家操作按钮必须根据阶段、身份和是否轮到自己禁用，但服务端仍需完整校验。

## 推荐抽象

- 抽出共享房间 HTTP/Hub 模板：创建、加入、添加 AI、移除玩家、发言、改名、广播、调度 AI、关闭订阅者。
- 抽出通用公开视图模式：`Public(roomID, viewerUserID)` 或 `Public(roomID, viewerPlayerID)`，禁止无 viewer 的隐藏信息游戏广播。
- 抽出 AI 决策管线：构建快照、列出合法动作、调用 provider、验证 action、应用动作、记录 speech/notes。
- 将后端大文件按职责拆分：
  - `manager.go`：房间生命周期和协调逻辑。
  - `rules_*.go`：规则和结算。
  - `public_view.go`：公开/私有视图。
  - `ai_*.go`：AI 状态、action alias、prompt/context 和 fallback。
  - `room_actor.go` / `agent.go`：Actor 主循环、AI Agent、记忆和调度。
  - `http.go`：只保留路由和请求解码。
  - `hub.go`：只保留订阅、广播和连接生命周期。
- 将前端页面按职责拆分：
  - slug 页面只做路由、鉴权和数据装配。
  - 按 `Lobby`、`RoomHeader`、`PlayerList`、`GameTable`、`ActionPanel`、`SpeechBubbleLayer`、`RulesModal` 等组件拆分。
  - WebSocket/API 状态进入 hook，例如 `useGameRoom` / `useSocialRoom`，组件只消费状态和事件函数。
  - i18n、按钮 variant、玩家状态点、发言气泡、规则弹窗等可复用能力进入共享模块。
- 文件行数不是机械指标，但如果单个文件难以扫读，或混合三种以上职责，应作为重构阶段的拆分对象。优先拆稳定边界，避免为了小文件制造空壳转发层。

## 必备验证

- 后端：`go test ./server/...`，涉及锁/AI/Hub 时跑 `go test -race ./server/...`。
- 前端：`pnpm lint:web`、`pnpm build:web`。
- 依赖：`pnpm --dir web audit --audit-level moderate`。
- 关键 UI：桌面横屏和手机竖屏截图，确认按钮可达、文本不溢出、气泡不被裁剪。
- 隐藏信息：为每个社交推理游戏写 public view 测试，明确不同 viewer 的 JSON 差异。
