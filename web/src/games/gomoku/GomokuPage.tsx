import type { ReactNode } from 'react'
import type { GomokuMove, GomokuPlayer, GomokuStone } from './online'
import type { GameSpeechEntry } from '@/games/speech'
import { ArrowLeft, Copy, RotateCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { useGomokuRoom } from './useGomokuRoom'

export function GomokuPage({ roomId }: { roomId: string }) {
  const { user } = useAuth()
  const { t } = useI18n()
  const { actions, error, isLoading, room } = useGomokuRoom(roomId)
  const [message, setMessage] = useState(() => t('gomoku.ready'))

  const human = useMemo(() => room?.players.find(player => player.userId === user?.id), [room?.players, user?.id])
  const currentPlayer = room?.players.find(player => player.id === room.currentPlayerId)
  const winner = room?.players.find(player => player.id === room.winnerId)
  const moveMap = useMemo(() => {
    const cells = new Map<string, GomokuMove>()
    room?.moves.forEach(move => cells.set(cellKey(move.x, move.y), move))
    return cells
  }, [room?.moves])
  const winningCells = useMemo(() => new Set(room?.winningLine.map(point => cellKey(point.x, point.y)) ?? []), [room?.winningLine])
  const lastMove = room?.moves.at(-1)
  const isHumanTurn = Boolean(room && human && room.phase === 'playing' && room.currentPlayerId === human.id)
  const isHost = Boolean(room && user && room.hostUserId === user.id)

  async function handleCopyRoom() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  async function handlePlace(x: number, y: number) {
    if (!isHumanTurn || moveMap.has(cellKey(x, y))) {
      return
    }

    await actions.place(x, y)
    setMessage(t('gomoku.placedAt', { point: formatPoint(x, y) }))
  }

  async function handleRestart() {
    await actions.start()
    setMessage(t('gomoku.restarted'))
  }

  if (isLoading || !room || !human) {
    return (
      <main className="grid h-svh place-items-center overflow-hidden bg-[#101714] px-4 text-[#f4f0e4]">
        <p className="text-sm font-black">{error ?? t('gomoku.loading')}</p>
      </main>
    )
  }

  return (
    <main className="min-h-svh overflow-hidden bg-[#101714] text-[#f4f0e4]">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_auto_minmax(0,1fr)] gap-2 py-2 sm:gap-3 sm:py-3">
        <header className="flex min-h-0 items-end justify-between gap-4">
          <div>
            <p className="mb-1 text-[11px] font-black tracking-normal text-[#f4f0e4]/75 sm:text-xs">ONLINE GOMOKU BOARD</p>
            <h1 className="text-[clamp(38px,7vw,80px)] font-black leading-[0.82] tracking-normal [text-shadow:0_7px_0_rgba(0,0,0,0.35)]">
              {t('gomoku.title')}
            </h1>
          </div>
          <Link
            className="inline-grid min-h-9 place-items-center rounded-full border border-white/36 bg-[#0b1110]/55 px-3 text-sm font-bold text-[#f4f0e4] transition hover:bg-[#0b1110]/72 sm:min-h-10 sm:px-4"
            to="/"
          >
            <ArrowLeft className="mr-2 inline size-4" />
            {t('common.backToLobby')}
          </Link>
        </header>

        <div className="flex min-h-0 flex-wrap items-center gap-1.5 rounded-lg border border-white/20 bg-[#0b1110]/55 p-1.5 shadow-[0_18px_46px_rgba(0,0,0,0.22)] backdrop-blur-md sm:gap-2 sm:p-2">
          <button className="gomoku-button gomoku-button-primary" type="button" onClick={handleCopyRoom}>
            <Copy className="size-4" />
            {t('common.copyLink')}
          </button>
          <StatusPill>{room.phase === 'finished' ? t('gomoku.finished') : t('gomoku.playing')}</StatusPill>
          <StatusPill>
            {t('gomoku.turn')}
            {currentPlayer?.name ?? '-'}
          </StatusPill>
          <StatusPill>
            {t('gomoku.moves')}
            {room.moves.length}
          </StatusPill>
          <StatusPill>
            {t('common.room')}
            {room.id}
          </StatusPill>
          <button className="gomoku-button ml-auto" disabled={!isHost} type="button" onClick={handleRestart}>
            <RotateCcw className="size-4" />
            {t('gomoku.restart')}
          </button>
        </div>

        <section className="grid min-h-0 gap-3 overflow-hidden lg:grid-cols-[minmax(0,1fr)_320px]">
          <div className="gomoku-panel grid min-h-0 place-items-center overflow-hidden p-2 sm:p-3">
            <div className="w-full max-w-[min(86svh,760px)]">
              <div className="gomoku-board mx-auto grid aspect-square w-full grid-cols-[repeat(15,minmax(0,1fr))] overflow-hidden rounded-lg border border-[#3a2514] p-2 shadow-[inset_0_0_0_2px_rgba(255,255,255,0.18),0_26px_70px_rgba(0,0,0,0.28)] sm:p-3">
                {Array.from({ length: room.boardSize * room.boardSize }, (_, index) => {
                  const x = index % room.boardSize
                  const y = Math.floor(index / room.boardSize)
                  const move = moveMap.get(cellKey(x, y))
                  const isWinning = winningCells.has(cellKey(x, y))
                  const isLast = lastMove?.x === x && lastMove.y === y

                  return (
                    <button
                      key={`${x}-${y}`}
                      aria-label={`${formatPoint(x, y)} ${move ? formatStone(move.stone, t) : t('gomoku.emptyPoint')}`}
                      className={cn(
                        'gomoku-cell group relative grid aspect-square place-items-center',
                        isHumanTurn && !move && 'cursor-pointer',
                        !isHumanTurn && 'cursor-default',
                      )}
                      disabled={!isHumanTurn || Boolean(move)}
                      type="button"
                      onClick={() => handlePlace(x, y)}
                    >
                      <span className="pointer-events-none absolute inset-0 border-r border-b border-[#4a321b]/70" />
                      {move && <StoneView isLast={isLast} isWinning={isWinning} stone={move.stone} />}
                      {!move && isHumanTurn && <span className="size-[44%] rounded-full bg-[#1f8f7b]/0 transition group-hover:bg-[#1f8f7b]/28" />}
                    </button>
                  )
                })}
              </div>
            </div>
          </div>

          <aside className="grid min-h-0 gap-3 overflow-hidden">
            <div className="gomoku-panel grid content-start gap-3 p-4">
              <h2 className="text-xl font-black">
                {room.phase === 'finished'
                  ? room.isDraw ? t('gomoku.draw') : t('gomoku.winner', { name: winner?.name ?? t('common.player') })
                  : isHumanTurn ? t('gomoku.yourTurn') : t('gomoku.thinking', { name: currentPlayer?.name ?? '-' })}
              </h2>
              <p className="min-h-6 text-sm font-bold text-[#f4f0e4]/72">{error ?? message}</p>
              <div className="grid gap-2">
                {room.players.map(player => (
                  <PlayerLine
                    key={player.id}
                    active={player.id === room.currentPlayerId}
                    player={player}
                    self={player.userId === user?.id}
                    speech={latestSpeechForPlayer(room.speeches, player.id)}
                    onRename={actions.renamePlayer}
                    onSpeak={actions.say}
                  />
                ))}
              </div>
            </div>

            <div className="gomoku-panel min-h-0 overflow-hidden p-4">
              <h2 className="mb-3 text-lg font-black">{t('gomoku.history')}</h2>
              <div className="grid max-h-60 gap-2 overflow-auto pr-1">
                {room.log.map(entry => (
                  <p key={entry.id} className="rounded-lg bg-[#0b1110]/58 px-3 py-2 text-sm font-bold leading-6 text-[#f4f0e4]/76">
                    {entry.text}
                  </p>
                ))}
              </div>
            </div>
          </aside>
        </section>
      </div>
    </main>
  )
}

function StatusPill({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-white/16 bg-[#f4f0e4]/10 px-3 text-xs font-black text-[#f4f0e4]/86 sm:text-sm">
      {children}
    </span>
  )
}

function StoneView({ isLast, isWinning, stone }: { isLast: boolean, isWinning: boolean, stone: GomokuStone }) {
  return (
    <span
      className={cn(
        'relative z-10 grid size-[76%] place-items-center rounded-full shadow-[0_8px_16px_rgba(0,0,0,0.32)]',
        stone === 'black'
          ? 'bg-[radial-gradient(circle_at_34%_28%,#4f5a52,#0b0d0c_62%,#020303)]'
          : 'bg-[radial-gradient(circle_at_34%_28%,#ffffff,#d9d4c6_64%,#9f9582)]',
        isWinning && 'ring-4 ring-[#1f8f7b]',
      )}
    >
      {isLast && <span className={cn('size-2 rounded-full', stone === 'black' ? 'bg-[#f4f0e4]' : 'bg-[#101714]')} />}
    </span>
  )
}

function PlayerLine({ active, onRename, onSpeak, player, self, speech }: { active: boolean, onRename: (name: string) => Promise<void>, onSpeak: (text: string) => Promise<void>, player: GomokuPlayer, self: boolean, speech?: GameSpeechEntry }) {
  const { t } = useI18n()
  return (
    <div className={cn('relative grid gap-2 rounded-lg border px-3 py-2', active ? 'border-[#1f8f7b] bg-[#1f8f7b]/18' : 'border-white/14 bg-[#0b1110]/50')}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className={cn('size-5 shrink-0 rounded-full', player.stone === 'black' ? 'bg-[#050706]' : 'bg-[#f4f0e4]')} />
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
              <strong className="block truncate text-sm">{player.name}</strong>
              {self && <PlayerNameEditor buttonClassName="text-[#f4f0e4]" className="min-w-[110px]" name={player.name} onSave={onRename} />}
            </div>
            <span className="text-xs font-bold text-[#f4f0e4]/60">
              {self ? t('gomoku.you') : player.isAI ? t('common.ai') : player.connected ? t('common.online') : t('common.offline')}
            </span>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {self && <SpeechButton palette="gomoku" onSend={onSpeak} />}
          <span className="rounded-full bg-[#f4f0e4]/12 px-2 py-0.5 text-xs font-black">
            {player.stone ? formatStone(player.stone, t) : '-'}
          </span>
        </div>
      </div>
      <SpeechBubble speech={speech} />
    </div>
  )
}

function cellKey(x: number, y: number) {
  return `${x}:${y}`
}

function formatPoint(x: number, y: number) {
  return `${String.fromCharCode(65 + x)}${y + 1}`
}

function formatStone(stone: GomokuStone, t: (key: string) => string) {
  return stone === 'black' ? t('gomoku.blackStone') : t('gomoku.whiteStone')
}
