import type { CSSProperties } from 'react'
import type { XiangqiOnlineMove, XiangqiOnlinePiece, XiangqiOnlinePlayer } from './online'
import type { XiangqiMove, XiangqiPiece, XiangqiPosition, XiangqiSide } from './types'
import type { GameSpeechEntry } from '@/games/speech'
import { ArrowLeft, Copy, Flag, RefreshCw, RotateCw } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { RoomConnectionStatus } from '@/games/RoomConnectionStatus'
import { latestSpeechForPlayer } from '@/games/speech'
import { useAutoFollowScroll } from '@/games/useAutoFollowScroll'
import { usePendingAction } from '@/games/usePendingAction'
import { useI18n } from '@/i18n/context'
import { Button } from '@/shared/components/ui/button'
import { cn } from '@/shared/lib/utils'
import { formatPiece, formatSide, getLegalMoves, getPieceAt, oppositeSide } from './engine'
import { useXiangqiRoom } from './useXiangqiRoom'

const BOARD_FILES = Array.from({ length: 9 }, (_, index) => index)
const BOARD_RANKS = Array.from({ length: 10 }, (_, index) => index)

export function XiangqiPage({ roomId }: { roomId: string }) {
  const { t } = useI18n()
  const { actions, connection, error, isLoading, room } = useXiangqiRoom(roomId)
  const [selectedId, setSelectedId] = useState<string>()
  const [perspective, setPerspective] = useState<XiangqiSide>('red')
  const [message, setMessage] = useState(() => t('xiangqi.tableReady'))
  const pending = usePendingAction()
  const recordScroll = useAutoFollowScroll<HTMLOListElement>()

  const human = useMemo(() => room?.players.find(player => player.id === room.youPlayerId), [room?.players, room?.youPlayerId])
  const currentPlayer = room?.players.find(player => player.id === room.currentPlayerId)
  const winner = room?.players.find(player => player.id === room.winnerId)
  const pieces = useMemo(() => room?.pieces.map(toEnginePiece) ?? [], [room?.pieces])
  const lastMove = room?.moves.at(-1) ? toEngineMove(room.moves.at(-1)!) : undefined
  const selectedPiece = pieces.find(piece => piece.id === selectedId)
  const isHumanTurn = Boolean(room && human && room.phase === 'playing' && room.currentPlayerId === human.id)
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const isAIThinking = Boolean(room && currentPlayer?.isAI && room.phase === 'playing')
  const isMovePending = pending.isPending('move')
  const isRestartPending = pending.isPending('restart')
  const engineState = useMemo(() => ({
    checkSide: room?.checkSide,
    message: '',
    moveHistory: [],
    pieces,
    status: room?.phase === 'finished' ? 'finished' as const : 'playing' as const,
    turn: currentPlayer?.side ?? 'red',
    winner: undefined,
  }), [currentPlayer?.side, pieces, room?.checkSide, room?.phase])
  const legalMoves = useMemo(() => selectedId && isHumanTurn ? getLegalMoves(engineState, selectedId) : [], [engineState, isHumanTurn, selectedId])
  const legalMoveKeys = useMemo(() => new Set(legalMoves.map(positionKey)), [legalMoves])
  const capturedByRed = room?.moves.flatMap(move => move.captured && move.captured.side === 'black' ? [toEnginePiece(move.captured)] : []) ?? []
  const capturedByBlack = room?.moves.flatMap(move => move.captured && move.captured.side === 'red' ? [toEnginePiece(move.captured)] : []) ?? []
  const checkBanner = lastMove?.check && room?.checkSide
    ? {
        id: lastMove.id,
        targetSide: room.checkSide,
        text: room.checkSide === currentPlayer?.side ? `${formatSide(room.checkSide)}${t('xiangqi.checked')}` : `${formatSide(lastMove.piece.side)}${t('xiangqi.check')}`,
      }
    : undefined

  useEffect(() => {
    pending.clearAll()
  }, [pending, pending.clearAll, room?.actionSeq, room?.currentPlayerId, room?.moves.length, room?.phase])

  async function handleCopyRoom() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  function handlePointClick(position: XiangqiPosition) {
    const target = getPieceAt(pieces, position)

    if (!room || room.phase === 'finished' || !isHumanTurn || isAIThinking || isMovePending) {
      return
    }

    if (selectedPiece && legalMoveKeys.has(positionKey(position))) {
      void pending.run('move', () => actions.move(selectedPiece.id, position.x, position.y), { releaseOnSettle: false })
      setMessage(t('xiangqi.movedTo', { position: t('xiangqi.position', { file: position.x + 1, rank: position.y + 1 }) }))
      setSelectedId(undefined)
      return
    }

    if (target && target.side === human?.side) {
      setSelectedId(target.id)
      return
    }

    if (selectedPiece) {
      void pending.run('move', () => actions.move(selectedPiece.id, position.x, position.y), { releaseOnSettle: false })
      setSelectedId(undefined)
    }
  }

  async function handleRestart() {
    await pending.run('restart', () => actions.start(), { releaseOnSettle: false })
    setMessage(t('xiangqi.restarted'))
    setSelectedId(undefined)
    setPerspective('red')
  }

  const displayFiles = perspective === 'red' ? BOARD_FILES : [...BOARD_FILES].reverse()
  const displayRanks = perspective === 'red' ? BOARD_RANKS : [...BOARD_RANKS].reverse()

  if (isLoading || !room || !human) {
    return (
      <main className="grid h-svh place-items-center overflow-hidden bg-[#202018] px-4 text-[#fff8e8]">
        <p className="text-sm font-black">{error ?? t('xiangqi.loading')}</p>
      </main>
    )
  }

  return (
    <main className="h-svh overflow-hidden bg-[#202018] text-[#fff8e8]">
      <div className="mx-auto grid h-full w-[min(1280px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-2 py-2 sm:gap-3 sm:py-3">
        <header className="flex min-h-0 flex-wrap items-end justify-between gap-3">
          <div>
            <p className="mb-0.5 text-xs font-black text-[#f2d59a]/80">ONLINE XIANGQI TABLE</p>
            <h1 className="text-sm font-black leading-none tracking-normal text-[#fff8e8]/92 sm:text-base">{t('xiangqi.title')}</h1>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button asChild className="border-[#fff8e8]/30 bg-[#fff8e8]/10 text-[#fff8e8] hover:bg-[#fff8e8]/18" variant="outline">
              <Link to="/">
                <ArrowLeft className="size-4" />
                {t('common.backToLobby')}
              </Link>
            </Button>
            <Button className="border-[#fff8e8]/30 bg-[#fff8e8]/10 text-[#fff8e8] hover:bg-[#fff8e8]/18" type="button" variant="outline" onClick={() => setPerspective(oppositeSide(perspective))}>
              <RotateCw className="size-4" />
              {t('xiangqi.flip')}
            </Button>
            <Button className="border-[#fff8e8]/30 bg-[#fff8e8]/10 text-[#fff8e8] hover:bg-[#fff8e8]/18" type="button" variant="outline" onClick={handleCopyRoom}>
              <Copy className="size-4" />
              {t('common.copyLink')}
            </Button>
            <RoomConnectionStatus connection={connection} className="self-center" />
            <Button className="bg-[#fff8e8] text-[#202018] hover:bg-[#f3deb3]" disabled={!isHost || isRestartPending} type="button" onClick={handleRestart}>
              <RefreshCw className="size-4" />
              {isRestartPending ? t('common.syncing') : t('xiangqi.restart')}
            </Button>
          </div>
        </header>

        <section className="grid min-h-0 gap-3 overflow-hidden xl:grid-cols-[minmax(0,800px)_340px]">
          <div className="grid min-h-0 place-items-center overflow-hidden rounded-lg border border-[#fff8e8]/18 bg-[radial-gradient(circle_at_20%_12%,rgba(207,48,39,0.18),transparent_30%),linear-gradient(145deg,#31423a,#171612)] p-2 shadow-[0_26px_80px_rgba(0,0,0,0.34)] sm:p-4">
            <XiangqiBoard
              checkBanner={checkBanner}
              displayFiles={displayFiles}
              displayRanks={displayRanks}
              lastMove={lastMove}
              legalMoveKeys={legalMoveKeys}
              pieces={pieces}
              pending={isMovePending}
              selectableSide={isHumanTurn ? human.side : undefined}
              selectedId={selectedId}
              onPointClick={handlePointClick}
            />
          </div>

          <aside className="grid min-h-0 gap-3 overflow-hidden xl:grid-rows-[auto_auto_minmax(0,1fr)]">
            <section className="min-h-0 rounded-lg border border-[#fff8e8]/18 bg-[#10100d]/70 p-3 shadow-[0_18px_46px_rgba(0,0,0,0.24)]">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs font-black text-[#f2d59a]/75">{t('xiangqi.currentTurn')}</p>
                  <h2 className={cn('mt-1 text-2xl font-black', currentPlayer?.side === 'red' ? 'text-[#ff6458]' : 'text-[#fff8e8]')}>
                    {room.phase === 'finished' ? t('xiangqi.winner', { name: winner?.name ?? t('common.player') }) : isAIThinking ? t('xiangqi.aiThinking') : currentPlayer?.name ?? '-'}
                  </h2>
                </div>
                <span className={cn(
                  'grid size-12 place-items-center rounded-full border-2 text-2xl font-black shadow-[0_12px_28px_rgba(0,0,0,0.22)]',
                  currentPlayer?.side === 'red' ? 'border-[#ffb5ae] bg-[#ffe8d8] text-[#c92820]' : 'border-[#fff8e8]/35 bg-[#2b2a24] text-[#fff8e8]',
                )}
                >
                  {room.phase === 'finished' ? <Flag className="size-6" /> : currentPlayer?.side === 'red' ? '帥' : '將'}
                </span>
              </div>
              <p className="mt-3 min-h-11 rounded-lg bg-[#fff8e8]/10 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/86">
                {error ?? (isAIThinking ? t('xiangqi.aiThinkingDetail', { name: currentPlayer?.name ?? t('common.ai') }) : message)}
              </p>
              {room.checkSide && room.phase === 'playing' && (
                <p className="mt-2 rounded-lg bg-[#cf3027]/22 px-3 py-2 text-sm font-black text-[#ffd6c9]">
                  {formatSide(room.checkSide)}
                  {t('xiangqi.checked')}
                </p>
              )}
              <div className="mt-3 grid gap-2">
                {room.players.map(player => (
                  <PlayerLine
                    key={player.id}
                    active={player.id === room.currentPlayerId}
                    player={player}
                    self={player.id === room.youPlayerId}
                    speech={latestSpeechForPlayer(room.speeches, player.id)}
                    onRename={actions.renamePlayer}
                    onSpeak={actions.say}
                  />
                ))}
              </div>
            </section>

            <section className="grid min-h-0 grid-cols-2 gap-3">
              <CapturedPanel pieces={capturedByRed} title={t('xiangqi.redCaptured')} />
              <CapturedPanel pieces={capturedByBlack} title={t('xiangqi.blackCaptured')} />
            </section>

            <section className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] rounded-lg border border-[#fff8e8]/18 bg-[#10100d]/70 p-3 shadow-[0_18px_46px_rgba(0,0,0,0.24)]">
              <div className="mb-2 flex min-h-0 items-center justify-between">
                <h2 className="text-sm font-black text-[#f2d59a]">{t('xiangqi.record')}</h2>
                <span className="text-xs font-bold text-[#fff8e8]/62">
                  {room.moves.length}
                  {t('xiangqi.moves')}
                </span>
              </div>
              <ol ref={recordScroll.containerRef} className="grid min-h-0 content-start gap-2 overflow-y-auto overscroll-contain pr-1" onScroll={recordScroll.handleScroll}>
                {room.moves.length === 0 && (
                  <li className="rounded-lg bg-[#fff8e8]/8 px-3 py-2 text-sm font-bold text-[#fff8e8]/62">{t('xiangqi.opening')}</li>
                )}
                {room.moves.map((onlineMove, index) => {
                  const move = toEngineMove(onlineMove)

                  return (
                    <li key={move.id} className="rounded-lg bg-[#fff8e8]/8 px-3 py-2 text-sm font-bold text-[#fff8e8]/82">
                      <span className="mr-2 text-[#f2d59a]">
                        {index + 1}
                        .
                      </span>
                      {formatSide(move.piece.side)}
                      {formatPiece(move.piece)}
                      {' '}
                      {move.from.x + 1}
                      ,
                      {move.from.y + 1}
                      {' -> '}
                      {move.to.x + 1}
                      ,
                      {move.to.y + 1}
                      {move.captured && (
                        <span className="text-[#ffb5ae]">
                          {' x '}
                          {formatPiece(move.captured)}
                        </span>
                      )}
                      {move.check && <span className="text-[#ffb5ae]">{` ${t('xiangqi.check')}`}</span>}
                    </li>
                  )
                })}
              </ol>
            </section>
          </aside>
        </section>
      </div>
    </main>
  )
}

function XiangqiBoard({
  checkBanner,
  displayFiles,
  displayRanks,
  lastMove,
  legalMoveKeys,
  pieces,
  pending,
  selectableSide,
  selectedId,
  onPointClick,
}: {
  checkBanner?: {
    id: string
    targetSide: XiangqiSide
    text: string
  }
  displayFiles: number[]
  displayRanks: number[]
  lastMove?: XiangqiMove
  legalMoveKeys: Set<string>
  pieces: XiangqiPiece[]
  pending?: boolean
  selectableSide?: XiangqiSide
  selectedId?: string
  onPointClick: (position: XiangqiPosition) => void
}) {
  return (
    <div className="relative aspect-[9/10] h-full max-h-full w-auto max-w-full rounded-lg border-4 border-[#5c321f] bg-[#d8a85e] shadow-[inset_0_0_0_2px_rgba(255,248,232,0.28),inset_0_0_70px_rgba(92,50,31,0.38)]">
      <svg aria-hidden="true" className="absolute inset-[5%] size-[90%] overflow-visible" preserveAspectRatio="none" viewBox="0 0 8 9">
        {BOARD_RANKS.map(rank => <line key={`rank-${rank}`} stroke="#5c321f" strokeWidth="0.035" x1="0" x2="8" y1={rank} y2={rank} />)}
        {[0, 8].map(file => <line key={`edge-${file}`} stroke="#5c321f" strokeWidth="0.035" x1={file} x2={file} y1="0" y2="9" />)}
        {BOARD_FILES.slice(1, 8).map(file => (
          <g key={`file-${file}`}>
            <line stroke="#5c321f" strokeWidth="0.035" x1={file} x2={file} y1="0" y2="4" />
            <line stroke="#5c321f" strokeWidth="0.035" x1={file} x2={file} y1="5" y2="9" />
          </g>
        ))}
        <line stroke="#5c321f" strokeWidth="0.035" x1="3" x2="5" y1="0" y2="2" />
        <line stroke="#5c321f" strokeWidth="0.035" x1="5" x2="3" y1="0" y2="2" />
        <line stroke="#5c321f" strokeWidth="0.035" x1="3" x2="5" y1="7" y2="9" />
        <line stroke="#5c321f" strokeWidth="0.035" x1="5" x2="3" y1="7" y2="9" />
      </svg>

      <div className="pointer-events-none absolute inset-x-[13%] top-1/2 flex -translate-y-1/2 justify-between text-base font-black tracking-[0.28em] text-[#6e3a24]/34 sm:text-2xl">
        <span>楚河</span>
        <span>漢界</span>
      </div>

      {checkBanner && <CheckBanner key={checkBanner.id} banner={checkBanner} />}

      {displayRanks.flatMap((y, rowIndex) => displayFiles.map((x, columnIndex) => {
        const position = { x, y }
        const piece = getPieceAt(pieces, position)
        const key = positionKey(position)
        const isLegal = legalMoveKeys.has(key)

        return (
          <button
            key={key}
            aria-label={`${x + 1},${y + 1}`}
            className={cn(
              'absolute grid size-6 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full transition sm:size-8',
              isLegal && !piece && 'bg-[#1f806d]/34 ring-2 ring-[#1f806d]/70',
              !piece && !isLegal && 'hover:bg-[#5c321f]/14',
            )}
            disabled={pending}
            style={pointStyle(columnIndex, rowIndex)}
            type="button"
            onClick={() => onPointClick(position)}
          >
            {isLegal && !piece ? <span className="size-3 rounded-full bg-[#1f806d]" /> : null}
          </button>
        )
      }))}

      {pieces.map((piece) => {
        const point = displayPointStyle(piece.position, displayFiles, displayRanks)
        const isSelected = piece.id === selectedId
        const isCaptureTarget = legalMoveKeys.has(positionKey(piece.position)) && piece.id !== selectedId
        const canInteract = piece.side === selectableSide || isCaptureTarget

        return (
          <button
            key={piece.id}
            aria-label={`${formatSide(piece.side)}${formatPiece(piece)}`}
            className={cn(
              'xiangqi-piece absolute z-10 grid size-9 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full border-2 text-xl font-black shadow-[0_10px_22px_rgba(52,27,16,0.28)] sm:size-12 sm:text-3xl md:size-14',
              piece.side === 'red' ? 'border-[#d43a30] bg-[#fff1dc] text-[#c92820]' : 'border-[#2d261f] bg-[#fff1dc] text-[#1c1712]',
              canInteract ? 'cursor-pointer' : 'cursor-default',
              isSelected && 'z-20 ring-4 ring-[#1f806d] ring-offset-2 ring-offset-[#d8a85e]',
              isCaptureTarget && 'ring-4 ring-[#cf3027] ring-offset-2 ring-offset-[#d8a85e]',
            )}
            disabled={pending || !canInteract}
            style={point}
            type="button"
            onClick={() => onPointClick(piece.position)}
          >
            {formatPiece(piece)}
          </button>
        )
      })}

      {lastMove?.captured && (
        <CapturedPieceBurst
          key={lastMove.id}
          displayFiles={displayFiles}
          displayRanks={displayRanks}
          move={lastMove}
        />
      )}
    </div>
  )
}

function CheckBanner({
  banner,
}: {
  banner: {
    targetSide: XiangqiSide
    text: string
  }
}) {
  return (
    <div className="pointer-events-none absolute inset-x-4 top-[42%] z-40 grid place-items-center">
      <div className={cn(
        'xiangqi-check-banner rounded-lg border-2 px-5 py-3 text-center text-3xl font-black shadow-[0_24px_58px_rgba(52,27,16,0.34)] sm:px-8 sm:text-5xl',
        banner.targetSide === 'red' ? 'border-[#ffb5ae] bg-[#fff1dc] text-[#c92820]' : 'border-[#2d261f] bg-[#1c1712] text-[#fff8e8]',
      )}
      >
        {banner.text}
      </div>
    </div>
  )
}

function CapturedPieceBurst({
  displayFiles,
  displayRanks,
  move,
}: {
  displayFiles: number[]
  displayRanks: number[]
  move: XiangqiMove
}) {
  if (!move.captured) {
    return null
  }

  return (
    <span
      className={cn(
        'xiangqi-capture-burst pointer-events-none absolute z-30 grid size-9 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full border-2 bg-[#fff1dc] text-xl font-black sm:size-12 sm:text-3xl md:size-14',
        move.captured.side === 'red' ? 'border-[#d43a30] text-[#c92820]' : 'border-[#2d261f] text-[#1c1712]',
      )}
      style={displayPointStyle(move.to, displayFiles, displayRanks)}
    >
      {formatPiece(move.captured)}
    </span>
  )
}

function PlayerLine({ active, onRename, onSpeak, player, self, speech }: { active: boolean, onRename: (name: string) => Promise<void>, onSpeak: (text: string) => Promise<void>, player: XiangqiOnlinePlayer, self: boolean, speech?: GameSpeechEntry }) {
  const { t } = useI18n()
  return (
    <div className={cn('relative grid gap-2 rounded-lg border px-3 py-2', active ? 'border-[#f2d59a] bg-[#f2d59a]/16' : 'border-[#fff8e8]/14 bg-[#10100d]/50')}>
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-2">
            <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
            <strong className="block truncate text-sm">{player.name}</strong>
            {self && <PlayerNameEditor buttonClassName="text-[#fff8e8]" className="min-w-[110px]" name={player.name} onSave={onRename} />}
          </div>
          <span className="text-xs font-bold text-[#fff8e8]/60">
            {self ? t('gomoku.you') : player.isAI ? t('common.ai') : player.connected ? t('common.online') : t('common.offline')}
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {self && <SpeechButton palette="xiangqi" onSend={onSpeak} />}
          <span className={cn('grid size-8 place-items-center rounded-full border bg-[#fff1dc] text-sm font-black', player.side === 'red' ? 'border-[#d43a30] text-[#c92820]' : 'border-[#2d261f] text-[#1c1712]')}>
            {player.side === 'red' ? '帥' : player.side === 'black' ? '將' : '-'}
          </span>
        </div>
      </div>
      <SpeechBubble speech={speech} />
    </div>
  )
}

function CapturedPanel({ pieces, title }: { pieces: XiangqiPiece[], title: string }) {
  const { t } = useI18n()
  return (
    <div className="min-h-24 rounded-lg border border-[#fff8e8]/18 bg-[#10100d]/70 p-3 shadow-[0_18px_46px_rgba(0,0,0,0.24)]">
      <div className="mb-2 text-xs font-black text-[#f2d59a]">{title}</div>
      <div className="flex flex-wrap gap-1.5">
        {pieces.length === 0 && <span className="text-sm font-bold text-[#fff8e8]/52">{t('xiangqi.noCaptured')}</span>}
        {pieces.map(piece => (
          <span
            key={piece.id}
            className={cn(
              'grid size-8 place-items-center rounded-full border bg-[#fff1dc] text-lg font-black',
              piece.side === 'red' ? 'border-[#d43a30] text-[#c92820]' : 'border-[#2d261f] text-[#1c1712]',
            )}
          >
            {formatPiece(piece)}
          </span>
        ))}
      </div>
    </div>
  )
}

function toEnginePiece(piece: XiangqiOnlinePiece): XiangqiPiece {
  return {
    id: piece.id,
    position: { x: piece.x, y: piece.y },
    side: piece.side,
    type: piece.type,
  }
}

function toEngineMove(move: XiangqiOnlineMove): XiangqiMove {
  return {
    captured: move.captured ? toEnginePiece(move.captured) : undefined,
    check: move.check,
    checkmate: move.checkmate,
    from: move.from,
    id: move.id,
    piece: {
      id: move.pieceId,
      position: move.to,
      side: move.side,
      type: move.pieceType,
    },
    to: move.to,
  }
}

function pointStyle(columnIndex: number, rowIndex: number): CSSProperties {
  return {
    left: `${5 + columnIndex * 11.25}%`,
    top: `${5 + rowIndex * 10}%`,
  }
}

function displayPointStyle(position: XiangqiPosition, displayFiles: number[], displayRanks: number[]) {
  return pointStyle(displayFiles.indexOf(position.x), displayRanks.indexOf(position.y))
}

function positionKey(position: XiangqiPosition) {
  return `${position.x}:${position.y}`
}
