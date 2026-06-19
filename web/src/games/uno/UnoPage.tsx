import type { CSSProperties, ReactNode } from 'react'
import type { UnoOnlinePlayer, UnoOnlineRoom } from './online'
import type { UnoCard, UnoColor } from './types'
import { ArrowLeft, Copy, RotateCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { cn } from '@/shared/lib/utils'
import { useUnoRoom } from './useUnoRoom'

const UNO_COLORS: UnoColor[] = ['red', 'yellow', 'green', 'blue']

const COLOR_STYLES: Record<UnoCard['color'], string> = {
  blue: 'bg-[#2768c7] text-[#fff8e8]',
  green: 'bg-[#1f9b55] text-[#fff8e8]',
  red: 'bg-[#d72f2f] text-[#fff8e8]',
  wild: 'bg-[linear-gradient(135deg,#d72f2f_0_25%,#f3c33c_25%_50%,#1f9b55_50%_75%,#2768c7_75%)] text-[#fff8e8]',
  yellow: 'bg-[#f3c33c] text-[#241915] [text-shadow:none]',
}

const COLOR_SWATCHES: Record<UnoColor, string> = {
  blue: 'bg-[#2768c7]',
  green: 'bg-[#1f9b55]',
  red: 'bg-[#d72f2f]',
  yellow: 'bg-[#f3c33c]',
}

export function UnoPage({ roomId }: { roomId: string }) {
  const { user } = useAuth()
  const { actions, error, isLoading, room } = useUnoRoom(roomId)
  const [selectedWildColor, setSelectedWildColor] = useState<UnoColor>('red')
  const [message, setMessage] = useState('服务端联机房间已接入。')

  const human = useMemo(() => room?.players.find(player => player.userId === user?.id), [room?.players, user?.id])
  const currentPlayer = room?.players.find(player => player.id === room.currentPlayerId)
  const winner = room?.players.find(player => player.id === room.winnerId)
  const hand = useMemo(() => human?.hand ?? [], [human?.hand])
  const isHumanTurn = Boolean(room && human && room.phase === 'playing' && room.currentPlayerId === human.id)
  const playableCards = useMemo(() => room ? hand.filter(card => isCardPlayable(card, room)) : [], [hand, room])
  const aiPlayerCount = room?.players.filter(player => player.isAI).length ?? 0
  const lastLog = room?.log.at(-1)?.text

  async function handleCopyRoom() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage('链接已复制。')
  }

  async function handlePlay(card: UnoCard) {
    if (!room || !isHumanTurn || !isCardPlayable(card, room)) {
      return
    }

    await actions.play(card.id, selectedWildColor)
    setMessage(`你打出了 ${formatCard(card)}。`)
  }

  async function handleDraw() {
    if (!isHumanTurn) {
      return
    }

    await actions.draw()
    setMessage('你摸牌并结束回合。')
  }

  async function handleRestart() {
    await actions.start()
    setMessage('新的一局开始。')
  }

  if (isLoading || !room || !human || !room.topCard) {
    return (
      <main className="grid h-svh place-items-center overflow-hidden bg-[#15110e] px-4 text-[#fff8e8]">
        <p className="text-sm font-black">{error ?? '正在进入 UNO 房间...'}</p>
      </main>
    )
  }

  return (
    <main className="h-svh overflow-hidden bg-[#15110e] text-[#fff8e8]">
      <div className="mx-auto grid h-full w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_auto_minmax(0,1fr)_auto] gap-2 py-2 sm:gap-3 sm:py-3">
        <header className="flex min-h-0 items-end justify-between gap-4">
          <div>
            <p className="mb-1 text-[11px] font-black tracking-normal text-[#fff8e8]/75 sm:text-xs">ONLINE UNO TABLE</p>
            <h1 className="text-[clamp(38px,7vw,82px)] font-black leading-[0.82] tracking-normal [text-shadow:0_7px_0_rgba(20,19,16,0.35)]">
              UNO
            </h1>
          </div>
          <Link
            className="inline-grid min-h-9 place-items-center rounded-full border border-white/40 bg-[#141310]/50 px-3 text-sm font-bold text-[#fff8e8] transition hover:bg-[#141310]/70 sm:min-h-10 sm:px-4"
            to="/"
          >
            <ArrowLeft className="mr-2 inline size-4" />
            游戏大厅
          </Link>
        </header>

        <section className="contents">
          <div className="flex min-h-0 flex-wrap items-center gap-1.5 rounded-lg border border-white/20 bg-[#100d0b]/50 p-1.5 shadow-[0_18px_46px_rgba(0,0,0,0.22)] backdrop-blur-md sm:gap-2 sm:p-2">
            <button className="uno-button uno-button-primary" type="button" onClick={handleCopyRoom}>
              <Copy className="size-4" />
              复制链接
            </button>
            <StatusPill>{room.phase === 'finished' ? '已结束' : '进行中'}</StatusPill>
            <StatusPill>
              回合：
              {currentPlayer?.name ?? '-'}
            </StatusPill>
            <StatusPill>
              当前颜色：
              {room.activeColor ? formatColor(room.activeColor) : '-'}
            </StatusPill>
            <StatusPill>
              方向：
              {room.direction === 1 ? '顺时针' : '逆时针'}
            </StatusPill>
            <StatusPill>
              AI：
              {aiPlayerCount}
            </StatusPill>
            <StatusPill>
              房间：
              {room.id}
            </StatusPill>
            <button className="uno-button ml-auto" disabled={room.hostUserId !== user?.id} type="button" onClick={handleRestart}>
              <RotateCcw className="size-4" />
              重开
            </button>
          </div>

          <section
            aria-label="UNO 圆桌"
            className="relative min-h-0 overflow-hidden rounded-lg border border-white/25 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.08),transparent_58%),rgba(11,10,9,0.42)] shadow-[inset_0_0_90px_rgba(0,0,0,0.42),0_24px_70px_rgba(0,0,0,0.3)]"
          >
            <div className="absolute inset-3 rounded-full border-[12px] border-[#49331d] bg-[radial-gradient(ellipse_at_center,rgba(255,248,232,0.08),transparent_52%),repeating-linear-gradient(90deg,rgba(255,255,255,0.025)_0_1px,transparent_1px_22px),#16533e] shadow-[inset_0_0_0_3px_rgba(255,248,232,0.18),inset_0_0_90px_rgba(0,0,0,0.32),0_24px_80px_rgba(0,0,0,0.36)] sm:inset-5 sm:border-[16px]">
              <div className="absolute left-1/2 top-1/2 z-20 grid w-[min(230px,48vw)] -translate-x-1/2 -translate-y-1/2 grid-cols-[64px_76px] items-center justify-center gap-2 text-center sm:w-[min(290px,44vw)] sm:grid-cols-[82px_96px] sm:gap-3.5">
                <button
                  aria-label="摸牌"
                  className="uno-card-back aspect-[2/3] w-16 rounded-xl p-0 transition hover:-translate-y-1 disabled:cursor-not-allowed disabled:opacity-45 sm:w-[82px]"
                  disabled={!isHumanTurn}
                  type="button"
                  onClick={handleDraw}
                />
                <div className="grid min-h-24 place-items-center sm:min-h-32">
                  <UnoCardView card={room.topCard} className="table-card w-[68px] sm:w-[86px]" />
                </div>
                <div className="col-span-full grid gap-0.5 rounded-lg bg-[#090807]/60 px-2 py-1.5 text-xs sm:gap-1 sm:px-3 sm:py-2 sm:text-sm">
                  <strong>{room.phase === 'finished' ? `${winner?.name ?? '玩家'} 获胜` : `${currentPlayer?.name ?? '-'} 的回合`}</strong>
                  <span>
                    当前颜色：
                    {room.activeColor ? formatColor(room.activeColor) : '-'}
                  </span>
                  <span>
                    牌堆
                    {' '}
                    {room.drawPileCount}
                  </span>
                </div>
              </div>

              {room.players.map((player, index) => (
                <PlayerSeat
                  key={player.id}
                  currentPlayerId={room.currentPlayerId}
                  hand={player.userId === user?.id ? hand : undefined}
                  index={index}
                  player={player}
                  total={room.players.length}
                />
              ))}
            </div>
          </section>

          <section className="min-h-0 rounded-lg border border-white/20 bg-[#100d0b]/50 p-2 shadow-[0_18px_46px_rgba(0,0,0,0.22)] backdrop-blur-md sm:p-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <strong>
                  你的手牌
                  {' '}
                  {hand.length}
                </strong>
                <p className="mt-0.5 text-xs font-semibold text-[#fff8e8]/75 sm:text-sm">
                  {isHumanTurn ? '选择发光的牌出牌，万能牌会使用右侧颜色。' : `等待 ${currentPlayer?.name ?? '玩家'} 行动。`}
                </p>
              </div>
              <div className="flex flex-wrap gap-1.5 sm:gap-2">
                {UNO_COLORS.map(color => (
                  <button
                    key={color}
                    aria-label={`选择${formatColor(color)}`}
                    className={cn(
                      'min-h-8 w-10 rounded-lg border-2 border-[#fff8e8]/70 transition hover:-translate-y-0.5 sm:min-h-10 sm:w-12',
                      COLOR_SWATCHES[color],
                      selectedWildColor === color && 'outline outline-3 outline-[#fff8e8]',
                    )}
                    type="button"
                    onClick={() => setSelectedWildColor(color)}
                  />
                ))}
              </div>
            </div>

            <div className="mt-2 flex max-h-[18svh] min-h-0 flex-wrap items-end gap-2 overflow-y-auto pb-1 sm:max-h-36 sm:gap-2.5">
              {hand.map(card => (
                <button
                  key={card.id}
                  className="p-0 transition disabled:cursor-not-allowed disabled:opacity-40"
                  disabled={!isHumanTurn || !isCardPlayable(card, room)}
                  type="button"
                  onClick={() => handlePlay(card)}
                >
                  <UnoCardView
                    card={card}
                    className={cn(
                      'w-[clamp(48px,6.2vw,76px)] cursor-pointer sm:w-[clamp(58px,7vw,82px)]',
                      isHumanTurn && isCardPlayable(card, room) && '-translate-y-1.5 outline outline-3 outline-[#fff8e8]',
                    )}
                  />
                </button>
              ))}
            </div>

            <p className="mt-1 min-h-5 text-xs font-black text-[#fff8e8] [overflow-wrap:anywhere] sm:mt-2 sm:text-sm">
              {error ?? lastLog ?? message}
              {' '}
              可出牌：
              {playableCards.length}
              {' '}
              张
            </p>
          </section>
        </section>
      </div>
    </main>
  )
}

function StatusPill({ children }: { children: ReactNode }) {
  return (
    <span className="inline-grid min-h-8 place-items-center rounded-full border border-white/25 bg-[#141310]/40 px-2.5 text-xs font-extrabold sm:min-h-9 sm:px-3 sm:text-sm">
      {children}
    </span>
  )
}

function PlayerSeat({
  currentPlayerId,
  hand,
  index,
  player,
  total,
}: {
  currentPlayerId?: string
  hand?: UnoCard[]
  index: number
  player: UnoOnlinePlayer
  total: number
}) {
  return (
    <article
      className={cn(
        'absolute z-30 grid min-h-[60px] w-[104px] -translate-x-1/2 -translate-y-1/2 justify-items-center gap-1 rounded-lg border border-white/25 bg-[#090807]/50 p-1.5 shadow-[0_16px_34px_rgba(0,0,0,0.24)] sm:min-h-[76px] sm:w-[150px] sm:gap-1.5 sm:p-2 lg:min-h-[82px] lg:w-[170px]',
        currentPlayerId === player.id && 'outline outline-3 outline-[#fff8e8]',
      )}
      style={seatStyle(index, total)}
    >
      <div className="flex max-w-full items-center gap-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-black text-[#fff8e8] max-[560px]:text-xs">
        {player.name}
        {player.role === 'host' && <span className="rounded-full bg-[#fff8e8] px-1.5 text-[11px] text-[#171411]">房主</span>}
        {player.isAI && <span className="rounded-full bg-[#fff8e8] px-1.5 text-[11px] text-[#171411]">AI</span>}
      </div>
      {hand ? <MiniFaceHand hand={hand} /> : <MiniBackHand count={player.handCount} />}
    </article>
  )
}

function seatStyle(index: number, total: number): CSSProperties {
  if (index === 0) {
    return { left: '50%', top: '88%' }
  }

  const opponents = Math.max(total - 1, 1)
  const spread = opponents > 5 ? 160 : 132
  const start = 270 - spread / 2
  const step = opponents === 1 ? 0 : spread / (opponents - 1)
  const angle = (start + step * (index - 1)) * Math.PI / 180
  const x = 50 + Math.cos(angle) * 43
  const y = 50 + Math.sin(angle) * 36

  return {
    left: `${x.toFixed(2)}%`,
    top: `${y.toFixed(2)}%`,
  }
}

function MiniBackHand({ count }: { count: number }) {
  const visible = Math.min(count, 12)
  const cardKeys = Array.from({ length: visible }, (_, slot) => `back-${count}-${slot}`)

  return (
    <div className="flex min-h-8 w-20 items-center justify-center sm:min-h-10 sm:w-28 lg:w-36">
      {cardKeys.map((key, index) => (
        <span
          key={key}
          className="uno-mini-card-back h-7 w-[18px] rounded-[5px] sm:h-8 sm:w-[21px] lg:h-[38px] lg:w-[25px]"
          style={{ marginLeft: index === 0 ? 0 : -9, transform: `translateY(${index % 3 * 2}px)` }}
        />
      ))}
    </div>
  )
}

function MiniFaceHand({ hand }: { hand: UnoCard[] }) {
  return (
    <div className="flex min-h-8 w-20 flex-wrap items-center justify-center gap-1 sm:min-h-10 sm:w-28 lg:w-36">
      {hand.slice(0, 9).map(card => (
        <span
          key={card.id}
          className={cn('grid h-7 w-[18px] place-items-center rounded-[5px] border-2 border-[#fff8e8] text-[10px] font-black sm:h-8 sm:w-[21px] lg:h-9 lg:w-6 lg:text-[11px]', COLOR_STYLES[card.color])}
        >
          {cardLabel(card)}
        </span>
      ))}
    </div>
  )
}

function UnoCardView({ card, className }: { card: UnoCard, className?: string }) {
  return (
    <div
      className={cn(
        'relative grid aspect-[2/3] place-items-center rounded-xl border-[3px] border-[#fff8e8] font-black shadow-[0_16px_30px_rgba(0,0,0,0.28)] [text-shadow:0_2px_0_rgba(0,0,0,0.26)] before:absolute before:inset-3 before:rounded-full before:border-2 before:border-white/45 before:content-[""] before:-rotate-[22deg]',
        COLOR_STYLES[card.color],
        className,
      )}
      title={formatCard(card)}
    >
      <span className="z-10 text-2xl">{cardLabel(card)}</span>
    </div>
  )
}

function isCardPlayable(card: UnoCard, room: UnoOnlineRoom) {
  if (!room.topCard) {
    return true
  }

  return card.color === 'wild'
    || card.color === room.activeColor
    || card.kind === room.topCard.kind
    || (card.kind === 'number' && room.topCard.kind === 'number' && card.value === room.topCard.value)
}

function formatColor(color: UnoColor) {
  const labels: Record<UnoColor, string> = {
    blue: '蓝色',
    green: '绿色',
    red: '红色',
    yellow: '黄色',
  }

  return labels[color]
}

function formatCard(card: UnoCard) {
  if (card.kind === 'number') {
    return `${card.color} ${card.value}`
  }
  return `${card.color} ${card.kind}`
}

function cardLabel(card: UnoCard) {
  if (card.kind === 'number') {
    return card.value
  }

  const labels: Record<UnoCard['kind'], string> = {
    'draw-two': '+2',
    'number': '',
    'reverse': '转',
    'skip': '禁',
    'wild': '变',
    'wild-draw-four': '+4',
  }

  return labels[card.kind]
}
