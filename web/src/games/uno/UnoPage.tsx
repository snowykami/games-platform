import type { CSSProperties, ReactNode } from 'react'
import type { UnoOnlinePlayer, UnoPublicAction } from './online'
import type { UnoCard, UnoColor } from './types'
import type { GameSpeechEntry } from '@/games/speech'
import { ArrowLeft, Ban, Copy, Palette, Plus, RefreshCw, RotateCcw, RotateCw, SkipForward } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { usePendingAction } from '@/games/usePendingAction'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { UnoVariantInfoButton } from './UnoVariantInfo'
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
  const { t } = useI18n()
  const { actions, error, isLoading, room } = useUnoRoom(roomId)
  const [message, setMessage] = useState(() => t('uno.tableReady'))
  const [activeAction, setActiveAction] = useState<UnoPublicAction>()
  const [singleClickPlay, setSingleClickPlay] = useState(() => window.localStorage.getItem('uno-single-click-play') === 'true')
  const [pendingCardId, setPendingCardId] = useState<string>()
  const [wildPicker, setWildPicker] = useState<{ card: UnoCard, color: UnoColor }>()
  const pending = usePendingAction()
  const lastActionSeqRef = useRef(0)

  const human = useMemo(() => room?.players.find(player => player.id === room.youPlayerId), [room?.players, room?.youPlayerId])
  const seatedPlayers = useMemo(() => room && human ? orderPlayersFromViewer(room.players, human.id) : [], [human, room])
  const currentPlayer = room?.players.find(player => player.id === room.currentPlayerId)
  const winner = room?.players.find(player => player.id === room.winnerId)
  const hand = useMemo(() => human?.hand ?? [], [human?.hand])
  const isHumanTurn = Boolean(room && human && room.phase === 'playing' && room.currentPlayerId === human.id)
  const playableCardIds = useMemo(() => new Set(room?.playableCardIds ?? []), [room?.playableCardIds])
  const playableCards = useMemo(() => hand.filter(card => playableCardIds.has(card.id)), [hand, playableCardIds])
  const aiPlayerCount = room?.players.filter(player => player.isAI).length ?? 0
  const catchableUnoPlayers = room?.players.filter(player => player.id !== room.youPlayerId && player.needsUno) ?? []
  const lastLog = room?.log.at(-1)?.text
  const selectedCardId = isHumanTurn && hand.some(card => card.id === pendingCardId) ? pendingCardId : undefined
  const activeWildPicker = wildPicker && isHumanTurn && hand.some(card => card.id === wildPicker.card.id) ? wildPicker : undefined

  useEffect(() => {
    window.localStorage.setItem('uno-single-click-play', String(singleClickPlay))
  }, [singleClickPlay])

  useEffect(() => {
    if (!room) {
      return undefined
    }

    const nextActions = room.recentActions.filter(action => action.seq > lastActionSeqRef.current)
    if (!nextActions.length) {
      return undefined
    }

    lastActionSeqRef.current = Math.max(...nextActions.map(action => action.seq))
    const timers: number[] = []
    nextActions.forEach((action, index) => {
      // Timers are collected and cleared together in the effect cleanup.
      // eslint-disable-next-line react/web-api-no-leaked-timeout
      const startTimer = window.setTimeout(() => {
        setActiveAction(action)
      }, index * 260)
      // eslint-disable-next-line react/web-api-no-leaked-timeout
      const endTimer = window.setTimeout(() => {
        setActiveAction(current => current?.seq === action.seq ? undefined : current)
      }, index * 260 + 920)

      timers.push(startTimer, endTimer)
    })

    return () => {
      timers.forEach(timer => window.clearTimeout(timer))
    }
  }, [room])

  useEffect(() => {
    pending.clearAll()
  }, [pending, pending.clearAll, room?.actionSeq, room?.currentPlayerId, room?.phase])

  async function handleCopyRoom() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  async function handlePlay(card: UnoCard) {
    if (!room || !isHumanTurn || !playableCardIds.has(card.id)) {
      return
    }
    if (card.color === 'wild') {
      setPendingCardId(card.id)
      setWildPicker({ card, color: chooseBestWildColor(hand, card.id, room.activeColor) })
      setMessage(t('uno.chooseColorFirst'))
      return
    }
    if (!singleClickPlay && selectedCardId !== card.id) {
      setPendingCardId(card.id)
      setMessage(t('uno.confirmCard'))
      return
    }

    await playConfirmedCard(card, card.color)
  }

  async function playConfirmedCard(card: UnoCard, color: UnoColor) {
    await pending.run(`play:${card.id}`, () => actions.play(card.id, color), { releaseOnSettle: false })
    setPendingCardId(undefined)
    setWildPicker(undefined)
    setMessage(t('uno.playedCard', { card: formatCard(card) }))
  }

  async function handleDraw() {
    if (!isHumanTurn) {
      return
    }

    await pending.run('draw', () => actions.draw(), { releaseOnSettle: false })
    setMessage(t('uno.drewCard'))
  }

  async function handleRestart() {
    await pending.run('restart', () => actions.start(), { releaseOnSettle: false })
    setMessage(t('uno.restarted'))
  }

  async function handleCallUno() {
    await pending.run('call-uno', () => actions.callUno(), { releaseOnSettle: false })
    setMessage('UNO!')
  }

  async function handleCatchUno(targetId: string) {
    await pending.run(`catch:${targetId}`, () => actions.catchUno(targetId), { releaseOnSettle: false })
    setMessage(t('uno.caughtUno'))
  }

  if (isLoading || !room || !human || !room.topCard) {
    return (
      <main className="grid h-svh place-items-center overflow-hidden bg-[#15110e] px-4 text-[#fff8e8]">
        <p className="text-sm font-black">{error ?? t('uno.loading')}</p>
      </main>
    )
  }

  return (
    <main className="min-h-svh overflow-y-auto bg-[#15110e] text-[#fff8e8] lg:h-svh lg:overflow-hidden">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_auto_minmax(0,1fr)] gap-2 py-2 sm:gap-3 sm:py-3 lg:h-full lg:min-h-0">
        <header className="flex min-h-0 items-end justify-between gap-4">
          <div>
            <p className="mb-0.5 text-[11px] font-black tracking-normal text-[#fff8e8]/75 sm:text-xs">ONLINE UNO TABLE</p>
            <h1 className="text-sm font-black leading-none tracking-normal text-[#fff8e8]/92 sm:text-base">
              UNO
            </h1>
          </div>
          <Link
            className="inline-grid min-h-9 place-items-center rounded-full border border-white/40 bg-[#141310]/50 px-3 text-sm font-bold text-[#fff8e8] transition hover:bg-[#141310]/70 sm:min-h-10 sm:px-4"
            to="/"
          >
            <ArrowLeft className="mr-2 inline size-4" />
            {t('common.backToLobby')}
          </Link>
        </header>

        <section className="contents">
          <div className="flex min-h-0 flex-wrap items-center gap-1.5 rounded-lg border border-white/20 bg-[#100d0b]/50 p-1.5 shadow-[0_18px_46px_rgba(0,0,0,0.22)] backdrop-blur-md sm:gap-2 sm:p-2">
            <button className="uno-button uno-button-primary" type="button" onClick={handleCopyRoom}>
              <Copy className="size-4" />
              {t('common.copyLink')}
            </button>
            <StatusPill>{room.phase === 'finished' ? t('uno.finished') : t('uno.playing')}</StatusPill>
            <StatusPill>
              {t('uno.turn')}
              {currentPlayer?.name ?? '-'}
            </StatusPill>
            {room.phase === 'playing' && room.turnDeadline && (
              <StatusPill>
                {t('uno.turnCountdown', { seconds: room.turnRemainingSeconds })}
              </StatusPill>
            )}
            <StatusPill>
              <span>{t('uno.currentColor')}</span>
              {room.activeColor ? <ColorDot color={room.activeColor} /> : <span>-</span>}
            </StatusPill>
            <StatusPill>
              {room.direction === 1 ? <RotateCw className="size-4" /> : <RotateCcw className="size-4" />}
              <span>{room.direction === 1 ? t('uno.clockwise') : t('uno.counterClockwise')}</span>
            </StatusPill>
            <StatusPill>
              AI：
              {aiPlayerCount}
            </StatusPill>
            <StatusPill>
              {t('common.room')}
              {room.id}
            </StatusPill>
            <UnoVariantInfoButton variantKey={room.variantKey} />
            {room.pendingDrawCount > 0 && (
              <StatusPill>
                {t('uno.penalty')}
                {room.pendingDrawCount}
              </StatusPill>
            )}
            {(room.rules.flip || room.flipSide) && <StatusPill>{room.flipSide ? t('uno.darkSide') : t('uno.lightSide')}</StatusPill>}
            <button className="uno-button ml-auto" disabled={pending.isPending('restart') || room.hostPlayerId !== room.youPlayerId} type="button" onClick={handleRestart}>
              <RotateCcw className="size-4" />
              {pending.isPending('restart') ? t('common.syncing') : t('uno.restart')}
            </button>
          </div>

          <section
            aria-label={t('uno.tableLabel')}
            className="grid min-h-0 grid-rows-[minmax(0,1fr)_auto] overflow-hidden rounded-lg border border-white/25 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.08),transparent_58%),rgba(11,10,9,0.42)] shadow-[inset_0_0_90px_rgba(0,0,0,0.42),0_24px_70px_rgba(0,0,0,0.3)]"
          >
            <div className="relative min-h-[300px] overflow-hidden sm:min-h-[360px]">
              <div className="absolute inset-3 rounded-full border-[12px] border-[#49331d] bg-[radial-gradient(ellipse_at_center,rgba(255,248,232,0.08),transparent_52%),repeating-linear-gradient(90deg,rgba(255,255,255,0.025)_0_1px,transparent_1px_22px),#16533e] shadow-[inset_0_0_0_3px_rgba(255,248,232,0.18),inset_0_0_90px_rgba(0,0,0,0.32),0_24px_80px_rgba(0,0,0,0.36)] sm:inset-5 sm:border-[16px]">
                <div className="absolute left-1/2 top-1/2 z-20 grid w-[min(230px,48vw)] -translate-x-1/2 -translate-y-1/2 grid-cols-[64px_76px] items-center justify-center gap-2 text-center sm:w-[min(290px,44vw)] sm:grid-cols-[82px_96px] sm:gap-3.5">
                  <button
                    aria-label={t('uno.drawPile')}
                    className="uno-card-back aspect-[2/3] w-16 rounded-xl p-0 transition hover:-translate-y-1 disabled:cursor-not-allowed disabled:opacity-45 sm:w-[82px]"
                    disabled={!isHumanTurn || pending.isPending('draw')}
                    type="button"
                    onClick={handleDraw}
                  />
                  <div className="grid min-h-24 place-items-center sm:min-h-32">
                    <UnoCardView card={room.topCard} className="table-card w-[68px] sm:w-[86px]" />
                  </div>
                  <div className="col-span-full grid gap-0.5 rounded-lg bg-[#090807]/60 px-2 py-1.5 text-xs sm:gap-1 sm:px-3 sm:py-2 sm:text-sm">
                    <strong>{room.phase === 'finished' ? t('uno.winner', { name: winner?.name ?? t('common.player') }) : t('uno.playerTurn', { name: currentPlayer?.name ?? '-' })}</strong>
                    <span>
                      {t('uno.currentColor')}
                      {room.activeColor ? formatColor(room.activeColor, t) : '-'}
                    </span>
                    <span>
                      {t('uno.discardPile')}
                      {' '}
                      {room.drawPileCount}
                    </span>
                  </div>
                </div>

                {seatedPlayers.map((player, index) => (
                  <PlayerSeat
                    key={player.id}
                    activeAction={activeAction}
                    currentPlayerId={room.currentPlayerId}
                    index={index}
                    isSelf={player.id === room.youPlayerId}
                    onRename={actions.renamePlayer}
                    onSpeak={actions.say}
                    player={player}
                    speech={latestSpeechForPlayer(room.speeches, player.id)}
                    total={seatedPlayers.length}
                  />
                ))}
                <ActionLayer action={activeAction} players={seatedPlayers} />
              </div>
            </div>

            <div className="min-h-0 bg-[#100d0b]/62 px-2 pb-2 pt-5 shadow-[inset_0_24px_36px_rgba(0,0,0,0.18)] backdrop-blur-md sm:px-3 sm:pb-3 sm:pt-6">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <strong>
                    {t('uno.yourHand')}
                    {' '}
                    {hand.length}
                  </strong>
                  <p className="mt-0.5 text-xs font-semibold text-[#fff8e8]/75 sm:text-sm">
                    {isHumanTurn ? t('uno.playableHint') : t('uno.waiting', { name: currentPlayer?.name ?? t('common.player') })}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <SpeechButton onSend={actions.say} />
                  {human.needsUno && (
                    <button className="uno-button uno-button-primary" disabled={pending.isPending('call-uno')} type="button" onClick={handleCallUno}>
                      {pending.isPending('call-uno') ? t('common.syncing') : 'UNO'}
                    </button>
                  )}
                  {catchableUnoPlayers.map(player => (
                    <button key={player.id} className="uno-button" disabled={pending.isPending(`catch:${player.id}`)} type="button" onClick={() => handleCatchUno(player.id)}>
                      {pending.isPending(`catch:${player.id}`) ? t('common.syncing') : t('uno.catchUno')}
                      {' '}
                      {player.name}
                    </button>
                  ))}
                  <label className="inline-flex min-h-9 items-center gap-2 rounded-full border border-white/25 bg-[#141310]/44 px-3 text-xs font-black text-[#fff8e8] sm:text-sm">
                    <input
                      checked={singleClickPlay}
                      className="size-4 accent-[#33f3ff]"
                      type="checkbox"
                      onChange={event => setSingleClickPlay(event.currentTarget.checked)}
                    />
                    {t('uno.clickToPlay')}
                  </label>
                </div>
              </div>

              <div className="mt-3 flex max-h-[18svh] min-h-0 flex-wrap items-end gap-2 overflow-y-auto overflow-x-hidden pb-1 pt-3 sm:max-h-36 sm:gap-2.5 sm:pt-4">
                {hand.map(card => (
                  <button
                    key={card.id}
                    className="p-0 transition disabled:cursor-not-allowed disabled:opacity-40"
                    disabled={!isHumanTurn || !playableCardIds.has(card.id) || pending.isPending(`play:${card.id}`)}
                    type="button"
                    onClick={() => handlePlay(card)}
                  >
                    <UnoCardView
                      card={card}
                      className={cn(
                        'w-[clamp(48px,6.2vw,76px)] cursor-pointer sm:w-[clamp(58px,7vw,82px)]',
                        isHumanTurn && playableCardIds.has(card.id) && '-translate-y-1.5 outline outline-3 outline-[#fff8e8]',
                        selectedCardId === card.id && 'uno-card-selected',
                      )}
                    />
                  </button>
                ))}
              </div>

              <p className="mt-1 min-h-5 text-xs font-black text-[#fff8e8] [overflow-wrap:anywhere] sm:mt-2 sm:text-sm">
                {error ?? lastLog ?? message}
                {' '}
                {t('uno.playableCards')}
                {playableCards.length}
                {' '}
                {t('uno.cards')}
              </p>
            </div>
          </section>
        </section>
      </div>
      {activeWildPicker && (
        <div className="fixed inset-0 z-[10000] grid place-items-center bg-[#090807]/72 px-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label={t('uno.chooseWildColor')}>
          <div className="relative z-[10001] w-[min(360px,calc(100vw-32px))] rounded-lg border border-white/25 bg-[#14110e] p-4 text-[#fff8e8] shadow-[0_28px_80px_rgba(0,0,0,0.45)]">
            <div className="flex items-center justify-between gap-3">
              <strong>{t('uno.chooseColor')}</strong>
              <span className="text-xs font-black text-[#fff8e8]/65">{formatCard(activeWildPicker.card)}</span>
            </div>
            <div className="mt-4 grid grid-cols-4 gap-2">
              {UNO_COLORS.map(color => (
                <button
                  key={color}
                  aria-label={t('uno.chooseNamedColor', { color: formatColor(color, t) })}
                  className={cn(
                    'grid min-h-16 place-items-center rounded-lg border-2 border-[#fff8e8]/70 transition hover:-translate-y-0.5',
                    COLOR_SWATCHES[color],
                    activeWildPicker.color === color && 'outline outline-4 outline-[#33f3ff]',
                  )}
                  type="button"
                  disabled={pending.isPending(`play:${activeWildPicker.card.id}`)}
                  onClick={() => playConfirmedCard(activeWildPicker.card, color)}
                >
                  <span className="sr-only">{formatColor(color, t)}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </main>
  )
}

function StatusPill({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex min-h-8 items-center justify-center gap-2 rounded-full border border-white/25 bg-[#141310]/40 px-2.5 text-xs font-extrabold sm:min-h-9 sm:px-3 sm:text-sm">
      {children}
    </span>
  )
}

function ColorDot({ color }: { color: UnoColor }) {
  const { t } = useI18n()
  return <span aria-label={formatColor(color, t)} className={cn('size-4 rounded-full border-2 border-[#fff8e8]', COLOR_SWATCHES[color])} />
}

function PlayerSeat({
  activeAction,
  currentPlayerId,
  index,
  isSelf,
  onRename,
  onSpeak,
  player,
  speech,
  total,
}: {
  activeAction?: UnoPublicAction
  currentPlayerId?: string
  index: number
  isSelf: boolean
  onRename: (name: string) => Promise<void>
  onSpeak: (text: string) => Promise<void>
  player: UnoOnlinePlayer
  speech?: GameSpeechEntry
  total: number
}) {
  const shouldPulseSeat = activeAction && activeAction.type !== 'play'
  const isActionTarget = shouldPulseSeat && (activeAction.actorId === player.id || activeAction.targetId === player.id)

  return (
    <article
      data-seat-id={player.id}
      className={cn(
        'absolute z-30 grid min-h-[60px] w-[104px] -translate-x-1/2 -translate-y-1/2 justify-items-center gap-1 rounded-lg border border-white/25 bg-[#090807]/50 p-1.5 shadow-[0_16px_34px_rgba(0,0,0,0.24)] sm:min-h-[76px] sm:w-[150px] sm:gap-1.5 sm:p-2 lg:min-h-[82px] lg:w-[170px]',
        isSelf && 'min-h-[42px] content-center sm:min-h-[48px] lg:min-h-[52px]',
        currentPlayerId === player.id && 'uno-current-seat',
        isActionTarget && 'uno-seat-action',
      )}
      style={seatStyle(index, total)}
    >
      <div className="flex w-full min-w-0 items-center justify-center gap-1 text-sm font-black text-[#fff8e8] max-[560px]:text-xs">
        <div className="flex min-w-0 items-center gap-1">
          <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
          <span className="min-w-0 truncate">{player.name}</span>
        </div>
        {player.role === 'host' && <HostBadge />}
        {player.isAI && <span className="shrink-0 rounded-full bg-[#fff8e8] px-1.5 text-[10px] leading-5 text-[#171411]">AI</span>}
        {isSelf && <PlayerNameEditor buttonClassName="text-[#fff8e8]" className="min-w-[96px]" name={player.name} onSave={onRename} />}
        {isSelf && <SpeechButton className="ml-1" onSend={onSpeak} />}
      </div>
      <SpeechBubble className="max-w-[130px] sm:max-w-[170px]" speech={speech} />
      {!isSelf && <MiniBackHand count={player.handCount} />}
    </article>
  )
}

interface ActionPoint {
  x: string
  y: string
}

interface FlightStyle extends CSSProperties {
  '--from-x': string
  '--from-y': string
  '--to-x': string
  '--to-y': string
}

const DRAW_PILE_POINT: ActionPoint = { x: '45%', y: '50%' }
const DISCARD_POINT: ActionPoint = { x: '55%', y: '50%' }

function ActionLayer({ action, players }: { action?: UnoPublicAction, players: UnoOnlinePlayer[] }) {
  if (!action) {
    return null
  }

  const isPlay = action.type === 'play' && action.card
  const isDraw = action.type === 'draw'
  const actorPoint = pointForPlayer(players, action.actorId)
  const drawTargetPoint = pointForPlayer(players, action.targetId ?? action.actorId)
  const playStyle = flightStyle(actorPoint, DISCARD_POINT)

  return (
    <div className="pointer-events-none absolute inset-0 z-40 overflow-hidden">
      {isPlay && action.card && (
        <div className="uno-flying-card uno-fly-to-discard" style={playStyle}>
          <UnoCardView card={action.card} className="w-[68px] sm:w-[78px]" />
        </div>
      )}
      {isDraw && Array.from({ length: Math.min(action.count ?? 1, 4) }, (_, index) => (
        <span
          key={`${action.seq}-${index}`}
          className="uno-flying-back uno-fly-to-seat"
          style={flightStyle(DRAW_PILE_POINT, drawTargetPoint, index * 90)}
        />
      ))}
      <div className="uno-action-toast">{action.message}</div>
    </div>
  )
}

function pointForPlayer(players: UnoOnlinePlayer[], playerId: string): ActionPoint {
  const index = players.findIndex(player => player.id === playerId)
  const style = seatStyle(index < 0 ? 0 : index, players.length || 1)

  return {
    x: String(style.left ?? '50%'),
    y: String(style.top ?? '84%'),
  }
}

function flightStyle(from: ActionPoint, to: ActionPoint, delayMs = 0): FlightStyle {
  return {
    '--from-x': from.x,
    '--from-y': from.y,
    '--to-x': to.x,
    '--to-y': to.y,
    'animationDelay': `${delayMs}ms`,
  }
}

function orderPlayersFromViewer(players: UnoOnlinePlayer[], viewerPlayerId: string) {
  const viewerIndex = players.findIndex(player => player.id === viewerPlayerId)
  if (viewerIndex < 0) {
    return players
  }

  return [...players.slice(viewerIndex), ...players.slice(0, viewerIndex)]
}

function seatStyle(index: number, total: number): CSSProperties {
  if (index === 0) {
    return { left: '50%', top: '84%' }
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
    <>
      <div className="relative grid min-h-8 w-12 place-items-center sm:hidden">
        <span className="uno-mini-card-back h-8 w-[22px] rounded-[5px]" />
        <span className="absolute -right-1 -top-1 grid min-h-5 min-w-5 place-items-center rounded-full border border-[#15110e] bg-[#fff8e8] px-1 text-[11px] font-black leading-none text-[#15110e]">
          {count}
        </span>
      </div>
      <div className="hidden min-h-10 w-28 items-center justify-center sm:flex lg:w-36">
        {cardKeys.map((key, index) => (
          <span
            key={key}
            className="uno-mini-card-back h-8 w-[21px] rounded-[5px] lg:h-[38px] lg:w-[25px]"
            style={{ marginLeft: index === 0 ? 0 : -9, transform: `translateY(${index % 3 * 2}px)` }}
          />
        ))}
      </div>
    </>
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
      <span className="z-10 grid place-items-center text-2xl"><CardFaceSymbol card={card} /></span>
    </div>
  )
}

function HostBadge() {
  const { t } = useI18n()
  return <span className="shrink-0 whitespace-nowrap rounded-full bg-[#fff8e8] px-1.5 text-[10px] leading-5 text-[#171411]">{t('uno.host')}</span>
}

function formatColor(color: UnoColor, t: (key: string) => string) {
  return t(`uno.colors.${color}`)
}

function chooseBestWildColor(hand: UnoCard[], playedCardId: string, activeColor?: UnoColor): UnoColor {
  const score: Record<UnoColor, number> = {
    blue: 0,
    green: 0,
    red: 0,
    yellow: 0,
  }

  hand.forEach((card) => {
    if (card.id === playedCardId || card.color === 'wild') {
      return
    }
    score[card.color]++
  })

  return UNO_COLORS.reduce((best, color) => {
    if (score[color] > score[best]) {
      return color
    }
    if (score[color] === score[best] && activeColor === color) {
      return color
    }
    return best
  }, activeColor ?? 'red')
}

function formatCard(card: UnoCard) {
  if (card.kind === 'number') {
    return `${card.color} ${card.value}`
  }
  return `${card.color} ${card.kind}`
}

function CardFaceSymbol({ card, compact = false }: { card: UnoCard, compact?: boolean }) {
  if (card.kind === 'number') {
    return card.value
  }

  const iconSize = compact ? 12 : 28
  const iconStroke = compact ? 4 : 3

  const icons: Record<Exclude<UnoCard['kind'], 'number'>, ReactNode> = {
    'draw-two': (
      <span className="inline-flex items-center gap-0.5">
        <Plus size={iconSize} strokeWidth={iconStroke} />
        <span>2</span>
      </span>
    ),
    'reverse': <RefreshCw size={iconSize} strokeWidth={iconStroke} />,
    'skip': <Ban size={iconSize} strokeWidth={iconStroke} />,
    'wild': <Palette size={iconSize} strokeWidth={iconStroke} />,
    'wild-draw-four': (
      <span className="inline-flex items-center gap-0.5">
        <Plus size={iconSize} strokeWidth={iconStroke} />
        <span>4</span>
      </span>
    ),
    'wild-draw-six': (
      <span className="inline-flex items-center gap-0.5">
        <Plus size={iconSize} strokeWidth={iconStroke} />
        <span>6</span>
      </span>
    ),
    'wild-draw-ten': (
      <span className="inline-flex items-center gap-0.5">
        <Plus size={iconSize} strokeWidth={iconStroke} />
        <span>10</span>
      </span>
    ),
    'flip': <RotateCw size={iconSize} strokeWidth={iconStroke} />,
  }

  if (card.kind === 'skip' && compact) {
    return <SkipForward size={iconSize} strokeWidth={iconStroke} />
  }

  return icons[card.kind]
}
