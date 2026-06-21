# Games Platform

一个基于 React + Vite 前端和 Go 后端的轻量小游戏平台。当前已实现游戏总览、用户登录入口、UNO 联机房间、服务端权威 UNO 状态、WebSocket 同步、AI 玩家和 Docker 单体镜像。

## 功能状态

- 游戏总览页：展示已注册游戏。
- 游戏 slug 路由：`/games/:slug`。
- UNO：
  - `/games/uno` 创建或加入房间。
  - `/games/uno?room=ROOMID` 进入指定房间。
  - 房主可添加 AI、开始游戏。
  - 出牌、摸牌、AI 自动行动由后端校验和推进。
- 用户系统：
  - 未登录进入游戏页会跳转登录页。
  - 支持游客登录，浏览器本地保存匿名 UUID。
  - 支持多 OIDC provider 配置。
  - 第一个用户默认成为平台管理员。
  - 角色：平台管理员、普通玩家。
  - 管理员接口支持封禁/解封用户。
- 部署：
  - 后端构建时自动构建前端到 `server/internal/web/dist`。
  - Go 使用 `embed` 将前端静态资源打入后端二进制。
  - Docker 镜像只运行 Go 后端，默认端口 `8901`。

## 技术栈

- 前端：React 19、Vite、TypeScript、Tailwind CSS、shadcn 风格本地组件、TanStack Query、Zod、Antfu ESLint。
- 后端：Go、chi、coder/websocket、go-oidc、oauth2、godotenv。
- 数据库规划：PostgreSQL，连接串使用 `DB_URL=postgres://...`。
- 缓存/会话规划：Redis，连接串使用 `REDIS_URL=redis://...`。

当前用户、房间和 UNO 对局状态仍是内存实现；`DB_URL` 和 `REDIS_URL` 已进入配置层，后续持久化时接入。AI 玩家当前仍是规则驱动，LLM provider 配置已预留 `LLM_API`、`LLM_MODEL` 和 `LLM_TOKEN`。

## 目录结构

```text
.
├── docs/                    # 项目指引、前端游戏 UI 规范和 AI Actor 重构计划
├── server/                  # Go 后端
│   ├── cmd/api/             # API 服务入口
│   └── internal/
│       ├── auth/            # 用户、会话、角色、OIDC
│       ├── config/          # 环境配置
│       ├── games/           # 游戏注册表
│       ├── httpx/           # HTTP JSON 工具
│       ├── uno/             # UNO 房间、规则、WebSocket
│       └── web/             # embed 前端静态资源
└── web/                     # React + Vite 前端
```

## 环境变量

配置支持直接环境变量和本地 `.env` 文件。真实环境变量优先，`.env` 只补齐本地未设置的值。

复制示例文件：

```bash
cp .env.example .env
```

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

`OIDC_PROVIDERS_JSON` 是 JSON 数组，例如：

```bash
OIDC_PROVIDERS_JSON='[{"key":"google","displayName":"Google","issuerUrl":"https://accounts.google.com","clientId":"xxx","clientSecret":"xxx","redirectUrl":"http://localhost:5173/api/auth/oidc/google/callback"}]'
```

生产环境建议设置 `APP_ENV=production` 或 `SESSION_COOKIE_SECURE=true`，确保 session cookie 只通过 HTTPS 发送。

## 本地开发

安装依赖：

```bash
pnpm install
```

启动本地 PostgreSQL 和 Redis：

```bash
pnpm dev:infra
```

启动后端：

```bash
pnpm dev:server
```

Go 后端不会热重载；修改 `server/` 下代码后需要停止并重启 `pnpm dev:server`，再进行浏览器验证。

启动前端开发服务器：

```bash
pnpm dev:web
```

开发模式下前端 Vite 代理会把 `/api` 和 `/ws` 转发到 `http://localhost:8901`。OIDC 开发回调建议配置为 `http://localhost:5173/api/auth/oidc/<provider>/callback`，由 Vite 统一反代到后端，避免浏览器跨域和回调域名不一致。

## 单体构建

构建单体二进制：

```bash
pnpm build:server
```

产物路径：

```text
dist/games-platform
```

访问：

```text
http://127.0.0.1:8901/
```

## Docker

Docker 构建会在镜像内安装前端依赖、构建 Vite 产物，再构建嵌入静态资源的 Go 后端；不需要提前提交或上传前端构建产物。

构建镜像：

```bash
docker build -t games-platform:local .
```

运行容器：

```bash
docker run --rm -p 8901:8901 --env-file .env games-platform:local
```

如果本机 `8901` 已被占用，可以换宿主机端口：

```bash
docker run --rm -p 8902:8901 --env-file .env games-platform:local
```

## 验证

后端测试：

```bash
go test ./server/...
```

前端 lint、构建和测试：

```bash
pnpm --dir web lint
pnpm --dir web build
pnpm --dir web test
```

UI 冒烟测试默认访问 `http://127.0.0.1:5173`：

```bash
pnpm --dir web verify:ui
```

验证 embed 或 Docker 形态时传入 `BASE_URL`：

```bash
BASE_URL=http://127.0.0.1:8901 pnpm --dir web verify:ui
```

## 相关文档

- [项目指引](docs/project-guidelines.md)
- [前端游戏 UI 开发规范](docs/frontend-game-ui-guidelines.md)
- [AI 玩家 Agent Actor 重构计划](docs/ai-player-agent-actor-architecture.md)
- [新游戏功能检查表](docs/game-feature-checklist.md)
- [AI Function Calling 规范](docs/ai-function-calling.md)
- [资产与 UI 指引](docs/assets-and-ui.md)

## 当前限制

- PG 和 Redis 尚未接入真实读写，当前状态保存在进程内存中。
- OIDC 已支持多 provider 配置和标准登录回调，但需要实际 provider 配置后验证。
- AI 玩家当前是规则驱动；`LLM_API` / `LLM_MODEL` / `LLM_TOKEN` 已预留为后续统一 AI provider。
