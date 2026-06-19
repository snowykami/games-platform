import type { FormEvent, ReactNode } from 'react'
import { ArrowLeft, Bot, Copy, DoorOpen, Plus, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { cn } from '@/shared/lib/utils'
import { createUnoRoom } from './online'
import { UnoPage } from './UnoPage'
import { useUnoRoom } from './useUnoRoom'

interface UnoRoomGateProps {
  roomId?: string
}

const UNO_VARIANTS = [
  { key: 'classic', name: '经典 UNO', description: '标准数字牌、禁、转向、+2、变色和 +4。' },
  { key: 'party', name: '派对狂欢', description: '预留 +6、+10、叠加惩罚等疯狂规则。' },
  { key: 'stacking', name: '叠加规则', description: '预留 +2/+4 连锁叠加与反制。' },
]

const UNO_THEMES = [
  { key: 'classic', name: '经典牌面', description: '高识别度四色牌。' },
  { key: 'neon', name: '霓虹牌面', description: '预留赛博高亮风格。' },
  { key: 'anime-collab', name: '联动牌面', description: '预留 IP 联动和角色主题牌。' },
]

export function UnoRoomGate({ roomId }: UnoRoomGateProps) {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { actions, error, isLoading, room } = useUnoRoom(roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState('朋友局，链接开桌。先进入房间，再开始游戏。')
  const [pendingAI, setPendingAI] = useState(false)
  const [variantKey, setVariantKey] = useState('classic')
  const [themeKey, setThemeKey] = useState('classic')
  const isHost = Boolean(user && room?.hostUserId === user.id)

  if (roomId && room && room.phase !== 'lobby') {
    return <UnoPage roomId={roomId} />
  }

  async function createRoom() {
    setMessage('正在创建服务端房间...')
    try {
      const nextRoom = await createUnoRoom({ themeKey, variantKey })

      navigate(`/games/uno?room=${nextRoom.id}`)
      setJoinCode(nextRoom.id)
      setMessage('房间已创建，复制链接就能邀请朋友。')
    }
    catch (err) {
      setMessage(err instanceof Error ? err.message : '创建房间失败。')
    }
  }

  function joinRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const normalizedCode = joinCode.trim().toUpperCase()

    if (!normalizedCode) {
      setMessage('链接里没有房间信息，请输入房间号。')
      return
    }

    navigate(`/games/uno?room=${encodeURIComponent(normalizedCode)}`)
    setMessage('已进入房间，等待房主开始。')
  }

  function copyLink() {
    navigator.clipboard?.writeText(window.location.href)
    setMessage('链接已复制。')
  }

  async function addAIPlayer() {
    if (pendingAI || !room) {
      return
    }

    setPendingAI(true)
    setMessage('AI 正在加入房间...')
    try {
      await actions.addAI()
      setMessage('AI 已加入房间。')
    }
    finally {
      setPendingAI(false)
    }
  }

  async function startGame() {
    setMessage('正在开始游戏...')
    await actions.start()
    setMessage('游戏已开始。')
  }

  if (!roomId) {
    return (
      <UnoShell>
        <section className="grid min-h-[min(560px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="uno-panel relative grid content-end overflow-hidden p-5 sm:p-6">
            <div className="pointer-events-none absolute -right-8 top-2 rotate-[-12deg] text-[clamp(120px,24vw,270px)] font-black leading-none text-white/20">
              UNO
            </div>
            <div className="relative max-w-xl">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">朋友局，链接开桌</h2>
              <p className="mt-3 text-sm leading-7 text-[#fff8e8]/80 sm:text-base">创建房间后复制当前链接，朋友打开会先登录，再进入同一个服务端房间。</p>
            </div>
          </div>

          <form className="uno-panel grid content-start gap-4 p-5 sm:p-6" onSubmit={joinRoom}>
            <h2 className="text-2xl font-black tracking-normal">进入 UNO</h2>
            <button className="uno-button uno-button-primary w-full" type="button" onClick={createRoom}>
              <Plus className="size-4" />
              创建并进入
            </button>
            <OptionGroup label="玩法类型" options={UNO_VARIANTS} value={variantKey} onChange={setVariantKey} />
            <OptionGroup label="牌面主题" options={UNO_THEMES} value={themeKey} onChange={setThemeKey} />
            <label className="grid gap-2 text-sm font-black" htmlFor="uno-room-code">
              房间号
              <input
                id="uno-room-code"
                className="min-h-11 rounded-lg border border-white/35 bg-[#141310]/40 px-3 uppercase text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#fff8e8]"
                placeholder="ROOM42"
                value={joinCode}
                onChange={event => setJoinCode(event.target.value)}
              />
            </label>
            <button className="uno-button w-full" type="submit">
              <DoorOpen className="size-4" />
              加入房间
            </button>
            <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{message}</p>
          </form>
        </section>
      </UnoShell>
    )
  }

  return (
    <UnoShell>
      <section className="grid min-h-[min(600px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="uno-panel grid grid-rows-[auto_minmax(0,1fr)_auto] gap-4 p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black text-[#fff8e8]/65">ROOM</p>
              <h2 className="text-2xl font-black tracking-normal">
                房间
                {' '}
                {roomId}
              </h2>
              <p className="mt-1 text-xs font-bold text-[#fff8e8]/60">
                {variantLabel(room?.variantKey)}
                {' / '}
                {themeLabel(room?.themeKey)}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className="uno-button" type="button" onClick={copyLink}>
                <Copy className="size-4" />
                复制链接
              </button>
              <button
                className={cn('uno-button', pendingAI && 'loading')}
                disabled={pendingAI || !isHost || !room || room.players.length >= 10}
                type="button"
                onClick={addAIPlayer}
              >
                <Bot className="size-4" />
                {pendingAI ? '生成中...' : '添加 AI'}
              </button>
            </div>
          </div>

          <div className="grid content-start gap-3 overflow-auto pr-1 sm:grid-cols-2">
            {isLoading && <p className="text-sm font-bold text-[#fff8e8]/70">正在连接房间...</p>}
            {room?.players.map(player => (
              <article key={player.id} className="rounded-lg border border-white/25 bg-[#090807]/50 p-4 shadow-[0_16px_34px_rgba(0,0,0,0.2)]">
                <div className="flex items-center justify-between gap-3">
                  <strong className="truncate text-lg">{player.name}</strong>
                  <span className="rounded-full bg-[#fff8e8] px-2 py-0.5 text-xs font-black text-[#171411]">
                    {player.role === 'host' ? '房主' : player.isAI ? 'AI' : player.connected ? '在线' : '离线'}
                  </span>
                </div>
                <p className="mt-2 min-h-10 text-sm leading-6 text-[#fff8e8]/72">
                  {player.ai?.personality ?? (player.kind === 'guest' ? '游客玩家，公开资料仅显示游客名。' : 'OIDC 玩家，等待开始。')}
                </p>
              </article>
            ))}
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <button className="uno-button uno-button-primary sm:w-40" disabled={!isHost || !room || room.players.length < 2} type="button" onClick={startGame}>
              开始游戏
            </button>
            <p className="text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
          </div>
        </div>

        <aside className="uno-panel grid content-start gap-3 p-5">
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            AI 制度
          </h2>
          <RuleLine>房主只能在准备页添加 AI。</RuleLine>
          <RuleLine>AI 是正式玩家，进入座位、手牌、回合和胜负流程。</RuleLine>
          <RuleLine>当前服务端使用规则 AI；后续统一通过 LLM_API / LLM_TOKEN 生成 AI 人格。</RuleLine>
          <RuleLine>AI 出牌走服务端动作校验，不绕过规则。</RuleLine>
        </aside>
      </section>
    </UnoShell>
  )
}

function UnoShell({ children }: { children: ReactNode }) {
  return (
    <main className="min-h-svh overflow-hidden bg-[#15110e] text-[#fff8e8]">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
        <header className="flex items-end justify-between gap-4">
          <div>
            <p className="mb-1 text-xs font-black tracking-normal text-[#fff8e8]/75">LAN UNO TABLE</p>
            <h1 className="text-[clamp(44px,8vw,86px)] font-black leading-[0.82] tracking-normal [text-shadow:0_7px_0_rgba(20,19,16,0.35)]">
              UNO
            </h1>
          </div>
          <Link
            className="inline-grid min-h-10 place-items-center rounded-full border border-white/40 bg-[#141310]/50 px-4 text-sm font-bold text-[#fff8e8] transition hover:bg-[#141310]/70"
            to="/"
          >
            <ArrowLeft className="mr-2 inline size-4" />
            游戏大厅
          </Link>
        </header>
        {children}
      </div>
    </main>
  )
}

function RuleLine({ children }: { children: ReactNode }) {
  return <p className="rounded-lg bg-[#090807]/50 px-3 py-2 text-sm leading-6 text-[#fff8e8]/78">{children}</p>
}

function OptionGroup({
  label,
  onChange,
  options,
  value,
}: {
  label: string
  onChange: (value: string) => void
  options: Array<{ description: string, key: string, name: string }>
  value: string
}) {
  return (
    <fieldset className="grid gap-2">
      <legend className="text-sm font-black">{label}</legend>
      <div className="grid gap-2">
        {options.map(option => (
          <button
            key={option.key}
            className={cn(
              'grid rounded-lg border border-white/20 bg-[#090807]/42 px-3 py-2 text-left transition hover:border-[#fff8e8]/70',
              value === option.key && 'border-[#f3c33c] bg-[#f3c33c]/15 shadow-[0_0_0_2px_rgba(243,195,60,0.18)]',
            )}
            type="button"
            onClick={() => onChange(option.key)}
          >
            <strong className="text-sm">{option.name}</strong>
            <span className="mt-1 text-xs leading-5 text-[#fff8e8]/64">{option.description}</span>
          </button>
        ))}
      </div>
    </fieldset>
  )
}

function variantLabel(key?: string) {
  return UNO_VARIANTS.find(item => item.key === key)?.name ?? '经典 UNO'
}

function themeLabel(key?: string) {
  return UNO_THEMES.find(item => item.key === key)?.name ?? '经典牌面'
}
