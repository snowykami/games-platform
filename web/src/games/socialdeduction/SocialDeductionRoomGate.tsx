import type { FormEvent } from 'react'
import type { SocialGameSlug, SocialRoom } from './online'
import { Bot, Copy, DoorOpen, Plus, Sparkles } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { getAICapabilities } from '@/games/ai'
import { useAutoFollowScroll } from '@/games/useAutoFollowScroll'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { createSocialRoom } from './online'
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
  const { actions, error, isLoading, room } = useSocialRoom(game, roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('room.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [llmEnabled, setLLMEnabled] = useState(false)
  const [llmModel, setLLMModel] = useState('')
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
      setLLMModel(capabilities.model ?? '')
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
    return <SocialGamePage actions={actions} config={config} error={error} game={game} room={room} />
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
                  {llmEnabled ? `AI: ${llmModel || t('common.ai')}` : t('social.llmRequired')}
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

          <PlayerGrid actions={actions} config={config} isHost={isHost} llmModel={llmModel} room={room} />
          {game === 'undercover' && <UndercoverLobbyConfig actions={actions} config={config} isHost={isHost} room={room} />}

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <p className="text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
          </div>
        </div>

        <aside className={cn('grid content-start gap-3 rounded-lg border p-5 shadow-[0_22px_64px_rgba(0,0,0,0.28)]', config.panel)}>
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            {t(config.rulesTitleKey)}
          </h2>
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
          {ta(config.rulesKey).map(rule => (
            <p key={rule} className="rounded-lg bg-black/24 px-3 py-2 text-sm leading-6 text-[#fff8e8]/78">{rule}</p>
          ))}
        </aside>
      </section>
    </SocialShell>
  )
}

function SocialGamePage({
  actions,
  config,
  error,
  game,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  error?: string
  game: SocialGameSlug
  room: SocialRoom
}) {
  const { t } = useI18n()
  const [message, setMessage] = useState(() => t('social.connected'))
  const tableLogScroll = useAutoFollowScroll<HTMLDivElement>()
  const you = room.players.find(player => player.id === room.youPlayerId)
  const leader = room.players.find(player => player.id === room.avalon.leaderId)
  const isHost = Boolean(room.hostPlayerId && room.hostPlayerId === room.youPlayerId)

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
      <section className="grid h-full min-h-0 gap-3 overflow-hidden lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className={cn('grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 overflow-hidden rounded-lg border p-3 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-4', config.panel)}>
          <div className="flex flex-wrap items-center gap-2 rounded-lg border border-white/14 bg-black/24 p-2">
            <button className={socialButton(config, true)} type="button" onClick={copyLink}>
              <Copy className="size-4" />
              {t('common.copyLink')}
            </button>
            <StatusPill>{t(`social.phases.${room.phase}`)}</StatusPill>
            <StatusPill>
              {game === 'werewolf' ? t('werewolf.dayCount', { day: room.werewolf.day || 1 }) : t('avalon.roundCount', { round: room.avalon.round || 1 })}
            </StatusPill>
            {game === 'avalon' && <StatusPill>{t('avalon.leader', { name: leader?.name ?? '-' })}</StatusPill>}
            <StatusPill>
              {t('common.room')}
              {' '}
              {room.id}
            </StatusPill>
            <button className={cn(socialButton(config), 'ml-auto')} disabled={!isHost} type="button" onClick={restart}>
              {t('social.restart')}
            </button>
          </div>

          <div className="grid min-h-0 gap-3 overflow-hidden lg:grid-cols-[minmax(0,1fr)_260px]">
            <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-3 overflow-hidden">
              <SelfIntel config={config} game={game} room={room} you={you} />
              <PlayerGrid actions={actions} className="min-h-0" config={config} isHost={false} llmModel="" room={room} compact />
            </div>

            <div className="min-h-0 overflow-auto pr-1">
              <ActionPanel
                key={`${room.phase}:${room.avalon.round}:${room.werewolf.day}`}
                actions={actions}
                config={config}
                game={game}
                isHost={isHost}
                room={room}
                setMessage={setMessage}
                you={you}
              />
            </div>
          </div>

          <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
        </div>

        <aside className={cn('grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 overflow-hidden rounded-lg border p-4 shadow-[0_22px_64px_rgba(0,0,0,0.28)]', config.panel)}>
          <h2 className="text-xl font-black">{t('social.tableLog')}</h2>
          <div ref={tableLogScroll.containerRef} className="grid min-h-0 content-start gap-2 overflow-auto pr-1" onScroll={tableLogScroll.handleScroll}>
            {room.log.map(entry => (
              <TableLogLine key={entry.id} entry={entry} room={room} />
            ))}
          </div>
          {room.phase === 'finished' && (
            <div className="rounded-lg bg-[#fff8e8] p-3 text-[#171411]">
              <p className="text-xs font-black">{t('social.result')}</p>
              <strong>{room.winnerMessage}</strong>
            </div>
          )}
        </aside>
      </section>
    </SocialShell>
  )
}
