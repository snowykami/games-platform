# 简单小游戏平台规格说明

## 1. 产品目标

构建一个基于 React + Vite 前端和后端 API/WebSocket 服务的简单小游戏平台。平台从一个轻量游戏大厅开始，通过 slug 路由进入不同游戏页面，后续逐步添加象棋、五子棋、麻将、Uno 等游戏。

第一版重点不在一次性实现所有游戏规则，而是先完成平台结构、路由、用户身份、房间加入和实时联机基础设施。具体游戏规则应封装在游戏适配器中，避免平台层变复杂。

## 2. 范围

### 2.1 MVP 范围

- 在一个页面展示所有可用游戏。
- 通过 slug 路由打开每个游戏，例如 `/games/gomoku`。
- 支持通过 `?room=xxxxx` 进入指定房间。
- 支持游客用户。
- 支持 AI 玩家制度，MVP 阶段使用规则驱动 AI。
- 为多 OIDC Provider 登录预留接口，但 MVP 不强制实现完整登录流程。
- 支持 WebSocket 联机。
- 提供最小房间生命周期：创建、加入、离开、重连、空房关闭。
- 提供游戏适配器约定，让每个游戏自行定义状态、动作和人数限制。

### 2.2 MVP 暂不包含

- 完整匹配队列。
- 排位、积分、排行榜、成就。
- 支付、物品、装扮系统。
- 生产级完整麻将规则。
- 语音或视频聊天。
- 超出服务端基础校验之外的复杂反作弊。
- 长期持久化回放存储。

## 3. 用户与身份

### 3.1 用户模式

| 模式 | 说明 | MVP 是否需要 |
| --- | --- | --- |
| 游客 | 自动生成展示名和会话 id 的匿名用户。 | 是 |
| OIDC | 通过一个或多个外部 OpenID Connect 提供方登录。 | 仅预留接口 |

### 3.2 游客会话

- 用户第一次打开应用时自动创建游客会话。
- 客户端在 localStorage 中保存游客 token 或 session id。
- 后端将游客视为临时用户。
- 游客展示名可以自动生成，后续可支持本地编辑。

### 3.3 多 OIDC Provider 会话

- OIDC 应作为可选身份提供方设计。
- OIDC 必须支持多个 provider，例如自建 Keycloak、Google、GitHub OIDC 代理或其他兼容 OpenID Connect 的服务。
- 每个 provider 使用稳定的内部 key 标识，例如 `keycloak`、`google`、`company-sso`。
- 登录入口不应写死单一 provider，前端应从后端获取可用 provider 列表。
- 平台对外暴露统一用户结构，不让游戏代码关心用户来自游客还是 OIDC。

```ts
type UserIdentity = {
  id: string;
  kind: "guest" | "oidc";
  displayName: string;
  avatarUrl?: string;
  providerKey?: string;
};
```

OIDC 账号绑定建议：

```ts
type OidcAccount = {
  providerKey: string;
  subject: string;
  userId: string;
  email?: string;
  displayName?: string;
};
```

约束：

- 同一个 `(providerKey, subject)` 只能绑定一个平台用户。
- 不同 provider 的 `subject` 不可直接比较，必须带上 `providerKey`。
- 用户表保存平台内用户，OIDC 账号表保存外部身份绑定。
- MVP 可先不做多账号合并，但数据模型应允许后续把多个 OIDC 账号绑定到同一个平台用户。

## 4. 路由

### 4.1 前端路由

| 路由 | 用途 |
| --- | --- |
| `/` | 游戏大厅页面。 |
| `/games/:slug` | 指定游戏页面。 |
| `/games/:slug?room=xxxxx` | 加入或打开指定联机房间。 |

### 4.2 路由行为

- 如果 `:slug` 不存在，展示未找到状态。
- 如果游戏不支持联机但 URL 中带有 `room`，忽略房间参数并展示本地/单机模式，同时给出轻量提示。
- 如果游戏支持联机且没有 `room`，展示创建房间和输入房间码入口，不直接进入对局。
- 如果 URL 中带有 `room`，在用户身份就绪后进入房间准备页，展示玩家列表、AI 玩家、复制链接和开始按钮。
- 房主开始游戏后才进入具体游戏桌面。

## 5. 游戏大厅

每个游戏通过平台级定义注册。

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

初始游戏列表建议：

| Slug | 名称 | 联机 | 本地 | MVP 状态 |
| --- | --- | --- | --- | --- |
| `gomoku` | 五子棋 | 是 | 是 | 首个可玩候选 |
| `xiangqi` | 象棋 | 是 | 是 | 后续 |
| `uno` | Uno | 是 | 否或可选 | 后续 |
| `mahjong` | 麻将 | 是 | 否 | 后续 |

## 6. 游戏页面体验

游戏页面包含三个概念区域：

- 游戏区域：棋盘、牌桌、手牌或其他主要操作界面。
- 会话面板：当前用户、玩家列表、房间号、连接状态。
- 操作区域：创建房间、复制邀请链接、加入房间、离开房间、准备/开始控制。

期望状态：

| 状态 | 说明 |
| --- | --- |
| Loading | 正在解析游戏定义、用户身份或房间。 |
| Local | 用户未进入 WebSocket 房间，本地游玩。 |
| Connecting | WebSocket 正在连接或重连。 |
| Lobby | 房间已存在，等待玩家或准备状态。 |
| Playing | 游戏适配器接管活跃游戏状态。 |
| Finished | 游戏结束，可按游戏规则重新开始。 |
| Error | 房间不存在、已满、不支持或连接失败。 |

## 7. 房间

### 7.1 房间 URL

联机游戏使用查询参数表达房间：

```text
/games/gomoku?room=G7K2QX
```

规则：

- 房间 id 应短小、URL 安全，并尽量大小写不敏感。
- 服务端返回的规范房间 id 应写回 URL。
- 复制邀请链接时复制完整绝对 URL。

### 7.2 房间模型

```ts
type Room = {
  id: string;
  gameSlug: string;
  status: "lobby" | "playing" | "finished";
  hostUserId: string;
  players: RoomPlayer[];
  spectators: RoomPlayer[];
  createdAt: string;
  updatedAt: string;
};

type RoomPlayer = {
  userId: string;
  displayName: string;
  kind: "human" | "ai";
  seat?: number;
  connected: boolean;
  ready: boolean;
};
```

### 7.3 房间生命周期

1. 客户端按游戏 slug 请求创建房间。
2. 服务端创建房间并返回 `roomId`。
3. 客户端更新 URL 为 `/games/:slug?room=:roomId`。
4. 客户端打开 WebSocket 并发送 `room.join`。
5. 服务端校验游戏是否支持联机、房间容量和用户身份。
6. 服务端向房间内所有连接广播房间状态。
7. 客户端渲染房间准备页，直到房主开始游戏。
8. 服务端广播游戏开始后，客户端进入具体游戏桌面。
9. 空房间在短暂宽限期后关闭。

## 8. WebSocket 协议

### 8.1 连接

建议端点：

```text
GET /ws
```

认证方式：

- 客户端在连接建立时，或连接后立即发送游客/OIDC access token。
- 服务端必须先解析出 `UserIdentity`，再接受房间动作。

### 8.2 消息信封

所有 WebSocket 消息使用统一信封。

```ts
type ClientMessage =
  | {
      type: "room.join";
      requestId: string;
      payload: { gameSlug: string; roomId: string };
    }
  | {
      type: "room.leave";
      requestId: string;
      payload: { roomId: string };
    }
  | {
      type: "room.ready";
      requestId: string;
      payload: { roomId: string; ready: boolean };
    }
  | {
      type: "game.action";
      requestId: string;
      payload: { roomId: string; action: unknown };
    };

type ServerMessage =
  | {
      type: "room.state";
      payload: { room: Room };
    }
  | {
      type: "game.state";
      payload: { roomId: string; state: unknown };
    }
  | {
      type: "error";
      requestId?: string;
      payload: { code: string; message: string };
    };
```

### 8.3 服务端权威

- 联机游戏以服务端状态为准。
- 客户端只发送意图动作。
- 游戏适配器负责校验动作是否合法。
- 非法动作返回错误，且不得修改游戏状态。

## 9. 游戏适配器约定

每个游戏实现统一适配器，让平台代码不依赖具体游戏规则。

```ts
type GameAdapter<State, Action> = {
  definition: GameDefinition;
  createInitialState(players: RoomPlayer[]): State;
  canStart(players: RoomPlayer[]): boolean;
  applyAction(state: State, action: Action, actor: UserIdentity): State;
  isFinished(state: State): boolean;
};
```

适配器职责：

- 定义初始游戏状态。
- 校验并应用玩家动作。
- 判断房间是否可以开始。
- 判断游戏是否结束。

平台职责：

- 将用户路由到正确页面。
- 管理身份、房间、Socket、重连和共享 UI。
- 持久化或广播房间与游戏状态。

## 10. 建议架构

### 10.0 仓库结构

建议使用一个仓库管理前端和后端，但保持构建系统简单。

```text
.
  apps/
    web/
    server/
  docs/
  packages/
    api-contract/
```

约定：

- `apps/web`：React + Vite 前端。
- `apps/server`：Go 后端。
- `packages/api-contract`：后续放 OpenAPI、WebSocket 协议说明或生成物；MVP 可先只保留文档。
- 前端使用 `pnpm`；后端使用 Go modules。
- 根目录可提供 `Makefile` 或 `Taskfile.yml` 聚合常用命令，但不强制引入复杂构建工具。

### 10.1 前端

推荐技术栈：

- React
- Vite
- TypeScript
- React Router
- Tailwind CSS
- shadcn/ui
- Radix UI primitives
- lucide-react
- TanStack Query
- Zustand
- antfu/eslint-config
- Vitest + Testing Library
- Playwright

建议结构：

```text
src/
  app/
    router.tsx
  auth/
    identity.ts
  games/
    registry.ts
    gomoku/
      GomokuPage.tsx
      adapter.ts
      types.ts
  rooms/
    roomClient.ts
    useRoom.ts
  styles/
    globals.css
  shared/
    components/
      ui/
    hooks/
    lib/
    types/
```

前端固定约定：

- 包管理器：优先使用 `pnpm`。
- 路由：使用 React Router 管理 `/`、`/games/:slug` 和查询参数 `room`。
- 样式：使用 Tailwind CSS，优先走 Vite 官方集成方式。
- 组件库：使用 shadcn/ui，组件代码放在 `src/shared/components/ui/`。
- 图标：使用 lucide-react。
- UI 基础：shadcn/ui 选择 `new-york` 风格、`neutral` 基色、CSS variables 模式。
- 服务端状态：使用 TanStack Query 管理 REST 请求缓存、加载态和错误态。
- 客户端临时状态：使用 Zustand 管理身份就绪状态、Socket 连接状态、房间面板 UI 状态等少量状态。
- WebSocket：使用原生 WebSocket 封装，不引入 socket.io，协议与后端 `ws/protocol.go` 对齐。
- 表单：简单表单可直接受控组件；复杂表单再引入 React Hook Form。
- 校验：前端使用 Zod 校验用户输入和 WebSocket 消息边界。
- 代码质量：使用 antfu/eslint-config 的 flat config，不再额外引入 Prettier。
- 测试：组件和 hooks 使用 Vitest + Testing Library，关键联机流程使用 Playwright。
- 路径别名：统一使用 `@/` 指向 `src/`。
- TypeScript：启用严格模式，禁止隐式 `any`。

### 10.1.1 前端设计系统约定

前端设计参考 `figma-generate-design` skill 的原则：优先复用设计系统组件、变量和样式，而不是临时手写零散样式。

- 页面以实际可用体验为第一屏，不做营销型 landing page。
- 游戏大厅是工作台式页面：清晰展示游戏列表、状态、人数、是否支持联机。
- 游戏详情页优先保证游戏区域清楚、房间状态可扫读、关键操作可快速点击。
- 不使用大面积装饰渐变、装饰光斑或纯视觉噪声。
- 颜色、圆角、间距、阴影统一通过 Tailwind/shadcn token 表达。
- 页面级布局可以手写 Tailwind class；按钮、输入框、弹窗、菜单、标签页、开关、提示等优先使用 shadcn/ui。
- 游戏卡片、房间面板、玩家列表等重复 UI 使用组件抽象。
- 卡片圆角默认不超过 `rounded-lg`，除非 shadcn/ui 默认组件已有明确样式。
- 工具按钮优先使用 lucide 图标，并提供 hover tooltip 或可访问标签。
- 桌面端优先双栏/三栏布局：游戏区域、房间/玩家面板、动作面板。
- 移动端优先单列布局，房间/动作面板折叠到游戏区域下方。
- 所有文字必须在移动端和桌面端都不溢出、不重叠。
- 完成明显 UI 变更后，使用浏览器截图检查桌面和移动视口。
- 具体游戏页面还必须遵循 [前端游戏 UI 开发规范](./frontend-game-ui-guidelines.md)；Uno 页面以 `/Users/sfkm/Downloads/xiangqi/` 的圆桌牌桌布局作为标准榜样。

### 10.2 后端

后端语言限制为 Go 或 Python。当前推荐优先选择 Go，因为平台核心包含较多 WebSocket 长连接、房间状态广播和服务端权威动作校验，Go 的并发模型更适合第一版联机能力。

推荐 Go 技术栈：

- HTTP：`net/http` + `chi`
- WebSocket：`coder/websocket`
- PostgreSQL：`pgx`
- SQL 类型生成：`sqlc`
- Redis：`go-redis`
- 数据库迁移：`goose` 或 `golang-migrate`
- OIDC：`coreos/go-oidc` + `golang.org/x/oauth2`

全局 AI 配置：

- `PORT`：后端 HTTP 服务端口，默认 `8901`。
- `DB_URL`：PostgreSQL 连接串，使用 `postgres://...` URL 格式。
- `REDIS_URL`：Redis 连接串，使用 `redis://...` URL 格式。
- `LLM_API`：所有 AI 玩家调用的统一 LLM API 地址。
- `LLM_MODEL`：所有 AI 玩家调用的统一 LLM 模型名称。
- `LLM_TOKEN`：所有 AI 玩家调用的统一 LLM token。
- 配置支持直接环境变量和本地 `.env`；真实环境变量优先，`.env` 只用于本地开发补齐未设置值。
- AI、OIDC、数据库、Redis 密钥只允许来自环境变量、忽略的本地 `.env` 或部署密钥，不进入前端、不写入日志。
- AI 玩家生成昵称、性格、说话风格时使用该全局 provider；具体游戏动作仍必须走各自游戏规则校验。

Python 备选技术栈：

- HTTP/API：FastAPI
- WebSocket：FastAPI WebSocket
- PostgreSQL：SQLAlchemy 2 async + asyncpg
- Redis：redis-py asyncio
- 数据库迁移：Alembic

第一版推荐使用模块化单体，不拆微服务。

建议结构：

```text
server/
  cmd/
    api/
      main.go
  internal/
    auth/
      identity.go
      guest.go
      oidc.go
      oidc_provider.go
    config/
      config.go
    games/
      registry.go
      gomoku/
        adapter.go
    rooms/
      store.go
      service.go
    ws/
      protocol.go
      server.go
  db/
    migrations/
    queries/
```

## 11. API 草案

### 11.1 REST

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/games` | 返回游戏列表。 |
| `POST` | `/api/auth/guest` | 创建或刷新游客身份。 |
| `GET` | `/api/auth/me` | 返回当前标准化用户身份。 |
| `GET` | `/api/auth/oidc/providers` | 返回可用 OIDC provider 列表。 |
| `GET` | `/api/auth/oidc/:providerKey/login` | 发起指定 provider 的 OIDC 登录。 |
| `GET` | `/api/auth/oidc/:providerKey/callback` | 处理指定 provider 的 OIDC 回调。 |
| `POST` | `/api/rooms` | 创建联机房间。 |
| `GET` | `/api/rooms/:roomId` | 返回房间元数据。 |

API 契约约定：

- REST API 后续使用 OpenAPI 3.1 描述。
- 前端可由 OpenAPI 生成请求类型，避免手写重复 DTO。
- WebSocket 协议先以 `docs/spec.md` 中的消息信封为准，稳定后可补 AsyncAPI。

### 11.2 创建房间请求

```ts
type CreateRoomRequest = {
  gameSlug: string;
};

type CreateRoomResponse = {
  roomId: string;
  url: string;
};
```

## 12. 错误处理

常见错误码：

| 错误码 | 含义 |
| --- | --- |
| `GAME_NOT_FOUND` | 未知游戏 slug。 |
| `GAME_OFFLINE_ONLY` | 不支持联机的游戏被请求房间。 |
| `ROOM_NOT_FOUND` | 房间不存在或已过期。 |
| `ROOM_FULL` | 房间没有可用玩家席位。 |
| `UNAUTHORIZED` | 缺少身份或身份无效。 |
| `OIDC_PROVIDER_NOT_FOUND` | 请求了不存在或未启用的 OIDC provider。 |
| `OIDC_STATE_INVALID` | OIDC 登录 state 无效或已过期。 |
| `OIDC_CALLBACK_FAILED` | OIDC 回调处理失败。 |
| `INVALID_ACTION` | 游戏适配器拒绝该动作。 |
| `CONNECTION_LOST` | 客户端断线且重连失败。 |

## 13. 持久化

MVP 可以使用内存存储：

- 游戏列表是静态代码。
- 游客会话可使用签名 token。
- 房间保存在内存中。
- 房间在无活动后过期。

后续持久化选项：

- Redis 用于房间和 Socket 协调。
- PostgreSQL 用于用户、资料、对局历史和回放。

## 14. 安全与校验

- 永远不要信任客户端游戏状态。
- 服务端校验每个房间动作。
- 使用注册表校验 `gameSlug`。
- 查询房间前先校验 `roomId` 格式。
- 限制创建房间和 WebSocket 消息频率。
- OIDC provider 配置只保存在服务端。
- 多 OIDC provider 的 `client_secret` 不得进入前端、日志或游戏代码。
- OIDC 登录回调必须校验 `state`，并按 provider 分别校验 issuer、client id、nonce 和 token。
- 不把原始 provider token 暴露给游戏代码。

## 15. 实现里程碑

1. 搭建 React + Vite + TypeScript 前端与后端工作区。
2. 配置 Tailwind CSS、shadcn/ui、antfu/eslint-config、路径别名和基础测试工具。
3. 实现游戏注册表和大厅页面。
4. 实现 `/games/:slug` 路由和未知游戏状态。
5. 实现游客身份。
6. 实现房间创建 API 和 URL 更新。
7. 实现 WebSocket 连接与房间加入/离开。
8. 添加规则驱动 AI 玩家制度，先在 Uno 中验证。
9. 添加最小五子棋适配器，作为首个真实可玩游戏。
10. 添加重连处理和房间清理。
11. 添加可选多 OIDC Provider 登录。
12. 通过适配器继续添加更多游戏。

## 16. 工程原则

- KISS：平台层只负责大厅、路由、身份、房间和传输。
- YAGNI：第一个联机游戏跑通前，不添加排行、持久化和复杂匹配。
- DRY：创建工作区后，共享前后端房间、身份和 WebSocket 协议类型。
- SOLID：把具体游戏规则隔离在适配器中，新增游戏不应迫使平台层重写。
