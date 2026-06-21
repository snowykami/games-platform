import type { FormEvent, ReactNode } from 'react'
import type { AIDebugTrace, SocialGameSlug, SocialRoom } from './online'
import { Bot, ClipboardList, Copy, DoorOpen, Eye, EyeOff, Plus, ScrollText, ShieldQuestion, Sparkles, X } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { getAICapabilities } from '@/games/ai'
import { ContinueRoomEntry } from '@/games/ContinueRoomEntry'
import { useAutoFollowScroll } from '@/games/useAutoFollowScroll'
import { useCurrentRoom } from '@/games/useCurrentRoom'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { createSocialRoom, getCurrentSocialRoom } from './online'
import { ActionPanel } from './socialActionPanel'
import { SelfIntel } from './socialIntel'
import { UndercoverLobbyConfig, WerewolfRoleSetup } from './socialLobbyConfig'
import { PlayerGrid, TableLogLine } from './socialPlayers'
import { roleTotal, socialButton } from './socialStyle'
import { GAME_COPY } from './socialTheme'
import { RoleList, SocialShell, StatusPill } from './socialUi'
import { useSocialRoom } from './useSocialRoom'

interface SocialDeductionRoomGateProps {
  game: SocialGameSlug
  roomId?: string
}

export function SocialDeductionRoomGate({ game, roomId }: SocialDeductionRoomGateProps) {
  const config = GAME_COPY[game]
  const navigate = useNavigate()
  const { t, ta } = useI18n()
  const [godView, setGodView] = useState(false)
  const { actions, error, isLoading, room } = useSocialRoom(game, roomId, godView)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('room.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [llmEnabled, setLLMEnabled] = useState(false)
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const loadCurrentRoom = useCallback(() => getCurrentSocialRoom(game), [game])
  const { currentRoom } = useCurrentRoom(!roomId, loadCurrentRoom)

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
    })
  }, [])

  async function createRoom() {
    setMessage(t('room.defaultMessage'))
    try {
      const nextRoom = await createSocialRoom(game)
      navigate(`/games/${game}?room=${nextRoom.id}`)
      setJoinCode(nextRoom.id)
      setMessage(t('room.created'))
    }
    catch (err) {
      setMessage(err instanceof Error ? err.message : t('room.createFailed'))
    }
  }

  function joinRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const normalizedCode = joinCode.trim().toUpperCase()
    if (!normalizedCode) {
      setMessage(t('room.enterCode'))
      return
    }
    navigate(`/games/${game}?room=${encodeURIComponent(normalizedCode)}`)
    setMessage(t('room.entered'))
  }

  function copyLink() {
    navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  function enterCurrentRoom() {
    if (!currentRoom) {
      return
    }
    navigate(`/games/${game}?room=${encodeURIComponent(currentRoom.id)}`)
  }

  async function addAIPlayer() {
    if (pendingAI || !room) {
      return
    }
    setPendingAI(true)
    setMessage(t('room.addingAI'))
    try {
      await actions.addAI('ai')
      setMessage(t('room.aiJoined'))
    }
    finally {
      setPendingAI(false)
    }
  }

  async function startGame() {
    setMessage(t('room.starting'))
    await actions.start()
    setMessage(t('room.gameStarted'))
  }

  if (!roomId) {
    return (
      <SocialShell config={config} game={game}>
        <section className="grid min-h-[min(600px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className={cn('relative grid content-end overflow-hidden rounded-lg border p-5 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-6', config.panel)}>
            <img alt="" className="absolute inset-0 size-full object-cover opacity-48" src={config.cover} />
            <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(0,0,0,0.05),rgba(0,0,0,0.74))]" />
            <div className="relative max-w-xl">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">{t(config.lobbyTitleKey)}</h2>
              <p className="mt-3 text-sm leading-7 text-[#fff8e8]/78 sm:text-base">{t(config.lobbyDescriptionKey)}</p>
            </div>
          </div>

          <form className={cn('grid content-start gap-4 rounded-lg border p-5 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-6', config.panel)} onSubmit={joinRoom}>
            <h2 className="text-2xl font-black tracking-normal">{t(config.enterTitleKey)}</h2>
            <button className={socialButton(config, true)} type="button" onClick={createRoom}>
              <Plus className="size-4" />
              {t('common.createAndEnter')}
            </button>
            {currentRoom && (
              <ContinueRoomEntry
                buttonClassName={socialButton(config)}
                className="border-white/18 bg-black/20 text-[#fff8e8]"
                room={currentRoom}
                onEnter={enterCurrentRoom}
              />
            )}
            <label className="grid gap-2 text-sm font-black" htmlFor={`${game}-room-code`}>
              {t('common.roomCode')}
              <input
                id={`${game}-room-code`}
                className="min-h-11 rounded-lg border border-white/24 bg-black/28 px-3 uppercase text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#fff8e8]"
                placeholder={config.roomPrefix}
                value={joinCode}
                onChange={event => setJoinCode(event.target.value)}
              />
            </label>
            <button className={socialButton(config)} type="submit">
              <DoorOpen className="size-4" />
              {t('common.joinRoom')}
            </button>
            <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{message}</p>
          </form>
        </section>
      </SocialShell>
    )
  }

  if (isLoading || !room) {
    return (
      <SocialShell config={config} game={game}>
        <div className={cn('grid min-h-[420px] place-items-center rounded-lg border p-6', config.panel)}>
          <p className="text-sm font-black">{error ?? t('room.connecting')}</p>
        </div>
      </SocialShell>
    )
  }

  if (room.phase !== 'lobby') {
    return <SocialGamePage actions={actions} config={config} error={error} game={game} godView={godView} room={room} setGodView={setGodView} />
  }

  return (
    <SocialShell config={config} game={game} phase={room.phase}>
      <section className="grid min-h-[min(640px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className={cn('grid grid-rows-[auto_minmax(0,1fr)_auto] gap-4 rounded-lg border p-4 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-5', config.panel)}>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black text-[#fff8e8]/65">ROOM</p>
              <h2 className="text-2xl font-black tracking-normal">
                {t('common.room')}
                {' '}
                {roomId}
              </h2>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className={socialButton(config)} type="button" onClick={copyLink}>
                <Copy className="size-4" />
                {t('common.copyLink')}
              </button>
              <div className="flex flex-wrap items-center gap-2">
                <span className="inline-flex min-h-10 items-center rounded-lg border border-white/16 bg-white/10 px-3 text-xs font-black text-[#fff8e8]/86">
                  {llmEnabled ? t('common.ai') : t('social.llmRequired')}
                </span>
                <button
                  className={cn(socialButton(config), pendingAI && 'opacity-70')}
                  disabled={pendingAI || !isHost || !llmEnabled || room.players.length >= room.maxPlayers}
                  type="button"
                  onClick={addAIPlayer}
                >
                  <Bot className="size-4" />
                  {pendingAI ? t('room.addingAI') : t('room.addAI')}
                </button>
                <button className={socialButton(config, true)} disabled={!isHost || room.players.length < room.minPlayers} type="button" onClick={startGame}>
                  {t('common.startGame')}
                </button>
              </div>
            </div>
          </div>

          <PlayerGrid actions={actions} config={config} isHost={isHost} room={room} />
          {game === 'undercover' && <UndercoverLobbyConfig actions={actions} config={config} isHost={isHost} room={room} />}

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <p className="text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
          </div>
        </div>

        <details className={cn('rounded-lg border p-4 shadow-[0_22px_64px_rgba(0,0,0,0.28)] lg:hidden', config.panel)}>
          <summary className="flex min-h-10 cursor-pointer list-none items-center gap-2 text-lg font-black">
            <Sparkles className="size-5" />
            {t(config.rulesTitleKey)}
          </summary>
          <LobbyRulesContent actions={actions} config={config} game={game} isHost={isHost} room={room} rules={ta(config.rulesKey)} />
        </details>

        <aside className={cn('hidden content-start gap-3 rounded-lg border p-5 shadow-[0_22px_64px_rgba(0,0,0,0.28)] lg:grid', config.panel)}>
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            {t(config.rulesTitleKey)}
          </h2>
          <LobbyRulesContent actions={actions} config={config} game={game} isHost={isHost} room={room} rules={ta(config.rulesKey)} />
        </aside>
      </section>
    </SocialShell>
  )
}

function LobbyRulesContent({
  actions,
  config,
  game,
  isHost,
  room,
  rules,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  isHost: boolean
  room: SocialRoom
  rules: string[]
}) {
  return (
    <div className="mt-3 grid content-start gap-3 lg:mt-0">
      {game === 'werewolf' && (
        <WerewolfRoleSetup
          key={`${room.players.length}:${room.werewolf.roleConfig.mode}:${room.werewolf.roleConfig.presetId}:${roleTotal(room.werewolf.roleConfig.counts)}`}
          actions={actions}
          config={config}
          isHost={isHost}
          room={room}
        />
      )}
      <RoleList game={game} />
      {rules.map(rule => (
        <p key={rule} className="rounded-lg bg-black/24 px-3 py-2 text-sm leading-6 text-[#fff8e8]/78">{rule}</p>
      ))}
    </div>
  )
}

function SocialGamePage({
  actions,
  config,
  error,
  game,
  godView,
  room,
  setGodView,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  error?: string
  game: SocialGameSlug
  godView: boolean
  room: SocialRoom
  setGodView: (enabled: boolean) => void
}) {
  const { t } = useI18n()
  const [message, setMessage] = useState(() => t('social.connected'))
  const [mobilePanel, setMobilePanel] = useState<SocialMobilePanel>(null)
  const tableLogScroll = useAutoFollowScroll<HTMLDivElement>()
  const you = room.players.find(player => player.id === room.youPlayerId)
  const leader = room.players.find(player => player.id === room.avalon.leaderId)
  const isHost = Boolean(room.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const renderActionPanel = () => (
    <ActionPanel
      key={`${room.phase}:${room.avalon.round}:${room.werewolf.day}`}
      actions={actions}
      config={config}
      game={game}
      room={room}
      setMessage={setMessage}
      you={you}
    />
  )
  const renderTableLog = (withAutoScroll: boolean) => (
    <div
      ref={withAutoScroll ? tableLogScroll.containerRef : undefined}
      className="grid min-h-0 content-start gap-2 overflow-auto pr-1"
      onScroll={withAutoScroll ? tableLogScroll.handleScroll : undefined}
    >
      {room.log.map(entry => (
        <TableLogLine key={entry.id} entry={entry} room={room} />
      ))}
    </div>
  )

  async function copyLink() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  async function restart() {
    await actions.start()
    setMessage(t('social.restarted'))
  }

  return (
    <SocialShell config={config} fixedViewport game={game} phase={room.phase}>
      <section className="relative grid h-full min-h-0 gap-3 overflow-hidden pb-[72px] lg:grid-cols-[minmax(0,1fr)_340px] lg:pb-0">
        <div className={cn('grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 overflow-hidden rounded-lg border p-3 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-4', config.panel)}>
          <div className="flex items-center gap-2 overflow-x-auto rounded-lg border border-white/14 bg-black/24 p-2">
            <button className={cn(socialButton(config, true), 'shrink-0')} type="button" onClick={copyLink}>
              <Copy className="size-4" />
              <span className="hidden sm:inline">{t('common.copyLink')}</span>
            </button>
            <StatusPill className="shrink-0">{t(`social.phases.${room.phase}`)}</StatusPill>
            <StatusPill className="shrink-0">
              {game === 'werewolf' ? t('werewolf.dayCount', { day: room.werewolf.day || 1 }) : t('avalon.roundCount', { round: room.avalon.round || 1 })}
            </StatusPill>
            {game === 'avalon' && <StatusPill className="shrink-0">{t('avalon.leader', { name: leader?.name ?? '-' })}</StatusPill>}
            <StatusPill className="hidden shrink-0 sm:inline-flex">
              {t('common.room')}
              {' '}
              {room.id}
            </StatusPill>
            {room.godViewAvailable && (
              <button
                className={cn(socialButton(config, godView), 'shrink-0', godView && 'ring-2 ring-[#fff8e8]/70')}
                title="仅平台管理员可用"
                type="button"
                onClick={() => setGodView(!godView)}
              >
                {godView ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
                上帝视角
              </button>
            )}
            <button className={cn(socialButton(config), 'shrink-0 lg:ml-auto')} disabled={!isHost} type="button" onClick={restart}>
              {t('social.restart')}
            </button>
          </div>

          <div className="grid min-h-0 gap-3 overflow-hidden lg:grid-cols-[minmax(0,1fr)_260px]">
            <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-3 overflow-hidden">
              <SelfIntel className="hidden md:block" config={config} game={game} room={room} you={you} />
              <PlayerGrid actions={actions} className="min-h-0" config={config} isHost={false} room={room} compact />
            </div>

            <div className="hidden min-h-0 overflow-auto pr-1 lg:block">
              {renderActionPanel()}
            </div>
          </div>

          <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
        </div>

        <aside className={cn('hidden min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 overflow-hidden rounded-lg border p-4 shadow-[0_22px_64px_rgba(0,0,0,0.28)] lg:grid', config.panel)}>
          <h2 className="text-xl font-black">{t('social.tableLog')}</h2>
          {renderTableLog(true)}
          <AIDebugPanel room={room} />
          {room.phase === 'finished' && (
            <div className="rounded-lg bg-[#fff8e8] p-3 text-[#171411]">
              <p className="text-xs font-black">{t('social.result')}</p>
              <strong>{room.winnerMessage}</strong>
            </div>
          )}
        </aside>

        <SocialMobileDock config={config} openPanel={setMobilePanel} />
        <SocialMobileDrawer config={config} panel={mobilePanel} room={room} onClose={() => setMobilePanel(null)}>
          {mobilePanel === 'intel' && <SelfIntel config={config} game={game} room={room} you={you} />}
          {mobilePanel === 'action' && renderActionPanel()}
          {mobilePanel === 'log' && (
            <div className="grid min-h-[55svh] grid-rows-[minmax(0,1fr)_auto] gap-3">
              {renderTableLog(false)}
              <AIDebugPanel room={room} />
              {room.phase === 'finished' && (
                <div className="rounded-lg bg-[#fff8e8] p-3 text-[#171411]">
                  <p className="text-xs font-black">{t('social.result')}</p>
                  <strong>{room.winnerMessage}</strong>
                </div>
              )}
            </div>
          )}
        </SocialMobileDrawer>
      </section>
    </SocialShell>
  )
}

function AIDebugPanel({ room }: { room: SocialRoom }) {
  const [selectedPlayerId, setSelectedPlayerId] = useState('')
  const traces = room.godViewEnabled ? room.aiDebugTraces ?? [] : []
  if (traces.length === 0) {
    return null
  }
  const playerOptions = aiDebugPlayerOptions(room, traces)
  const activePlayerId = playerOptions.some(option => option.id === selectedPlayerId) ? selectedPlayerId : ''
  const visibleTraces = activePlayerId ? traces.filter(trace => trace.playerId === activePlayerId) : traces

  return (
    <section className="grid max-h-72 min-h-0 grid-rows-[auto_auto_minmax(0,1fr)] gap-2 overflow-hidden rounded-lg border border-cyan-200/18 bg-cyan-950/18 p-3">
      <div className="flex items-center justify-between gap-2">
        <h3 className="flex items-center gap-2 text-sm font-black text-cyan-100">
          <Bot className="size-4" />
          AI 调试
        </h3>
        <span className="rounded-full bg-cyan-100/12 px-2 py-1 text-[0.68rem] font-black text-cyan-100/80">
          {visibleTraces.length}
          /
          {traces.length}
        </span>
      </div>
      <div className="flex min-h-8 gap-1.5 overflow-x-auto pb-1">
        <AIDebugPlayerFilterButton
          active={activePlayerId === ''}
          count={traces.length}
          label="全部"
          onClick={() => setSelectedPlayerId('')}
        />
        {playerOptions.map(option => (
          <AIDebugPlayerFilterButton
            key={option.id}
            active={activePlayerId === option.id}
            count={option.count}
            label={option.label}
            onClick={() => setSelectedPlayerId(option.id)}
          />
        ))}
      </div>
      <div className="grid min-h-0 content-start gap-2 overflow-y-auto pr-1">
        {[...visibleTraces].reverse().map(trace => <AIDebugTraceCard key={trace.id} trace={trace} />)}
      </div>
    </section>
  )
}

function AIDebugPlayerFilterButton({
  active,
  count,
  label,
  onClick,
}: {
  active: boolean
  count: number
  label: string
  onClick: () => void
}) {
  return (
    <button
      className={cn(
        'inline-flex min-h-7 shrink-0 items-center gap-1.5 rounded-md border px-2 text-[0.68rem] font-black transition',
        active ? 'border-cyan-100/70 bg-cyan-100 text-cyan-950' : 'border-white/10 bg-white/8 text-[#fff8e8]/72 hover:bg-white/12',
      )}
      type="button"
      onClick={onClick}
    >
      <span>{label}</span>
      <span className={cn('rounded-full px-1.5 py-0.5 text-[0.62rem]', active ? 'bg-cyan-950/12' : 'bg-black/26')}>
        {count}
      </span>
    </button>
  )
}

function AIDebugTraceCard({ trace }: { trace: AIDebugTrace }) {
  const thinking = trace.thinking?.trim()
  return (
    <article className="grid gap-2 rounded-lg bg-black/28 p-2 text-xs text-[#fff8e8]/78">
      <div className="flex flex-wrap items-center gap-2 font-black text-[#fff8e8]">
        <span>{trace.playerName ?? trace.playerId ?? 'AI'}</span>
        <span className="rounded-md bg-white/10 px-1.5 py-1 text-[0.65rem] uppercase text-[#fff8e8]/70">{trace.phase}</span>
        <span className="rounded-md bg-white/10 px-1.5 py-1 text-[0.65rem] uppercase text-[#fff8e8]/70">{trace.scope}</span>
        <span className="ml-auto text-[0.65rem] text-[#fff8e8]/56">
          {trace.durationMs}
          ms
        </span>
      </div>
      {trace.actionId && (
        <p>
          <strong>Action:</strong>
          {' '}
          {trace.actionId}
        </p>
      )}
      {trace.reason && (
        <p>
          <strong>Reason:</strong>
          {' '}
          {trace.reason}
        </p>
      )}
      {trace.speech && (
        <p>
          <strong>Speech:</strong>
          {' '}
          {trace.speech}
        </p>
      )}
      {trace.error && (
        <p className="text-rose-200">
          <strong>Error:</strong>
          {' '}
          {trace.error}
        </p>
      )}
      <details className="rounded-md border border-white/10 bg-black/20 p-2">
        <summary className="cursor-pointer text-[0.7rem] font-black text-cyan-100">
          Thinking
          {!trace.thinkingAvailable && <span className="ml-2 text-[#fff8e8]/48">未返回</span>}
        </summary>
        <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-[0.68rem] leading-5 text-[#fff8e8]/72">{thinking || '当前模型未返回 thinking。'}</pre>
      </details>
      {trace.actions && trace.actions.length > 0 && (
        <details className="rounded-md border border-white/10 bg-black/20 p-2">
          <summary className="cursor-pointer text-[0.7rem] font-black text-[#fff8e8]/80">合法动作</summary>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {trace.actions.map(action => (
              <span key={action.id} className="rounded-md bg-white/10 px-1.5 py-1 text-[0.65rem] font-bold text-[#fff8e8]/72" title={action.description}>
                {action.label || action.id}
              </span>
            ))}
          </div>
        </details>
      )}
    </article>
  )
}

function aiDebugPlayerOptions(room: SocialRoom, traces: AIDebugTrace[]) {
  const counts = new Map<string, number>()
  const names = new Map<string, string>()
  for (const trace of traces) {
    if (!trace.playerId) {
      continue
    }
    counts.set(trace.playerId, (counts.get(trace.playerId) ?? 0) + 1)
    names.set(trace.playerId, trace.playerName ?? trace.playerId)
  }
  for (const player of room.players) {
    if (!player.isAI) {
      continue
    }
    if (!counts.has(player.id)) {
      counts.set(player.id, 0)
    }
    names.set(player.id, player.name)
  }
  return [...counts.entries()]
    .map(([id, count]) => ({ count, id, label: names.get(id) ?? id }))
    .sort((left, right) => {
      const leftPlayer = room.players.find(player => player.id === left.id)
      const rightPlayer = room.players.find(player => player.id === right.id)
      const leftSeat = leftPlayer?.seat ?? Number.MAX_SAFE_INTEGER
      const rightSeat = rightPlayer?.seat ?? Number.MAX_SAFE_INTEGER
      return leftSeat - rightSeat || left.label.localeCompare(right.label)
    })
}

type SocialMobilePanel = 'intel' | 'action' | 'log' | null

function SocialMobileDock({
  config,
  openPanel,
}: {
  config: typeof GAME_COPY[SocialGameSlug]
  openPanel: (panel: Exclude<SocialMobilePanel, null>) => void
}) {
  const { t } = useI18n()

  return (
    <nav className={cn('fixed inset-x-3 bottom-3 z-30 grid grid-cols-3 gap-2 rounded-lg border bg-black/58 p-2 shadow-[0_18px_60px_rgba(0,0,0,0.42)] backdrop-blur lg:hidden', config.panel)} aria-label="social game panels">
      <button className={socialButton(config, true)} type="button" onClick={() => openPanel('action')}>
        <ClipboardList className="size-4" />
        {t('social.mobileActions')}
      </button>
      <button className={socialButton(config)} type="button" onClick={() => openPanel('intel')}>
        <ShieldQuestion className="size-4" />
        {t('social.mobileIntel')}
      </button>
      <button className={socialButton(config)} type="button" onClick={() => openPanel('log')}>
        <ScrollText className="size-4" />
        {t('social.mobileLog')}
      </button>
    </nav>
  )
}

function SocialMobileDrawer({
  children,
  config,
  onClose,
  panel,
  room,
}: {
  children: ReactNode
  config: typeof GAME_COPY[SocialGameSlug]
  onClose: () => void
  panel: SocialMobilePanel
  room: SocialRoom
}) {
  const { t } = useI18n()

  if (!panel) {
    return null
  }

  return (
    <div className="fixed inset-0 z-40 lg:hidden" role="dialog" aria-modal="true" aria-label={mobilePanelTitle(panel, t)}>
      <button className="absolute inset-0 size-full cursor-default bg-black/62" aria-label={t('common.close')} type="button" onClick={onClose} />
      <div className={cn('absolute inset-x-0 bottom-0 grid max-h-[86svh] min-h-[42svh] grid-rows-[auto_minmax(0,1fr)] gap-3 overflow-hidden rounded-t-lg border p-4 shadow-[0_-24px_80px_rgba(0,0,0,0.5)]', config.panel)}>
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-black text-[#fff8e8]/62">
              {t(`social.phases.${room.phase}`)}
            </p>
            <h2 className="truncate text-xl font-black">{mobilePanelTitle(panel, t)}</h2>
          </div>
          <button className={cn(socialButton(config), 'min-h-9 shrink-0 px-2')} type="button" onClick={onClose}>
            <X className="size-4" />
            <span className="sr-only">{t('common.close')}</span>
          </button>
        </div>
        <div className="min-h-0 overflow-auto pr-1">
          {children}
        </div>
      </div>
    </div>
  )
}

function mobilePanelTitle(panel: Exclude<SocialMobilePanel, null>, t: (key: string) => string) {
  if (panel === 'action') {
    return t('social.mobileActions')
  }
  if (panel === 'intel') {
    return t('social.mobileIntel')
  }
  return t('social.mobileLog')
}
