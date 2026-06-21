import type { FormEvent, ReactNode } from 'react'
import type { MahjongClaimOption, MahjongOnlinePlayer, MahjongOnlineRoom, MahjongOnlineTile, MahjongOnlineWinResult, MahjongSpeechEntry, MahjongWind } from './online'
import type { AILevel } from '@/games/ai'
import { ArrowLeft, Bot, CircleDot, Copy, DoorOpen, Hand, Plus, RefreshCw, ScrollText, Sparkles, UserMinus } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { getAICapabilities, getAILevelLabel } from '@/games/ai'
import { AILevelBadgeSelect } from '@/games/AILevelBadgeSelect'
import { AILevelPicker } from '@/games/AILevelPicker'
import { ContinueRoomEntry } from '@/games/ContinueRoomEntry'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { useCurrentRoom } from '@/games/useCurrentRoom'
import { usePendingAction } from '@/games/usePendingAction'
import { cn } from '@/shared/lib/utils'
import { createMahjongRoom, getCurrentMahjongRoom } from './online'
import { useMahjongRoom } from './useMahjongRoom'

interface MahjongRoomGateProps {
  roomId?: string
}

export function MahjongRoomGate({ roomId }: MahjongRoomGateProps) {
  const navigate = useNavigate()
  const { actions, error, isLoading, room } = useMahjongRoom(roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState('创建或加入一个麻将房间。')
  const [pendingAI, setPendingAI] = useState(false)
  const [aiLevel, setAILevel] = useState<AILevel>('normal')
  const [llmEnabled, setLLMEnabled] = useState(false)
  const pending = usePendingAction()
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const loadCurrentRoom = useCallback(() => getCurrentMahjongRoom(), [])
  const { currentRoom } = useCurrentRoom(!roomId, loadCurrentRoom)

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
      if (!capabilities.llmEnabled && aiLevel === 'ai') {
        setAILevel('normal')
      }
    })
  }, [aiLevel])

  useEffect(() => {
    pending.clearAll()
  }, [pending, pending.clearAll, room?.actionSeq, room?.phase, room?.players.length])

  if (roomId && room && room.phase !== 'lobby') {
    return <MahjongOnlineTable error={error} room={room} roomId={roomId} />
  }

  async function createRoom() {
    setMessage('正在创建房间...')
    try {
      const nextRoom = await pending.run('create', () => createMahjongRoom())
      if (!nextRoom) {
        return
      }

      navigate(`/games/mahjong?room=${nextRoom.id}`)
      setJoinCode(nextRoom.id)
      setMessage('房间已创建。')
    }
    catch (err) {
      setMessage(err instanceof Error ? err.message : '创建房间失败。')
    }
  }

  function joinRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const normalizedCode = joinCode.trim().toUpperCase()
    if (!normalizedCode) {
      setMessage('请输入房间号。')
      return
    }

    navigate(`/games/mahjong?room=${encodeURIComponent(normalizedCode)}`)
    setMessage('正在进入房间。')
  }

  function copyLink() {
    navigator.clipboard?.writeText(window.location.href)
    setMessage('链接已复制。')
  }

  function enterCurrentRoom() {
    if (!currentRoom) {
      return
    }
    navigate(`/games/mahjong?room=${encodeURIComponent(currentRoom.id)}`)
  }

  async function addAIPlayer() {
    if (pendingAI || pending.isPending('add-ai') || !room) {
      return
    }

    setPendingAI(true)
    setMessage('正在添加 AI...')
    try {
      await pending.run('add-ai', () => actions.addAI(aiLevel), { releaseOnSettle: false })
      setMessage('AI 已加入。')
    }
    finally {
      setPendingAI(false)
    }
  }

  async function startGame() {
    setMessage('正在开始牌局...')
    await pending.run('start', () => actions.start(), { releaseOnSettle: false })
    setMessage('牌局开始。')
  }

  if (!roomId) {
    return (
      <MahjongShell>
        <section className="grid min-h-[min(560px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="rounded-lg border border-[#d8b66a]/35 bg-[#10251f]/80 p-6 shadow-[0_24px_70px_rgba(0,0,0,0.25)]">
            <div className="grid h-full content-end">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">开一桌国标麻将</h2>
              <p className="mt-3 text-sm leading-7 text-[#fff8e8]/75 sm:text-base">
                服务端房间会同步牌桌状态，隐藏非本人手牌，并允许房主用 AI 补足四人。
              </p>
            </div>
          </div>

          <form className="rounded-lg border border-[#d8b66a]/35 bg-[#081914]/82 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.22)]" onSubmit={joinRoom}>
            <h2 className="text-2xl font-black tracking-normal">进入房间</h2>
            <button className="mahjong-action mahjong-action-primary mt-4 w-full" disabled={pending.isPending('create')} type="button" onClick={createRoom}>
              <Plus className="size-4" />
              {pending.isPending('create') ? '同步中...' : '创建并进入'}
            </button>
            {currentRoom && (
              <ContinueRoomEntry
                buttonClassName="mahjong-action w-full"
                className="mt-4 border-[#d8b66a]/25 bg-[#10251f]/62 text-[#fff8e8]"
                room={currentRoom}
                onEnter={enterCurrentRoom}
              />
            )}
            <label className="mt-4 grid gap-2 text-sm font-black" htmlFor="mahjong-room-code">
              房间号
              <input
                id="mahjong-room-code"
                className="min-h-11 rounded-lg border border-[#d8b66a]/35 bg-[#10251f]/80 px-3 uppercase text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#ffd166]"
                placeholder="MJ123"
                value={joinCode}
                onChange={event => setJoinCode(event.target.value)}
              />
            </label>
            <button className="mahjong-action mt-3 w-full" type="submit">
              <DoorOpen className="size-4" />
              加入房间
            </button>
            <p className="mt-3 min-h-6 text-sm font-bold text-[#fff8e8]/72">{message}</p>
          </form>
        </section>
      </MahjongShell>
    )
  }

  return (
    <MahjongShell>
      <section className="grid min-h-[min(600px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="rounded-lg border border-[#d8b66a]/35 bg-[#081914]/82 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.22)]">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black text-[#fff8e8]/60">ROOM</p>
              <h2 className="text-2xl font-black tracking-normal">
                房间
                {' '}
                {roomId}
              </h2>
            </div>
            <div className="flex flex-wrap items-end gap-2">
              <button className="mahjong-action" type="button" onClick={copyLink}>
                <Copy className="size-4" />
                复制链接
              </button>
              <AILevelPicker level={aiLevel} llmEnabled={llmEnabled} palette="dark" onChange={setAILevel} />
              <button
                className={cn('mahjong-action', pendingAI && 'loading')}
                disabled={pendingAI || pending.isPending('add-ai') || !isHost || !room || room.players.length >= 4}
                type="button"
                onClick={addAIPlayer}
              >
                <Bot className="size-4" />
                {pendingAI || pending.isPending('add-ai') ? '添加中' : `添加 AI (${getAILevelLabel(aiLevel, 'zh')})`}
              </button>
              <button className="mahjong-action mahjong-action-primary" disabled={pending.isPending('start') || !isHost || !room || room.players.length < 4} type="button" onClick={startGame}>
                {pending.isPending('start') ? '同步中...' : '开始游戏'}
              </button>
            </div>
          </div>

          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            {isLoading && <p className="text-sm font-bold text-[#fff8e8]/70">正在连接...</p>}
            {room?.players.map(player => (
              <article key={player.id} className="relative rounded-lg border border-[#d8b66a]/25 bg-[#10251f]/72 p-4">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
                    <strong className="truncate text-lg">{player.name}</strong>
                    {player.id === room.youPlayerId && <PlayerNameEditor buttonClassName="text-[#fff8e8]" name={player.name} onSave={actions.renamePlayer} />}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {player.id === room.youPlayerId && <SpeechButton palette="mahjong" onSend={actions.say} />}
                    {player.ai
                      ? (
                          <AILevelBadgeSelect
                            disabled={!isHost || room.phase !== 'lobby'}
                            level={player.ai.level}
                            llmEnabled={llmEnabled}
                            palette="mahjong"
                            onChange={level => void actions.updateAI(player.id, level)}
                          />
                        )
                      : (
                          <span className="rounded-full bg-[#ffd166] px-2 py-0.5 text-xs font-black text-[#143128]">
                            {player.role === 'host' ? '房主' : player.connected ? '在线' : '离线'}
                          </span>
                        )}
                    {isHost && room.phase === 'lobby' && player.role !== 'host' && (
                      <button
                        aria-label="移除玩家"
                        className="mahjong-action min-h-7 px-2"
                        title="移除玩家"
                        type="button"
                        onClick={() => void actions.removePlayer(player.id)}
                      >
                        <UserMinus className="size-4" />
                      </button>
                    )}
                  </div>
                </div>
                <SpeechBubble speech={latestSpeechForPlayer(room.speeches, player.id)} />
                <p className="mt-2 min-h-10 text-sm leading-6 text-[#fff8e8]/72">
                  {player.ai?.personality ?? '准备入座。'}
                </p>
              </article>
            ))}
          </div>

          <p className="mt-4 min-h-6 text-sm font-bold text-[#fff8e8]/72">{error ?? message}</p>
        </div>

        <aside className="rounded-lg border border-[#d8b66a]/35 bg-[#10251f]/80 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.22)]">
          <h2 className="text-xl font-black">房间规则</h2>
          <RuleLine>四人开局，房主可以用 AI 补足座位。</RuleLine>
          <RuleLine>服务端只向本人下发完整手牌，对手仅显示张数、副露和牌河。</RuleLine>
          <RuleLine>首版沿用国标麻将 8 番起胡，后续规则集可继续扩展。</RuleLine>
        </aside>
      </section>
    </MahjongShell>
  )
}

function MahjongOnlineTable({ error, room, roomId }: { error?: string, room: MahjongOnlineRoom, roomId: string }) {
  const { actions } = useMahjongRoom(roomId)
  const pending = usePendingAction()
  const human = room.players.find(player => player.id === room.youPlayerId)
  const currentPlayer = room.players.find(player => player.id === room.currentPlayerId)
  const winner = room.players.find(player => player.id === room.winnerId)

  useEffect(() => {
    pending.clearAll()
  }, [pending, pending.clearAll, room.actionSeq, room.currentPlayerId, room.hasDrawn, room.log.length, room.phase])

  if (!human || !currentPlayer) {
    return (
      <main className="grid min-h-svh place-items-center bg-[#1b342b] px-4 text-[#fff8e8]">
        <p className="text-sm font-black">{error ?? '正在同步麻将房间...'}</p>
      </main>
    )
  }

  const isHumanTurn = room.phase === 'playing' && currentPlayer.id === human.id
  const canHumanDraw = isHumanTurn && !room.hasDrawn
  const canHumanDiscard = isHumanTurn && room.hasDrawn
  const lastLog = room.log.at(-1)?.text
  const isRestartPending = pending.isPending('restart')
  const isDrawPending = pending.isPending('draw')
  const isSelfDrawPending = pending.isPending('self-draw')
  const isSkipClaimsPending = pending.isPending('skip-claims')
  const hasPendingClaim = room.claimOptions.some(option => pending.isPending(`claim:${option.id}`))
  const hasPendingDiscard = human.hand.some(tile => pending.isPending(`discard:${tile.id}`))
  const hasPendingTableAction = isDrawPending || isSelfDrawPending || isSkipClaimsPending || hasPendingClaim || hasPendingDiscard

  return (
    <main className="min-h-svh bg-[#1b342b] text-[#fff8e8]">
      <div className="mx-auto grid min-h-svh w-[min(1380px,calc(100vw-24px))] grid-rows-[auto_auto_minmax(0,1fr)_auto] gap-3 py-3">
        <header className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="mb-0.5 text-xs font-black text-[#ffd166]">ONLINE CHINESE OFFICIAL MAHJONG</p>
            <h1 className="text-sm font-black leading-none tracking-normal text-[#fff8e8]/92 sm:text-base">国标麻将</h1>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Link className="mahjong-action" to="/">
              <ArrowLeft className="size-4" />
              游戏大厅
            </Link>
            <button
              className="mahjong-action mahjong-action-primary"
              disabled={room.hostPlayerId !== room.youPlayerId || isRestartPending}
              type="button"
              onClick={() => void pending.run('restart', () => actions.start(), { releaseOnSettle: false })}
            >
              <RefreshCw className="size-4" />
              {isRestartPending ? '同步中...' : '重开'}
            </button>
          </div>
        </header>

        <section className="flex min-h-0 flex-wrap items-center gap-2 rounded-lg border border-[#d8b66a]/35 bg-[#10251f]/80 p-2 shadow-[0_18px_46px_rgba(0,0,0,0.22)]">
          <StatusPill icon={<ScrollText className="size-4" />}>{room.ruleset.name}</StatusPill>
          <StatusPill>
            {room.ruleset.minFan}
            番起胡
          </StatusPill>
          <StatusPill>
            圈风：
            {formatWind(room.roundWind)}
          </StatusPill>
          <StatusPill>
            牌墙：
            {room.wallCount}
          </StatusPill>
          <StatusPill>
            回合：
            {currentPlayer.name}
          </StatusPill>
          <StatusPill>
            房间：
            {room.id}
          </StatusPill>
        </section>

        <section className="grid min-h-[560px] gap-3 overflow-hidden rounded-lg border border-[#d8b66a]/45 bg-[radial-gradient(circle_at_center,rgba(255,248,232,0.12),transparent_42%),linear-gradient(135deg,#173b31,#10251f)] p-3 shadow-[inset_0_0_80px_rgba(0,0,0,0.28),0_24px_70px_rgba(0,0,0,0.25)] lg:grid-cols-[240px_minmax(0,1fr)_240px] lg:grid-rows-[150px_minmax(0,1fr)_180px]">
          <PlayerPanel className="lg:col-start-2 lg:row-start-1" currentPlayerId={currentPlayer.id} player={room.players[2]} speech={latestSpeechForPlayer(room.speeches, room.players[2]?.id)} />
          <PlayerPanel className="lg:col-start-1 lg:row-start-2" currentPlayerId={currentPlayer.id} player={room.players[3]} speech={latestSpeechForPlayer(room.speeches, room.players[3]?.id)} />

          <div className="relative grid min-h-[270px] place-items-center rounded-lg border border-[#d8b66a]/30 bg-[#0f211c]/70 p-3 lg:col-start-2 lg:row-start-2">
            <div className="absolute inset-4 rounded-lg border border-[#d8b66a]/25" />
            <div className="z-10 grid w-full max-w-xl gap-3 text-center">
              <div className="mx-auto grid size-24 place-items-center rounded-full border-4 border-[#d8b66a] bg-[#fff8e8] text-[#143128] shadow-[0_18px_50px_rgba(0,0,0,0.26)]">
                <CircleDot className="size-10" strokeWidth={2.8} />
              </div>
              <div>
                <strong className="text-xl">{room.phase === 'finished' ? (winner ? `${winner.name} 和牌` : '流局') : `${currentPlayer.name} 行动中`}</strong>
                <p className="mt-1 text-sm font-bold text-[#fff8e8]/72">{error ?? lastLog ?? room.ruleset.description}</p>
              </div>
              <ScorePanel result={room.winResult} />
            </div>
          </div>

          <PlayerPanel className="lg:col-start-3 lg:row-start-2" currentPlayerId={currentPlayer.id} player={room.players[1]} speech={latestSpeechForPlayer(room.speeches, room.players[1]?.id)} />

          <section className="relative grid min-h-0 gap-3 rounded-lg border border-[#d8b66a]/35 bg-[#081914]/82 p-3 lg:col-span-3 lg:row-start-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <div className="flex min-w-0 flex-wrap items-center gap-2">
                  <strong className="text-lg">你的手牌</strong>
                  <PlayerNameEditor buttonClassName="text-[#fff8e8]" className="min-w-[120px]" name={human.name} onSave={actions.renamePlayer} />
                </div>
                <p className="text-sm font-bold text-[#fff8e8]/70">
                  {room.phase === 'claiming' ? '可以声明吃、碰或胡。' : canHumanDiscard ? '选择一张牌打出。' : canHumanDraw ? '轮到你摸牌。' : `等待 ${currentPlayer.name}。`}
                </p>
                <SpeechBubble speech={latestSpeechForPlayer(room.speeches, human.id)} />
              </div>
              <div className="flex flex-wrap gap-2">
                <SpeechButton palette="mahjong" onSend={actions.say} />
                <button
                  className="mahjong-action"
                  disabled={!canHumanDraw || isDrawPending || hasPendingTableAction}
                  type="button"
                  onClick={() => void pending.run('draw', actions.draw, { releaseOnSettle: false })}
                >
                  <Hand className="size-4" />
                  {isDrawPending ? '同步中...' : '摸牌'}
                </button>
                <button
                  className="mahjong-action mahjong-action-primary"
                  disabled={!canHumanDiscard || isSelfDrawPending || hasPendingTableAction}
                  type="button"
                  onClick={() => void pending.run('self-draw', actions.selfDraw, { releaseOnSettle: false })}
                >
                  <Sparkles className="size-4" />
                  {isSelfDrawPending ? '同步中...' : '自摸'}
                </button>
              </div>
            </div>

            {room.claimOptions.length > 0 && (
              <div className="flex flex-wrap items-center gap-2 rounded-lg bg-[#fff8e8]/10 p-2">
                {room.claimOptions.map(option => (
                  <button
                    key={option.id}
                    className="mahjong-action mahjong-action-primary"
                    disabled={hasPendingTableAction}
                    type="button"
                    onClick={() => void pending.run(`claim:${option.id}`, () => actions.claim(option.id), { releaseOnSettle: false })}
                  >
                    {pending.isPending(`claim:${option.id}`) ? '同步中...' : claimLabel(option)}
                  </button>
                ))}
                <button
                  className="mahjong-action"
                  disabled={hasPendingTableAction}
                  type="button"
                  onClick={() => void pending.run('skip-claims', actions.skipClaims, { releaseOnSettle: false })}
                >
                  {isSkipClaimsPending ? '同步中...' : '跳过'}
                </button>
              </div>
            )}

            <div className="flex min-h-24 flex-wrap items-end gap-1.5 overflow-y-auto pb-1">
              {human.hand.map(tile => (
                <button
                  key={tile.id}
                  className="p-0 transition hover:-translate-y-1 disabled:cursor-not-allowed disabled:opacity-55"
                  disabled={!canHumanDiscard || hasPendingTableAction}
                  title={canHumanDiscard ? `打出 ${formatTile(tile)}` : formatTile(tile)}
                  type="button"
                  onClick={() => void pending.run(`discard:${tile.id}`, () => actions.discard(tile.id), { releaseOnSettle: false })}
                >
                  <TileView tile={tile} />
                </button>
              ))}
            </div>
          </section>
        </section>

        <p className="pb-1 text-xs font-bold text-[#fff8e8]/65">{room.ruleset.description}</p>
      </div>
    </main>
  )
}

function MahjongShell({ children }: { children: ReactNode }) {
  return (
    <main className="min-h-svh overflow-y-auto bg-[#1b342b] text-[#fff8e8]">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
        <header className="flex items-end justify-between gap-4">
          <div>
            <p className="mb-0.5 text-xs font-black tracking-normal text-[#ffd166]">ONLINE MAHJONG TABLE</p>
            <h1 className="text-sm font-black leading-none tracking-normal text-[#fff8e8]/92 sm:text-base">国标麻将</h1>
          </div>
          <Link className="inline-grid min-h-10 place-items-center rounded-full border border-[#d8b66a]/45 bg-[#081914]/55 px-4 text-sm font-bold text-[#fff8e8] transition hover:bg-[#081914]/72" to="/">
            <ArrowLeft className="mr-2 inline size-4" />
            游戏大厅
          </Link>
        </header>
        {children}
      </div>
    </main>
  )
}

function PlayerPanel({ className, currentPlayerId, player, speech }: { className?: string, currentPlayerId: string, player?: MahjongOnlinePlayer, speech?: MahjongSpeechEntry }) {
  if (!player) {
    return <article className={cn('rounded-lg border border-[#d8b66a]/35 bg-[#081914]/40 p-3', className)} />
  }
  const isCurrent = currentPlayerId === player.id

  return (
    <article className={cn('relative grid min-h-32 content-between rounded-lg border border-[#d8b66a]/35 bg-[#081914]/72 p-3 shadow-[0_18px_40px_rgba(0,0,0,0.18)]', isCurrent && 'ring-2 ring-[#ffd166]', className)}>
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className="flex items-center gap-2">
            <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
            <strong>{player.name}</strong>
            <span className="rounded-full bg-[#ffd166] px-2 py-0.5 text-xs font-black text-[#143128]">
              {formatWind(player.wind)}
              风
            </span>
          </div>
          <p className="mt-1 text-xs font-bold text-[#fff8e8]/65">
            手牌
            {' '}
            {player.handCount}
            {' '}
            张
          </p>
        </div>
        {player.hand.length > 0 ? <span className="text-xs font-black text-[#ffd166]">SELF</span> : <OpponentTiles count={player.handCount} />}
      </div>
      <SpeechBubble speech={speech} />
      <div className="mt-2 flex min-h-10 flex-wrap content-end gap-1">
        {player.melds.map(meld => (
          <span key={meld.id} className="rounded-md bg-[#fff8e8]/12 px-2 py-1 text-xs font-black text-[#fff8e8]">
            {meld.tiles.map(formatTile).join(' ')}
          </span>
        ))}
        {player.discards.slice(-10).map(tile => (
          <MiniTile key={tile.id} tile={tile} />
        ))}
      </div>
    </article>
  )
}

function StatusPill({ children, icon }: { children: ReactNode, icon?: ReactNode }) {
  return (
    <span className="inline-flex min-h-9 items-center gap-2 rounded-lg border border-[#d8b66a]/35 bg-[#fff8e8]/10 px-3 text-sm font-black">
      {icon}
      {children}
    </span>
  )
}

function RuleLine({ children }: { children: ReactNode }) {
  return <p className="mt-3 rounded-lg bg-[#081914]/58 px-3 py-2 text-sm leading-6 text-[#fff8e8]/78">{children}</p>
}

function TileView({ tile }: { tile: MahjongOnlineTile }) {
  return (
    <span className="mahjong-tile">
      <span className={cn('text-[11px] font-black', tileColor(tile))}>{tileSuitLabel(tile)}</span>
      <strong className={cn('text-2xl leading-none', tileColor(tile))}>{tileMainLabel(tile)}</strong>
    </span>
  )
}

function MiniTile({ tile }: { tile: MahjongOnlineTile }) {
  return (
    <span className="grid h-8 w-6 place-items-center rounded bg-[#fff8e8] text-[11px] font-black text-[#143128] shadow">
      {formatTile(tile)}
    </span>
  )
}

function OpponentTiles({ count }: { count: number }) {
  return (
    <div className="flex min-w-16 justify-end">
      {Array.from({ length: Math.min(count, 8) }, (_, index) => (
        <span key={`${count}-${index}`} className="h-8 w-4 rounded bg-[#d8b66a] shadow" style={{ marginLeft: index === 0 ? 0 : -8 }} />
      ))}
    </div>
  )
}

function ScorePanel({ result }: { result?: MahjongOnlineWinResult }) {
  const patterns = result?.patterns ?? []

  if (!result || patterns.length === 0) {
    return <div className="mx-auto rounded-lg bg-[#081914]/70 px-3 py-2 text-sm font-bold text-[#fff8e8]/70">暂无可胡番型</div>
  }

  return (
    <div className="mx-auto max-w-full rounded-lg bg-[#081914]/70 px-3 py-2 text-sm font-bold text-[#fff8e8]">
      <span className={result.canWin ? 'text-[#ffd166]' : 'text-[#fff8e8]/70'}>
        {result.fan}
        {' '}
        番
      </span>
      <span className="mx-2 text-[#fff8e8]/35">|</span>
      {patterns.map(pattern => `${pattern.name}${pattern.fan}`).join(' / ')}
    </div>
  )
}

function claimLabel(option: MahjongClaimOption) {
  if (option.kind === 'hu') {
    return `胡 ${option.winResult?.fan ?? 0}番`
  }
  if (option.kind === 'peng') {
    return `碰 ${formatTile(option.tile)}`
  }

  return `吃 ${[...option.tilesFromHand, option.tile].sort((left, right) => left.code.localeCompare(right.code)).map(formatTile).join('')}`
}

function formatTile(tile: MahjongOnlineTile) {
  if (tile.kind === 'characters') {
    return `${tile.rank}万`
  }
  if (tile.kind === 'dots') {
    return `${tile.rank}筒`
  }
  if (tile.kind === 'bamboo') {
    return `${tile.rank}条`
  }
  if (tile.kind === 'wind') {
    return formatWind(tile.wind ?? 'east')
  }

  const labels = {
    green: '发',
    red: '中',
    white: '白',
  }

  return labels[tile.dragon ?? 'white']
}

function formatWind(wind: MahjongWind) {
  const labels: Record<MahjongWind, string> = {
    east: '东',
    north: '北',
    south: '南',
    west: '西',
  }

  return labels[wind]
}

function tileMainLabel(tile: MahjongOnlineTile) {
  if (tile.rank) {
    return tile.rank
  }

  return formatTile(tile)
}

function tileSuitLabel(tile: MahjongOnlineTile) {
  if (tile.kind === 'characters') {
    return '万'
  }
  if (tile.kind === 'dots') {
    return '筒'
  }
  if (tile.kind === 'bamboo') {
    return '条'
  }

  return '字'
}

function tileColor(tile: MahjongOnlineTile) {
  if (tile.kind === 'characters' || tile.dragon === 'red') {
    return 'text-[#b63b2f]'
  }
  if (tile.kind === 'bamboo' || tile.dragon === 'green') {
    return 'text-[#177554]'
  }

  return 'text-[#173047]'
}

function latestSpeechForPlayer(speeches: MahjongSpeechEntry[], playerId: string | undefined) {
  if (!playerId) {
    return undefined
  }

  return speeches.findLast(speech => speech.playerId === playerId)
}
