# AI Function Calling 规范

LLM 只作为决策层，不拥有规则权威。规则 bot 仍应提供稳定 baseline。

## AI 等级

- `beginner`：弱策略，允许随机合法动作或明显启发式失误。
- `normal`：简单合法启发式，稳定快速。
- `master`：更强启发式，优先胜利、阻挡、素材/分数、节奏和风险。
- `ai`：LLM 决策，仅在 `LLM_API`、`LLM_MODEL`、`LLM_TOKEN` 配置可用时开放。

如果创建房间 AI 时请求 `ai` 但 provider 不可用，应拒绝或降级为 UI 不可选。通用决策 endpoint 可在明确记录日志后 fallback 到 deterministic strategy。

## Function Tool

默认只暴露一个 function：

```json
{
  "name": "choose_action",
  "description": "Choose one legal action for the current game turn.",
  "parameters": {
    "type": "object",
    "properties": {
      "actionId": {
        "type": "string",
        "description": "The exact id of one action from the legal actions list."
      },
      "reason": {
        "type": "string",
        "description": "Short reason in Chinese."
      },
      "speech": {
        "type": "string",
        "description": "Optional short table talk in Chinese."
      },
      "notes": {
        "type": "object",
        "additionalProperties": {"type": "string"}
      }
    },
    "required": ["actionId"],
    "additionalProperties": false
  }
}
```

`notes` 只允许一层 `map[string]string`。服务端需要兼容部分 provider 把 map stringify 的情况。

## Prompt 输入

System message：

```text
You are a game AI. You must choose exactly one action from the provided legal actions. Do not invent actions. Reply only by calling choose_action.
```

User message 使用扁平 JSON：

- `game`
- `level`
- `sessionId`
- `playerName`
- `personality`
- `speechStyle`
- `state`：该 AI 当前允许知道的紧凑 public/private state
- `actions`：legal actions only

社交推理游戏的 `state` 必须使用座位别名，禁止暴露 `kind`、`isAI`、`userId`、`aiProfile`、`connected` 或其他账号/人机身份字段。

## 校验

必须校验：

- Provider 已配置。
- Legal actions 非空。
- 返回的 `actionId` 精确匹配某个 legal action。
- LLM 调用使用平台共享 timeout，默认 30 秒。
- 失败、超时、非法 action 走 deterministic fallback 或跳过，并打结构化日志。

禁止：

- 要求 LLM 返回状态 patch。
- 让 LLM 看到该玩家不该知道的隐藏信息。
- 让 LLM 的解释影响规则执行。
- 在等待 LLM 时阻塞整个房间。
- 在日志里记录 token。

## Action ID

Action id 必须稳定、可解析、可校验：

- Uno：`play:<cardId>:<color>` 或 `draw`。
- 五子棋：`place:<x>:<y>`。
- 象棋：`move:<pieceId>:<x>:<y>`。
- 麻将：`discard:<tileId>`、`claim:<claimId>`、`draw`。
- 社交推理：`vote:seat_2`、`target:seat_3`、`say:clue`。

优先使用服务端可校验对象 id。社交推理游戏对 LLM 使用座位别名，服务端应用前再映射回内部 player id。
