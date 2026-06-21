# 游戏平台项目指引

本文件是 Liteyuki Games 平台的项目级开发指引。它合并了早期规格说明、游戏开发 skill 和工程审查清单，作为后续开发者和 AI Agent 的共同参考。

## 产品目标

构建一个 React + Vite 前端、Go 后端的轻量联机小游戏平台。

平台入口是游戏总览页，通过 `/games/:slug` 进入具体游戏。联机游戏进入 slug 页面后先展示创建/加入房间流程，房间链接使用 `?room=xxxxx`，开始后才进入具体游戏桌面。

当前平台重点：

- 游戏总览和游戏 slug 路由。
- 游客登录和多 OIDC provider 登录。
- WebSocket 联机房间。
- 服务端权威游戏状态。
- AI 玩家制度和统一 LLM provider。
- 后续可持续添加棋类、牌类、桌游和社交推理游戏。

平台层只负责大厅、路由、身份、房间、Socket、广播和共享能力。具体游戏规则必须隔离在各自游戏模块或适配器中。

## 当前仓库结构

```text
.
├── docs/                    # 项目指引、UI 规范、AI Actor 重构计划
├── server/                  # Go 后端
│   ├── cmd/api/             # API 服务入口
│   └── internal/
│       ├── auth/            # 用户、会话、角色、OIDC
│       ├── config/          # 环境配置
│       ├── games/           # 游戏注册表
│       ├── httpx/           # HTTP JSON 工具
│       ├── i18n/            # 后端 i18n
│       ├── roommeta/        # 房间元信息
│       ├── runtimecheck/    # 启动依赖检查
│       ├── uno/             # UNO
│       ├── socialdeduction/ # 狼人杀、阿瓦隆、谁是卧底
│       └── web/             # embed 前端静态资源
└── web/                     # React + Vite 前端
```

约定：

- 前端目录固定为 `web/`。
- 后端目录固定为 `server/`。
- 根目录使用 `pnpm` 聚合常用命令。
- 不引入微服务；第一阶段保持模块化单体。

## 本地开发

常用命令：

```bash
pnpm install
pnpm dev:infra
pnpm dev:server
pnpm dev:web
```

重要约定：

- 前端 Vite 支持热更新。
- 后端 `pnpm dev:server` / `go run ./server/cmd/api` **不会热重载**。修改 Go 代码后必须停止并重启后端。
- 大重构或后端行为验证前，必须确认后端已经重启。
- 浏览器验证前，确认当前访问的是重启后的后端进程。

构建：

```bash
pnpm build:web
pnpm build:server
```

验证：

```bash
pnpm test:server
pnpm lint:web
pnpm build:web
pnpm --dir web test
```

涉及锁、Hub、Actor、AI 并发时，建议补充：

```bash
go test -race ./server/...
```

## 配置与部署

环境变量支持直接环境变量和本地 `.env` 文件。真实环境变量优先，`.env` 只用于本地开发补齐未设置值。

常用配置：

```bash
PORT=8901
APP_ENV=development
SESSION_COOKIE_SECURE=false
DB_URL=postgres://games_platform:games_platform@localhost:5436/games_platform?sslmode=disable
REDIS_URL=redis://localhost:6385/0
LLM_API=
LLM_MODEL=
LLM_TOKEN=
OIDC_PROVIDERS_JSON=
```

约定：

- 后端默认端口是 `8901`。
- 生产环境设置 `APP_ENV=production` 或 `SESSION_COOKIE_SECURE=true`，让 session cookie 带 `Secure`。
- PostgreSQL 使用 `DB_URL`，格式为 `postgres://...` 或兼容 URL。
- Redis 使用 `REDIS_URL`，格式为 `redis://...`。
- LLM 使用 `LLM_API`、`LLM_MODEL`、`LLM_TOKEN`。
- OIDC 使用 `OIDC_PROVIDERS_JSON`，必须支持多个 provider。
- AI、OIDC、数据库、Redis 密钥只允许来自环境变量、忽略的本地 `.env` 或部署密钥，不进入前端、不写入日志。
- PostgreSQL 和 Redis 是配置后的必需依赖，启动检查失败时后端应 fail fast。
- Dockerfile 构建后端镜像时必须同时构建前端，并将前端产物 embed 到 Go 二进制中；不要要求提前提交构建产物。

## 用户与身份

用户模式：

| 模式 | 说明 |
| --- | --- |
| 游客 | 浏览器本地保存匿名 UUID，服务端按游客身份保存对局数据。其他玩家只看到游客展示名。 |
| OIDC | 支持多个 OpenID Connect provider。前端从后端获取 provider 列表，不写死单一登录入口。 |

平台角色只有两种：

- `admin`：平台管理员。第一个用户默认成为管理员。可执行封禁/解封等管理操作。
- `player`：普通玩家。

约束：

- 同一个 `(providerKey, subject)` 只能绑定一个平台用户。
- 不同 provider 的 `subject` 不可直接比较，必须带上 `providerKey`。
- 游戏代码只消费标准化用户结构，不关心用户来自游客还是 OIDC。
- 未登录打开分享房间链接时，跳转登录页；用户可选择游客登录或 OIDC 登录后继续打开原链接。

## 路由与房间体验

前端路由：

| 路由 | 用途 |
| --- | --- |
| `/` | 游戏大厅页面。 |
| `/games/:slug` | 指定游戏页面。 |
| `/games/:slug?room=xxxxx` | 加入或打开指定联机房间。 |

行为：

- 如果 `:slug` 不存在，展示未找到状态。
- 如果联机游戏没有 `room`，展示创建房间和输入房间码入口，不直接进入对局。
- 如果 URL 中带有 `room`，在身份就绪后进入房间准备页。
- 准备页展示玩家、AI、复制链接、玩法类型和开始按钮。
- 房主开始游戏后才进入具体游戏桌面。
- 进入具体游戏 slug 页面后，视觉风格服从游戏本身，不沿用大厅卡片风格。

房间生命周期：

1. 客户端按 game slug 请求创建房间。
2. 服务端创建房间并返回 room id。
3. 客户端更新 URL 为 `/games/:slug?room=:roomId`。
4. 客户端打开 WebSocket 并加入房间。
5. 服务端校验游戏是否支持联机、房间容量和用户身份。
6. 服务端向房间内连接广播该 viewer 对应的公开视图。
7. 房主开始后，服务端广播游戏开始。
8. 空房或所有真人离线达到游戏定义的超时后关闭。

## 游戏注册与页面标准

每个游戏通过注册表暴露平台级定义：

```ts
type GameDefinition = {
  slug: string;
  title: string;
  description: string;
  minPlayers: number;
  maxPlayers: number;
  supportsOnline: boolean;
  supportsLocal: boolean;
  status: "available" | "coming-soon";
};
```

新游戏必须交付：

- 游戏注册信息。
- `/games/:slug` 页面。
- 游戏专属创建/加入/准备页。
- 支持联机的游戏需要 HTTP + WebSocket 房间能力。
- 游戏规则、胜负条件、非法动作处理。
- 公开视图和私有视图。
- AI 能力决策：规则 AI、LLM AI 或不支持 AI。
- 游戏卡片图、SVG 图标或 favicon。
- i18n 文案。

游戏页面必须遵循 [前端游戏 UI 开发规范](./frontend-game-ui-guidelines.md)。

## WebSocket 与服务端权威

联机游戏永远以服务端状态为准。

原则：

- 客户端只发送意图，不发送可信游戏结果。
- 服务端校验每个动作是否合法。
- 非法动作返回错误，不修改游戏状态。
- WebSocket 广播不能被慢连接拖死，写入需要 timeout 和错误清理。
- 广播必须按 viewer 重新生成视图，隐藏信息游戏禁止无 viewer 的全量广播。

后续 Actor 重构以 [AI 玩家 Agent Actor 重构计划](./ai-player-agent-actor-architecture.md) 为准。

过渡期约束：

- `RunAIAction` / `RunAIOptionalSpeech` / `ScheduleAIAction` / `ScheduleAIOptionalSpeech` 必须由 `RoomCommandRegistry` 包裹运行，不能绕过 room actor 写状态。
- 新的 AI 玩家能力必须优先接入 `gameactor.RoomActor`、`gameactor.RoomVersion` 和 `aiagent.Controller`；不要在游戏 manager 里直接管理 LLM broker / registry。
- 新的房间写操作必须经由 `RoomCommandRegistry` 或后续完整 `RoomActor` adapter；HTTP、WebSocket、后台 ticker、AI scheduler 不能直接并发写同一个房间。
- 同一房间的正式 AI 动作与可选发言不得并发执行；发言必须让位于 required action。
- 游戏 manager 不应直接调用 `aiProvider.Decide`；LLM 请求必须经由每个 AI 玩家自己的 `aiagent.Agent`，再以 intent 形式交还规则层校验和应用。
- 每迁移一个游戏，必须清理该游戏的旧命名和重复调度入口，避免长期双轨；大型 manager/http 文件要继续拆薄到规则、视图、AI context、传输层组件。

## 后端工程标准

后端使用 Go，核心依赖：

- HTTP：`net/http` + `chi`
- WebSocket：`coder/websocket`
- PostgreSQL：`pgx`
- Redis：`go-redis`
- OIDC：`coreos/go-oidc` + `oauth2`

核心原则：

- 服务端永远是权威状态。
- 公共视图和私有视图必须分离。
- 所有真人动作和 AI 动作必须走同一套合法动作校验。
- 不在锁内执行 LLM、HTTP、数据库、Redis 或 WebSocket 写入等慢 I/O。
- 外部调用必须使用 context timeout。LLM 默认使用平台共享 30 秒超时。
- 新增或重构游戏时必须同步治理文件结构和文件行数。

推荐拆分：

- `manager.go`：房间生命周期和协调逻辑。
- `rules_*.go`：规则、阶段推进、结算、胜负判定。
- `view_*.go` / `public_view.go`：公开/私有视图。
- `ai_*.go`：AI 状态、action alias、prompt/context、fallback。
- `room_actor.go` / `agent.go`：Actor 主循环、AI Agent、记忆和调度。
- `http.go`：路由注册、请求解码、响应编码。
- `hub.go`：订阅、广播、连接生命周期。

文件行数不是机械指标，但如果单个文件难以扫读，或混合三种以上职责，应在重构阶段拆分。优先拆稳定边界，避免为了小文件制造空壳转发层。

## 前端工程标准

前端技术栈：

- React
- Vite
- TypeScript
- React Router
- Tailwind CSS
- shadcn 风格本地组件 / Radix primitives
- lucide-react
- TanStack Query
- Zustand
- Zod
- antfu/eslint-config
- Vitest + Testing Library
- Playwright

固定约定：

- 包管理器使用 `pnpm`。
- 路径别名统一使用 `@/` 指向 `src/`。
- TypeScript 启用严格模式，禁止隐式 `any`。
- REST/WS 边界使用 Zod 或等价方式校验。
- 组件和 hooks 使用 Vitest + Testing Library。
- 明显 UI 变更后使用浏览器截图检查桌面横屏和手机竖屏。

结构建议：

- slug 页面只做路由、鉴权和数据装配。
- 游戏 UI 拆成 `Lobby`、`RoomHeader`、`PlayerList`、`GameTable`、`ActionPanel`、`SpeechBubbleLayer`、`RulesModal` 等聚焦组件。
- WebSocket/API 房间状态进入 hook，例如 `useGameRoom` 或游戏专属 hook。
- i18n、按钮 variant、玩家状态点、发言气泡、规则弹窗等复用能力进入共享模块。

移动端标准：

- 移动端适配优先交互和人体工学。
- 可以牺牲装饰、复杂动画、非关键统计和增强效果。
- 不能丢失必要信息、主要操作入口、确认流程、错误/胜负反馈和倒计时等核心状态。
- 关键按钮要方便拇指操作：点击区域充足、位置稳定、不会被滚动容器或弹窗遮挡。

## 设计系统与 UI

全局平台：

- 游戏大厅是平台入口，清晰展示游戏列表、状态、人数、是否支持联机。
- 大厅可以是工作台式信息架构。
- 全局按钮和颜色组合必须复用既有 variants，不临时拼接一次性按钮配色。
- 使用 lucide 图标表达常见动作和状态。
- 平台与游戏 favicon/logo 使用 SVG。
- 游戏卡片可使用生成或提供的 bitmap artwork。

具体游戏：

- slug 页面视觉服从游戏本身。
- 游戏详情页第一屏是实际可用体验，不做营销 landing page。
- 桌面横屏尽量避免主流程滚动。
- 手机竖屏允许必要滚动，尤其是房间、创建、加入页。
- 卡牌/麻将等隐藏信息在移动端优先压缩显示，例如单张背面牌 + 数量。
- 发言使用浮动气泡，锚定玩家区域，不把发言文本塞进玩家卡导致布局跳动。
- 人类玩家随时可以发言，不受当前回合限制。

## AI 玩家标准

AI 是正式玩家制度，不是演示数据。

通用规则：

- AI 玩家和真人共享座位、回合、胜负、手牌数量、状态点和公开日志。
- AI level 存在每个 AI 玩家上。
- 规则 bot level：`beginner`、`normal`、`master`。
- LLM level：`ai`，仅当 LLM provider 可用时开放。
- 后端暴露共享 AI profiles/personas，至少 10 个可复用 profile。
- 前端 LLM badge 显示 `AI: {LLM_MODEL}`，如果模型名可用。
- LLM 使用 function calling，只能返回合法动作 id。
- LLM 不可信，返回 action 必须重新校验。
- LLM 输入和输出 schema 保持扁平，避免深层嵌套 JSON。
- LLM 失败、stale、非法 action、provider 响应异常必须打结构化日志，但不能记录 token。

社交推理游戏：

- 狼人杀、阿瓦隆、谁是卧底等发言推理游戏只使用 LLM AI。
- 每个 AI 玩家保留独立 LLM session、persona、记忆和私人备注。
- AI context 必须最小披露，不泄露其他玩家隐藏身份、词、阵营、手牌、私人备注或账号信息。
- LLM context 禁止出现 `kind`、`isAI`、`userId`、`aiProfile`、`connected` 等真人/AI 或账号身份字段。
- 其他玩家统一使用 `seat_1`、`seat_2` 等座位别名，legal action id 也使用同一套别名。
- 服务端应用 intent 前再把座位别名映射回内部 player id。
- AI 发言应短、自然、像真实玩家桌面发言，不要使用“生活里常见”“具体场景”“不能说太细”等机器人套话。

主动发言：

- AI 可以在安全时机主动发言。
- 主动发言是 speech-only action，不能修改权威游戏状态。
- required action pending 时不得触发会打断正式动作的 optional speech。
- 后续 Actor 模型中，speech/presence/log 事件不得使 required action stale。

## UNO 标准

- UNO 规则必须 variant-driven，为 DLC 风格扩展预留，例如 +6、+10、叠加惩罚、Flip-like 模式和 house rules。
- 经典规则先可玩，再开放 variants。
- 主题牌面和规则扩展分开建模。
- 房间页和对局页展示当前玩法类型，可用 info 图标打开规则说明。
- 牌面支持主题扩展，但未获得授权的 IP 主题只能作为系统占位，不直接使用侵权资产。
- 回合超时由服务端维护，前端只显示服务端下发的 deadline。
- 当前玩家断线或 30 秒内无动作，服务端自动行动。
- 真人全离线超过 60 秒，清理 lobby 或 playing 房间。
- 超时和自动行动必须写入 recent actions/log。

## 安全与校验

- 永远不要信任客户端游戏状态。
- 新接口默认挂在 `auth.RequireUser` 之后；管理接口额外使用 `auth.RequireAdmin`。
- 首个管理员引导生产环境应使用显式 bootstrap secret、OIDC 白名单或部署期初始化。
- Session cookie 生产环境必须设置 `Secure`，保持 `HttpOnly` 和合适的 `SameSite`。
- 登录、发言、添加 AI、LLM 决策、创建/加入房间需要速率限制。
- OIDC state 要有过期时间和清理机制。
- OIDC provider 配置只保存在服务端。
- 多 OIDC provider 的 `client_secret` 不得进入前端、日志或游戏代码。
- OIDC 回调必须校验 `state`，并按 provider 分别校验 issuer、client id、nonce 和 token。
- 静态响应建议加基础安全头：`X-Content-Type-Options`、`Referrer-Policy`、`X-Frame-Options` 或 CSP `frame-ancestors`，生产环境加 HSTS。
- 房间号不是权限边界；如果房间链接被视为私密入口，应使用 crypto random 并考虑短期邀请 token。

## 新游戏上线检查表

1. 读本文件、[前端游戏 UI 开发规范](./frontend-game-ui-guidelines.md) 和必要的 Actor 计划。
2. 按 [新游戏功能检查表](./game-feature-checklist.md) 写清人数、阶段、动作、结算、胜负条件和异常情况。
3. 对照真实规则，列出有意简化的部分。
4. 确定公开状态、私有状态、隐藏信息和 viewer 可见性。
5. 确定 AI 支持策略和 LLM function calling action。
6. 创建游戏注册、slug 页面、房间准备页和对局页。
7. 支持联机的游戏实现 HTTP、WebSocket、服务端权威状态和广播。
8. 生成或提供游戏卡片图和 SVG 图标。
9. 所有用户可见文案 i18n-ready。
10. 针对隐藏信息写不同 viewer 的 JSON 测试。
11. 针对 AI context 写最小披露测试。
12. 移动端检查关键操作是否符合人体工学。

## 验证策略

按风险选择验证范围。

常规：

```bash
pnpm test:server
pnpm lint:web
pnpm build:web
```

前端 UI 明显改动：

```bash
pnpm --dir web test
pnpm --dir web build
pnpm --dir web verify:ui
```

后端并发、Hub、AI、Actor 改动：

```bash
go test ./server/...
go test -race ./server/...
```

浏览器检查：

- 桌面横屏，例如 `1440x900` 或 `1366x768`。
- 手机竖屏，例如 `390x900`。
- 至少一个带 `?room=` 的联机 URL。
- 浏览器检查前必须重启后端。

## 相关文档

- [前端游戏 UI 开发规范](./frontend-game-ui-guidelines.md)
- [AI 玩家 Agent Actor 重构计划](./ai-player-agent-actor-architecture.md)
- [新游戏功能检查表](./game-feature-checklist.md)
- [AI Function Calling 规范](./ai-function-calling.md)
- [资产与 UI 指引](./assets-and-ui.md)
