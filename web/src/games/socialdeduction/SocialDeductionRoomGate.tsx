import type { FormEvent, ReactNode } from 'react'
import type { SocialGameSlug, SocialPlayer, SocialRole, SocialRoom, WerewolfRoleCounts } from './online'
import { ArrowLeft, Bot, Copy, DoorOpen, Plus, Shield, Skull, Sparkles, UserMinus, Vote } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { getAICapabilities } from '@/games/ai'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerNoteEditor } from '@/games/PlayerNoteEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { createSocialRoom } from './online'
import { useSocialRoom } from './useSocialRoom'

const GAME_COPY = {
  werewolf: {
    titleKey: 'werewolf.title',
    subtitleKey: 'werewolf.subtitle',
    lobbyTitleKey: 'werewolf.lobbyTitle',
    lobbyDescriptionKey: 'werewolf.lobbyDescription',
    enterTitleKey: 'werewolf.enterTitle',
    rulesTitleKey: 'werewolf.rulesTitle',
    rulesKey: 'werewolf.rules',
    roomPrefix: 'WWF42',
    bg: 'bg-[#081018]',
    cover: '/game-covers/werewolf.webp',
    accent: 'text-[#ffd166]',
    panel: 'border-[#c8d7ff]/18 bg-[#101827]/76',
    button: 'border-[#c8d7ff]/20 bg-[#111b2a]/72 text-[#f7f1de] hover:bg-[#f7f1de] hover:text-[#111827]',
    primary: 'bg-[#ffd166] text-[#111827] hover:bg-[#ffe29a]',
  },
  avalon: {
    titleKey: 'avalon.title',
    subtitleKey: 'avalon.subtitle',
    lobbyTitleKey: 'avalon.lobbyTitle',
    lobbyDescriptionKey: 'avalon.lobbyDescription',
    enterTitleKey: 'avalon.enterTitle',
    rulesTitleKey: 'avalon.rulesTitle',
    rulesKey: 'avalon.rules',
    roomPrefix: 'AVL42',
    bg: 'bg-[#10140f]',
    cover: '/game-covers/avalon.webp',
    accent: 'text-[#d9b35f]',
    panel: 'border-[#f2dfad]/18 bg-[#162016]/78',
    button: 'border-[#f2dfad]/22 bg-[#1a2619]/72 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#172114]',
    primary: 'bg-[#d9b35f] text-[#172114] hover:bg-[#f2cf77]',
  },
  undercover: {
    titleKey: 'undercover.title',
    subtitleKey: 'undercover.subtitle',
    lobbyTitleKey: 'undercover.lobbyTitle',
    lobbyDescriptionKey: 'undercover.lobbyDescription',
    enterTitleKey: 'undercover.enterTitle',
    rulesTitleKey: 'undercover.rulesTitle',
    rulesKey: 'undercover.rules',
    roomPrefix: 'UND42',
    bg: 'bg-[#130f18]',
    cover: '/game-covers/undercover.svg',
    accent: 'text-[#f4c7ff]',
    panel: 'border-[#f4c7ff]/18 bg-[#211728]/78',
    button: 'border-[#f4c7ff]/22 bg-[#2a1b33]/72 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#211728]',
    primary: 'bg-[#f4c7ff] text-[#211728] hover:bg-[#ffe2ff]',
  },
} satisfies Record<SocialGameSlug, Record<string, string>>

const ROLES_BY_GAME: Record<SocialGameSlug, SocialRole[]> = {
  werewolf: ['villager', 'werewolf', 'seer', 'witch', 'hunter', 'idiot', 'guard'],
  avalon: ['merlin', 'assassin', 'minion', 'loyal'],
  undercover: ['civilian', 'undercover', 'blank'],
}

const ROLE_ALIGNMENT: Record<SocialRole, 'good' | 'evil' | 'neutral'> = {
  assassin: 'evil',
  guard: 'good',
  hunter: 'good',
  idiot: 'good',
  loyal: 'good',
  merlin: 'good',
  minion: 'evil',
  seer: 'good',
  villager: 'good',
  werewolf: 'evil',
  witch: 'good',
  civilian: 'good',
  undercover: 'evil',
  blank: 'neutral',
}

interface SocialDeductionRoomGateProps {
  game: SocialGameSlug
  roomId?: string
}

export function SocialDeductionRoomGate({ game, roomId }: SocialDeductionRoomGateProps) {
  const config = GAME_COPY[game]
  const navigate = useNavigate()
  const { user } = useAuth()
  const { t, ta } = useI18n()
  const { actions, error, isLoading, room } = useSocialRoom(game, roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('room.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [llmEnabled, setLLMEnabled] = useState(false)
  const [llmModel, setLLMModel] = useState('')
  const isHost = Boolean(user && room?.hostUserId === user.id)

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
    <SocialShell config={config} game={game}>
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
              </div>
            </div>
          </div>

          <PlayerGrid actions={actions} config={config} isHost={isHost} llmModel={llmModel} room={room} />
          {game === 'undercover' && <UndercoverLobbyConfig actions={actions} config={config} isHost={isHost} room={room} />}

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <button className={cn(socialButton(config, true), 'sm:w-40')} disabled={!isHost || room.players.length < room.minPlayers} type="button" onClick={startGame}>
              {t('common.startGame')}
            </button>
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

function WerewolfRoleSetup({
  actions,
  config,
  isHost,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  room: SocialRoom
}) {
  const { t } = useI18n()
  const [draft, setDraft] = useState<WerewolfRoleCounts>(room.werewolf.roleConfig.counts)
  const [message, setMessage] = useState('')
  const rolePresets = room.werewolf.rolePresets ?? []
  const total = roleTotal(draft)
  const canSave = isHost
    && total === room.players.length
    && draft.werewolf > 0
    && draft.werewolf < room.players.length
    && draft.seer <= 1
    && draft.guard <= 1
    && draft.witch <= 1
    && draft.hunter <= 1
    && draft.idiot <= 1

  async function selectPreset(presetId: string) {
    const preset = rolePresets.find(candidate => candidate.id === presetId)
    if (!preset) {
      return
    }
    setMessage(t('werewolf.updatingRoles'))
    await actions.updateWerewolfRoles({
      counts: preset.counts,
      description: preset.description,
      mode: 'preset',
      name: preset.name,
      presetId: preset.id,
    })
    setMessage(t('werewolf.rolesUpdated'))
  }

  async function saveCustom() {
    if (!canSave) {
      setMessage(t('werewolf.roleCountMismatch', { count: room.players.length, total }))
      return
    }
    setMessage(t('werewolf.updatingRoles'))
    await actions.updateWerewolfRoles({
      counts: draft,
      description: t('werewolf.customRolesDescription'),
      mode: 'custom',
      name: t('werewolf.customRoles'),
    })
    setMessage(t('werewolf.rolesUpdated'))
  }

  return (
    <section className="grid gap-3 rounded-lg border border-white/14 bg-black/22 p-3">
      <div>
        <h3 className="text-sm font-black">{t('werewolf.roleSetup')}</h3>
        <p className="mt-1 text-xs font-bold leading-5 text-[#fff8e8]/62">
          {t('werewolf.roleTotal', { count: room.players.length, total: roleTotal(room.werewolf.roleConfig.counts) })}
        </p>
      </div>

      <label className="grid gap-1 text-xs font-black">
        {t('werewolf.rolePreset')}
        <select
          className="min-h-10 rounded-lg border border-white/20 bg-black/30 px-3 text-sm font-black text-[#fff8e8] outline-none"
          disabled={!isHost || rolePresets.length === 0}
          value={room.werewolf.roleConfig.mode === 'preset' ? room.werewolf.roleConfig.presetId ?? '' : 'custom'}
          onChange={event => event.target.value === 'custom' ? undefined : void selectPreset(event.target.value)}
        >
          {rolePresets.length === 0 && <option value="">{t('werewolf.needMorePlayers')}</option>}
          {rolePresets.map(preset => <option key={preset.id} className="text-[#171411]" value={preset.id}>{preset.name}</option>)}
          <option className="text-[#171411]" value="custom">{t('werewolf.customRoles')}</option>
        </select>
      </label>

      <div className="grid gap-2">
        <RoleCountLine count={draft.werewolf} disabled={!isHost} role="werewolf" onChange={count => setDraft({ ...draft, werewolf: count })} />
        <RoleCountLine count={draft.villager} disabled={!isHost} role="villager" onChange={count => setDraft({ ...draft, villager: count })} />
        <RoleCountLine count={draft.seer} disabled={!isHost} max={1} role="seer" onChange={count => setDraft({ ...draft, seer: count })} />
        <RoleCountLine count={draft.witch} disabled={!isHost} max={1} role="witch" onChange={count => setDraft({ ...draft, witch: count })} />
        <RoleCountLine count={draft.hunter} disabled={!isHost} max={1} role="hunter" onChange={count => setDraft({ ...draft, hunter: count })} />
        <RoleCountLine count={draft.idiot} disabled={!isHost} max={1} role="idiot" onChange={count => setDraft({ ...draft, idiot: count })} />
        <RoleCountLine count={draft.guard} disabled={!isHost} max={1} role="guard" onChange={count => setDraft({ ...draft, guard: count })} />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button className={socialButton(config, true)} disabled={!canSave} type="button" onClick={() => void saveCustom()}>
          {t('werewolf.applyCustomRoles')}
        </button>
        <span className={cn('rounded-full px-2 py-1 text-xs font-black', total === room.players.length ? 'bg-emerald-200 text-emerald-950' : 'bg-[#4a1424] text-[#ffd6df]')}>
          {total}
          {' / '}
          {room.players.length}
        </span>
      </div>
      <p className="min-h-5 text-xs font-bold text-[#fff8e8]/65">{message || room.werewolf.roleConfig.description}</p>
    </section>
  )
}

function RoleCountLine({ count, disabled, max = 12, onChange, role }: { count: number, disabled: boolean, max?: number, onChange: (count: number) => void, role: SocialRole }) {
  const { t } = useI18n()
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 rounded-lg bg-black/24 px-2 py-2">
      <RoleBadge role={role} />
      <div className="grid grid-cols-[28px_34px_28px] items-center gap-1">
        <button className="grid size-7 place-items-center rounded-md bg-white/10 text-sm font-black disabled:opacity-40" disabled={disabled || count <= 0} type="button" onClick={() => onChange(Math.max(0, count - 1))}>-</button>
        <span className="text-center text-sm font-black" title={roleLabel(role, t)}>{count}</span>
        <button className="grid size-7 place-items-center rounded-md bg-white/10 text-sm font-black disabled:opacity-40" disabled={disabled || count >= max} type="button" onClick={() => onChange(count + 1)}>+</button>
      </div>
    </div>
  )
}

function UndercoverLobbyConfig({
  actions,
  config,
  isHost,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  room: SocialRoom
}) {
  const { t } = useI18n()
  const presets = room.undercover.presets ?? []
  const selectedPreset = presets.find(preset => preset.id === room.undercover.presetId)

  return (
    <section className="grid gap-2 rounded-lg border border-white/14 bg-black/24 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <strong className={config.accent}>{t('undercover.presetTitle')}</strong>
          <p className="text-xs font-bold text-[#fff8e8]/62">{selectedPreset?.description ?? t('undercover.presetHint')}</p>
        </div>
        <label className="flex items-center gap-2 text-xs font-black text-[#fff8e8]/78">
          <input
            checked={room.undercover.includeBlank}
            className="size-4 accent-[#f4c7ff]"
            disabled={!isHost}
            type="checkbox"
            onChange={event => void actions.undercoverConfig(room.undercover.presetId, event.target.checked)}
          />
          {t('undercover.includeBlank')}
        </label>
      </div>
      <div className="grid gap-2 sm:grid-cols-4">
        {presets.map(preset => (
          <button
            key={preset.id}
            className={cn(socialButton(config), preset.id === room.undercover.presetId && 'ring-2 ring-[#f4c7ff]')}
            disabled={!isHost}
            type="button"
            onClick={() => void actions.undercoverConfig(preset.id, room.undercover.includeBlank)}
          >
            {preset.name}
          </button>
        ))}
      </div>
    </section>
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
  const { user } = useAuth()
  const { t } = useI18n()
  const [message, setMessage] = useState(() => t('social.connected'))
  const you = room.players.find(player => player.id === room.youPlayerId || player.userId === user?.id)
  const leader = room.players.find(player => player.id === room.avalon.leaderId)
  const isHost = Boolean(user && room.hostUserId === user.id)

  async function copyLink() {
    await navigator.clipboard?.writeText(window.location.href)
    setMessage(t('room.copied'))
  }

  async function restart() {
    await actions.start()
    setMessage(t('social.restarted'))
  }

  return (
    <SocialShell config={config} game={game}>
      <section className="grid min-h-0 gap-3 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className={cn('grid min-h-[min(680px,calc(100svh-150px))] grid-rows-[auto_minmax(0,1fr)_auto] gap-3 rounded-lg border p-3 shadow-[0_22px_64px_rgba(0,0,0,0.28)] sm:p-4', config.panel)}>
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
            <div className="grid min-h-0 content-start gap-3 overflow-auto pr-1">
              <SelfIntel config={config} game={game} room={room} you={you} />
              <PlayerGrid actions={actions} config={config} isHost={false} llmModel="" room={room} compact />
            </div>

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

          <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
        </div>

        <aside className={cn('grid min-h-0 content-start gap-3 overflow-hidden rounded-lg border p-4 shadow-[0_22px_64px_rgba(0,0,0,0.28)]', config.panel)}>
          <h2 className="text-xl font-black">{t('social.tableLog')}</h2>
          <div className="grid max-h-[34svh] gap-2 overflow-auto pr-1">
            {room.log.map(entry => (
              <p key={entry.id} className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/76">{entry.text}</p>
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

function ActionPanel({
  actions,
  config,
  game,
  isHost,
  room,
  setMessage,
  you,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  isHost: boolean
  room: SocialRoom
  setMessage: (message: string) => void
  you?: SocialPlayer
}) {
  const { t } = useI18n()
  const [selectedTeam, setSelectedTeam] = useState<string[]>(room.avalon.team)
  const [description, setDescription] = useState('')
  const livingTargets = room.players.filter(player => player.alive && player.id !== you?.id)
  const alivePlayers = room.players.filter(player => player.alive)
  const onQuest = Boolean(you && room.avalon.team.includes(you.id))
  const isLeader = Boolean(you && room.avalon.leaderId === you.id)
  const isAssassin = you?.role === 'assassin'
  const isCurrentUndercoverSpeaker = Boolean(you && room.undercover.currentSpeakerId === you.id)
  const hunterPending = room.players.find(player => player.id === room.werewolf.hunterPendingId)

  function toggleTeam(playerId: string) {
    setSelectedTeam(selectedTeam.includes(playerId)
      ? selectedTeam.filter(id => id !== playerId)
      : selectedTeam.length < room.avalon.requiredTeam ? [...selectedTeam, playerId] : selectedTeam)
  }

  if (!you) {
    return <Panel config={config}>{t('room.connecting')}</Panel>
  }

  if (room.phase === 'finished') {
    return (
      <Panel config={config}>
        <Skull className="size-6" />
        <h2 className="text-xl font-black">{t('social.finished')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{room.winnerMessage}</p>
      </Panel>
    )
  }

  if (game === 'werewolf') {
    if (room.phase === 'night') {
      const canAct = ['werewolf', 'seer', 'guard', 'witch'].includes(you.role ?? '')
      const witchVictim = room.players.find(player => player.id === room.werewolf.witchVictimId)
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.nightAction')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">{canAct ? t('werewolf.chooseNightTarget') : t('werewolf.noNightAction')}</p>
          {you.role === 'witch' && (
            <>
              <p className="rounded-lg bg-black/24 px-3 py-2 text-xs font-black text-[#fff8e8]/72">
                {witchVictim ? t('werewolf.witchVictim', { name: witchVictim.name }) : t('werewolf.witchNoVictim')}
              </p>
              {witchVictim && !room.werewolf.witchAntidoteUsed && (
                <button className={socialButton(config, true)} type="button" onClick={() => void actions.nightAction(`save:${witchVictim.id}`).then(() => setMessage(t('werewolf.actionSubmitted')))}>
                  <Shield className="size-4" />
                  {t('werewolf.useAntidote', { name: witchVictim.name })}
                </button>
              )}
              {!room.werewolf.witchPoisonUsed && livingTargets.map(player => (
                <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.nightAction(`poison:${player.id}`).then(() => setMessage(t('werewolf.actionSubmitted')))}>
                  <Skull className="size-4" />
                  {t('werewolf.usePoison', { name: player.name })}
                </button>
              ))}
              <button className={socialButton(config)} type="button" onClick={() => void actions.nightAction('skip:witch').then(() => setMessage(t('werewolf.actionSubmitted')))}>
                {t('werewolf.skipWitch')}
              </button>
            </>
          )}
          {canAct && you.role !== 'witch' && livingTargets.map(player => (
            <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.nightAction(`target:${player.id}`).then(() => setMessage(t('werewolf.actionSubmitted')))}>
              <Shield className="size-4" />
              {player.name}
            </button>
          ))}
        </Panel>
      )
    }

    if (room.phase === 'hunter') {
      const canShoot = you.id === room.werewolf.hunterPendingId
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.hunterShot')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">
            {canShoot ? t('werewolf.chooseHunterTarget') : t('werewolf.waitHunter', { name: hunterPending?.name ?? '-' })}
          </p>
          {canShoot && livingTargets.map(player => (
            <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.hunterShot(player.id).then(() => setMessage(t('werewolf.hunterSubmitted')))}>
              <Skull className="size-4" />
              {player.name}
            </button>
          ))}
          {canShoot && (
            <button className={socialButton(config, true)} type="button" onClick={() => void actions.hunterShot('').then(() => setMessage(t('werewolf.hunterSubmitted')))}>
              {t('werewolf.skipHunter')}
            </button>
          )}
        </Panel>
      )
    }

    if (room.phase === 'day') {
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.dayDiscussion')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">{room.werewolf.lastNight || t('werewolf.dayHint')}</p>
          <button className={socialButton(config, true)} disabled={!isHost} type="button" onClick={() => void actions.advanceDay().then(() => setMessage(t('werewolf.voteStarted')))}>
            <Vote className="size-4" />
            {t('werewolf.startVote')}
          </button>
        </Panel>
      )
    }

    if (room.phase === 'vote') {
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.exileVote')}</h2>
          {livingTargets.map(player => (
            <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.werewolfVote(player.id).then(() => setMessage(t('werewolf.voted')))}>
              <Vote className="size-4" />
              {player.name}
            </button>
          ))}
        </Panel>
      )
    }
  }

  if (game === 'undercover') {
    if (room.phase === 'describe') {
      const currentSpeaker = room.players.find(player => player.id === room.undercover.currentSpeakerId)
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('undercover.describeTitle')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">
            {isCurrentUndercoverSpeaker ? t('undercover.yourTurnDescribe') : t('undercover.waitDescribe', { name: currentSpeaker?.name ?? '-' })}
          </p>
          <textarea
            className="min-h-24 resize-none rounded-lg border border-white/18 bg-black/28 p-3 text-sm font-bold text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#f4c7ff]"
            disabled={!isCurrentUndercoverSpeaker}
            maxLength={80}
            placeholder={t('undercover.describePlaceholder')}
            value={description}
            onChange={event => setDescription(event.target.value)}
          />
          <button
            className={socialButton(config, true)}
            disabled={!isCurrentUndercoverSpeaker || !description.trim()}
            type="button"
            onClick={() => void actions.undercoverDescribe(description).then(() => {
              setDescription('')
              setMessage(t('undercover.described'))
            })}
          >
            {t('undercover.submitDescription')}
          </button>
        </Panel>
      )
    }

    if (room.phase === 'undercover_vote') {
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('undercover.voteTitle')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">{t('undercover.voteHint')}</p>
          {livingTargets.map(player => (
            <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.undercoverVote(player.id).then(() => setMessage(t('undercover.voted')))}>
              <Vote className="size-4" />
              {player.name}
            </button>
          ))}
        </Panel>
      )
    }
  }

  if (room.phase === 'team') {
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.proposeTeam')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{t('avalon.teamSize', { count: room.avalon.requiredTeam })}</p>
        <div className="grid gap-2">
          {alivePlayers.map(player => (
            <button
              key={player.id}
              className={cn(socialButton(config), selectedTeam.includes(player.id) && 'ring-2 ring-[#fff8e8]')}
              disabled={!isLeader}
              type="button"
              onClick={() => toggleTeam(player.id)}
            >
              {player.name}
            </button>
          ))}
        </div>
        <button
          className={socialButton(config, true)}
          disabled={!isLeader || selectedTeam.length !== room.avalon.requiredTeam}
          type="button"
          onClick={() => void actions.proposeTeam(selectedTeam).then(() => setMessage(t('avalon.teamProposed')))}
        >
          {t('avalon.submitTeam')}
        </button>
      </Panel>
    )
  }

  if (room.phase === 'team_vote') {
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.teamVote')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{room.avalon.team.map(id => room.players.find(player => player.id === id)?.name).filter(Boolean).join(' / ')}</p>
        <div className="grid grid-cols-2 gap-2">
          <button className={socialButton(config, true)} type="button" onClick={() => void actions.teamVote(true).then(() => setMessage(t('avalon.voted')))}>{t('avalon.approve')}</button>
          <button className={socialButton(config)} type="button" onClick={() => void actions.teamVote(false).then(() => setMessage(t('avalon.voted')))}>{t('avalon.reject')}</button>
        </div>
      </Panel>
    )
  }

  if (room.phase === 'quest') {
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.quest')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{onQuest ? t('avalon.playQuestCard') : t('avalon.waitQuest')}</p>
        {onQuest && (
          <div className="grid grid-cols-2 gap-2">
            <button className={socialButton(config, true)} type="button" onClick={() => void actions.playQuest('success').then(() => setMessage(t('avalon.questSubmitted')))}>{t('avalon.successCard')}</button>
            <button className={socialButton(config)} disabled={you.alignment !== 'evil'} type="button" onClick={() => void actions.playQuest('fail').then(() => setMessage(t('avalon.questSubmitted')))}>{t('avalon.failCard')}</button>
          </div>
        )}
      </Panel>
    )
  }

  if (room.phase === 'assassination') {
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.assassination')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{isAssassin ? t('avalon.chooseMerlin') : t('avalon.waitAssassin')}</p>
        {isAssassin && room.players.filter(player => player.alignment === 'good' || !player.visibleToYou).map(player => (
          <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.assassinate(player.id).then(() => setMessage(t('avalon.assassinated')))}>
            <Skull className="size-4" />
            {player.name}
          </button>
        ))}
      </Panel>
    )
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}

function PlayerGrid({
  actions,
  compact = false,
  config,
  isHost,
  llmModel,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  compact?: boolean
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  llmModel: string
  room: SocialRoom
}) {
  const { t } = useI18n()
  return (
    <div className={cn('grid content-start gap-3 overflow-auto pr-1', compact ? 'sm:grid-cols-2 xl:grid-cols-3' : 'sm:grid-cols-2')}>
      {room.players.map(player => (
        <article key={player.id} className="relative rounded-lg border border-white/16 bg-black/26 p-3 shadow-[0_16px_34px_rgba(0,0,0,0.18)]">
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2">
              <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
              <strong className="truncate text-base">{player.name}</strong>
              {player.id === room.youPlayerId && <PlayerNameEditor buttonClassName="text-[#fff8e8]" className="min-w-[120px]" name={player.name} onSave={actions.renamePlayer} />}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {player.id === room.youPlayerId && <SpeechButton onSend={actions.say} />}
              {player.ai
                ? (
                    <span className="shrink-0 rounded-full bg-[#fff8e8] px-2 py-0.5 text-xs font-black text-[#171411]">
                      {`AI: ${llmModel || t('common.ai')}`}
                    </span>
                  )
                : (
                    <span className="rounded-full bg-[#fff8e8] px-2 py-0.5 text-xs font-black text-[#171411]">
                      {player.roomRole === 'host' ? t('common.host') : player.connected ? t('common.online') : t('common.offline')}
                    </span>
                  )}
              {isHost && room.phase === 'lobby' && player.roomRole !== 'host' && (
                <button
                  aria-label={t('common.removePlayer')}
                  className={cn(socialIconButton(config))}
                  title={t('common.removePlayer')}
                  type="button"
                  onClick={() => void actions.removePlayer(player.id)}
                >
                  <UserMinus className="size-4" />
                </button>
              )}
            </div>
          </div>
          <SpeechBubble speech={latestSpeechForPlayer(room.speeches, player.id)} />
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <span className={cn('rounded-full px-2 py-0.5 text-xs font-black', player.alive ? 'bg-emerald-200 text-emerald-950' : 'bg-zinc-300 text-zinc-950')}>
              {player.alive ? t('social.alive') : t('social.out')}
            </span>
            {player.visibleToYou && player.role
              ? <RoleBadge role={player.role} />
              : <span className="rounded-full bg-white/12 px-2 py-0.5 text-xs font-black">{t('social.hiddenRole')}</span>}
          </div>
          <p className="mt-2 min-h-8 text-sm leading-6 text-[#fff8e8]/70">
            {player.ai?.personality ?? (player.kind === 'guest' ? t('xiangqi.guestReady') : t('xiangqi.oidcReady'))}
          </p>
          {room.youPlayerId && player.id !== room.youPlayerId && (
            <PlayerNoteEditor className="mt-2" note={player.note} onSave={note => actions.updatePlayerNote(player.id, note)} />
          )}
        </article>
      ))}
    </div>
  )
}

function SelfIntel({ config, game, room, you }: { config: typeof GAME_COPY[SocialGameSlug], game: SocialGameSlug, room: SocialRoom, you?: SocialPlayer }) {
  const { t } = useI18n()
  const visiblePlayers = room.players.filter(player => player.visibleToYou && player.id !== you?.id && player.role)
  const seerChecks = Object.entries(room.werewolf.seerChecks ?? {})
  const yourWord = undercoverWord(room, you)
  return (
    <section className={cn('rounded-lg border p-4', config.panel)}>
      <h2 className={cn('text-xl font-black', config.accent)}>{t('social.yourIntel')}</h2>
      <div className="mt-3 flex flex-wrap gap-2">
        {you?.role ? <RoleBadge role={you.role} size="large" /> : <span className="rounded-full bg-white/12 px-3 py-1 text-sm font-black">{t('social.hiddenRole')}</span>}
        {you?.alignment && <AlignmentBadge alignment={you.alignment} />}
      </div>
      <p className="mt-3 text-sm leading-6 text-[#fff8e8]/72">
        {game === 'werewolf' ? t('werewolf.intelHint') : game === 'undercover' ? t('undercover.intelHint') : t('avalon.intelHint')}
      </p>
      {game === 'undercover' && (
        <div className="mt-3 rounded-lg bg-[#fff8e8] p-3 text-[#211728]">
          <p className="text-xs font-black">{t('undercover.yourWord')}</p>
          <strong className="text-2xl">{yourWord || t('undercover.blankWord')}</strong>
          {room.phase === 'finished' && room.undercover.wordPair && (
            <p className="mt-2 text-xs font-black">
              {t('undercover.finalWords', {
                civilian: room.undercover.wordPair.civilianWord ?? '-',
                undercover: room.undercover.wordPair.undercoverWord ?? '-',
              })}
            </p>
          )}
        </div>
      )}
      {visiblePlayers.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {visiblePlayers.map(player => (
            <span key={player.id} className="inline-flex items-center gap-1.5 rounded-full bg-black/28 px-3 py-1 text-xs font-black">
              {player.name}
              <RoleBadge role={player.role!} />
            </span>
          ))}
        </div>
      )}
      {game === 'werewolf' && seerChecks.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {seerChecks.map(([playerId, alignment]) => (
            <span key={playerId} className={cn('rounded-full px-3 py-1 text-xs font-black', alignmentClass(alignment))}>
              {room.players.find(player => player.id === playerId)?.name ?? t('common.player')}
              {' · '}
              {t(`social.alignments.${alignment}`)}
            </span>
          ))}
        </div>
      )}
    </section>
  )
}

function undercoverWord(room: SocialRoom, you?: SocialPlayer) {
  if (!you || !room.undercover.wordPair) {
    return ''
  }
  if (you.role === 'undercover') {
    return room.undercover.wordPair.undercoverWord ?? ''
  }
  if (you.role === 'civilian') {
    return room.undercover.wordPair.civilianWord ?? ''
  }
  return ''
}

function RoleList({ game }: { game: SocialGameSlug }) {
  return (
    <div className="flex flex-wrap gap-2">
      {ROLES_BY_GAME[game].map(role => <RoleBadge key={role} role={role} />)}
    </div>
  )
}

function RoleBadge({ role, size = 'normal' }: { role: SocialRole, size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <span className={cn('rounded-full px-2 py-0.5 font-black', size === 'large' ? 'text-sm' : 'text-xs', alignmentClass(ROLE_ALIGNMENT[role]))}>
      {roleLabel(role, t)}
    </span>
  )
}

function AlignmentBadge({ alignment }: { alignment: 'good' | 'evil' | 'neutral' }) {
  const { t } = useI18n()
  return <span className={cn('rounded-full px-3 py-1 text-sm font-black', alignmentClass(alignment))}>{t(`social.alignments.${alignment}`)}</span>
}

function alignmentClass(alignment: 'good' | 'evil' | 'neutral') {
  if (alignment === 'evil') {
    return 'bg-[#4a1424] text-[#ffd6df] ring-1 ring-[#ff7a9a]/35'
  }
  if (alignment === 'neutral') {
    return 'bg-[#e7d5ff] text-[#241833] ring-1 ring-[#f4c7ff]/45'
  }
  return 'bg-[#fff0b8] text-[#1f2114] ring-1 ring-[#ffd166]/45'
}

function roleTotal(counts: WerewolfRoleCounts) {
  return (counts.villager ?? 0)
    + (counts.werewolf ?? 0)
    + (counts.seer ?? 0)
    + (counts.guard ?? 0)
    + (counts.witch ?? 0)
    + (counts.hunter ?? 0)
    + (counts.idiot ?? 0)
}

function SocialShell({ children, config, game }: { children: ReactNode, config: typeof GAME_COPY[SocialGameSlug], game: SocialGameSlug }) {
  const { t } = useI18n()
  return (
    <main className={cn('relative min-h-svh overflow-y-auto text-[#fff8e8]', config.bg)}>
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(255,209,102,0.12),transparent_28%),radial-gradient(circle_at_88%_12%,rgba(115,171,191,0.16),transparent_26%)]" />
      <div className="relative mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
        <header className="flex items-end justify-between gap-4">
          <div>
            <p className="mb-1 text-xs font-black tracking-normal text-[#fff8e8]/72">{t(GAME_COPY[game].subtitleKey)}</p>
            <h1 className="text-[clamp(40px,8vw,84px)] font-black leading-[0.82] tracking-normal [text-shadow:0_8px_0_rgba(0,0,0,0.34)]">
              {t(GAME_COPY[game].titleKey)}
            </h1>
          </div>
          <Link className={cn('inline-grid min-h-10 place-items-center rounded-full border px-4 text-sm font-bold transition', config.button)} to="/">
            <ArrowLeft className="mr-2 inline size-4" />
            {t('common.backToLobby')}
          </Link>
        </header>
        {children}
      </div>
    </main>
  )
}

function Panel({ children, config }: { children: ReactNode, config: typeof GAME_COPY[SocialGameSlug] }) {
  return <aside className={cn('grid content-start gap-3 rounded-lg border p-4', config.panel)}>{children}</aside>
}

function StatusPill({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-white/16 bg-white/10 px-3 text-xs font-black text-[#fff8e8]/86 sm:text-sm">
      {children}
    </span>
  )
}

function socialButton(config: typeof GAME_COPY[SocialGameSlug], primary = false) {
  return cn('inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border px-3 text-sm font-black transition disabled:cursor-not-allowed disabled:opacity-45', primary ? config.primary : config.button)
}

function socialIconButton(config: typeof GAME_COPY[SocialGameSlug]) {
  return cn('inline-grid size-8 place-items-center rounded-lg border transition', config.button)
}

function roleLabel(role: SocialRole, t: (key: string) => string) {
  return t(`social.roles.${role}`)
}
