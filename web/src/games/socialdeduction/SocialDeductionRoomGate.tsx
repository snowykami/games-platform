import type { FormEvent, KeyboardEvent, ReactNode } from 'react'
import type { SocialGameSlug, SocialPlayer, SocialRole, SocialRoom, WerewolfRoleCounts } from './online'
import { ArrowLeft, Bot, Copy, DoorOpen, Plus, Shield, Skull, Sparkles, UserMinus, Vote } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { getAICapabilities } from '@/games/ai'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerNoteEditor } from '@/games/PlayerNoteEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { useAutoFollowScroll } from '@/games/useAutoFollowScroll'
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

const PLAYER_COLOR_PALETTE = [
  { border: 'rgba(96,165,250,0.62)', ink: '#1d4ed8', soft: 'rgba(96,165,250,0.16)', solid: '#60a5fa', text: '#bfdbfe' },
  { border: 'rgba(251,113,133,0.62)', ink: '#be123c', soft: 'rgba(251,113,133,0.16)', solid: '#fb7185', text: '#fecdd3' },
  { border: 'rgba(52,211,153,0.62)', ink: '#047857', soft: 'rgba(52,211,153,0.16)', solid: '#34d399', text: '#a7f3d0' },
  { border: 'rgba(251,191,36,0.64)', ink: '#a16207', soft: 'rgba(251,191,36,0.18)', solid: '#fbbf24', text: '#fde68a' },
  { border: 'rgba(196,181,253,0.64)', ink: '#6d28d9', soft: 'rgba(196,181,253,0.17)', solid: '#c4b5fd', text: '#ddd6fe' },
  { border: 'rgba(45,212,191,0.62)', ink: '#0f766e', soft: 'rgba(45,212,191,0.16)', solid: '#2dd4bf', text: '#99f6e4' },
  { border: 'rgba(244,114,182,0.62)', ink: '#be185d', soft: 'rgba(244,114,182,0.16)', solid: '#f472b6', text: '#fbcfe8' },
  { border: 'rgba(129,140,248,0.62)', ink: '#4338ca', soft: 'rgba(129,140,248,0.16)', solid: '#818cf8', text: '#c7d2fe' },
  { border: 'rgba(251,146,60,0.62)', ink: '#c2410c', soft: 'rgba(251,146,60,0.16)', solid: '#fb923c', text: '#fed7aa' },
  { border: 'rgba(125,211,252,0.62)', ink: '#0369a1', soft: 'rgba(125,211,252,0.16)', solid: '#7dd3fc', text: '#bae6fd' },
  { border: 'rgba(163,230,53,0.62)', ink: '#4d7c0f', soft: 'rgba(163,230,53,0.16)', solid: '#a3e635', text: '#d9f99d' },
  { border: 'rgba(250,204,21,0.62)', ink: '#854d0e', soft: 'rgba(250,204,21,0.16)', solid: '#facc15', text: '#fef08a' },
] satisfies PlayerAccent[]

interface PlayerAccent {
  border: string
  ink: string
  soft: string
  solid: string
  text: string
}

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
    <SocialShell config={config} game={game} phase={room.phase}>
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
          <div ref={tableLogScroll.containerRef} className="grid max-h-[34svh] gap-2 overflow-auto pr-1" onScroll={tableLogScroll.handleScroll}>
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
  const [selectedWerewolfVote, setSelectedWerewolfVote] = useState('')
  const [selectedHunterTarget, setSelectedHunterTarget] = useState('')
  const [selectedUndercoverVote, setSelectedUndercoverVote] = useState('')
  const [selectedTeamVote, setSelectedTeamVote] = useState<boolean>()
  const [teamVoteSubmitted, setTeamVoteSubmitted] = useState(false)
  const [selectedQuestCard, setSelectedQuestCard] = useState<'success' | 'fail'>()
  const [questSubmitted, setQuestSubmitted] = useState(false)
  const [selectedAssassinationTarget, setSelectedAssassinationTarget] = useState('')
  const [description, setDescription] = useState('')
  const livingTargets = room.players.filter(player => player.alive && player.id !== you?.id)
  const alivePlayers = room.players.filter(player => player.alive)
  const yourWerewolfVote = you ? room.werewolf.votes[you.id] : undefined
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

  async function submitUndercoverDescription() {
    const nextDescription = description.trim()
    if (!isCurrentUndercoverSpeaker || !nextDescription) {
      return
    }

    await actions.undercoverDescribe(nextDescription)
    setDescription('')
    setMessage(t('undercover.described'))
  }

  function handleDescriptionKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) {
      return
    }

    event.preventDefault()
    void submitUndercoverDescription()
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

  const canUseHunterDeathAction = game === 'werewolf' && room.phase === 'hunter' && you.id === room.werewolf.hunterPendingId
  if (!you.alive && !canUseHunterDeathAction) {
    return (
      <DeadPlayerPanel
        actions={actions}
        config={config}
        game={game}
        isHost={isHost}
        room={room}
        setMessage={setMessage}
      />
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
                {witchVictim
                  ? (
                      <span className="inline-flex flex-wrap items-center gap-2">
                        {t('werewolf.witchVictimPrefix')}
                        <PlayerRefLabel player={witchVictim} room={room} />
                      </span>
                    )
                  : t('werewolf.witchNoVictim')}
              </p>
              {witchVictim && !room.werewolf.witchAntidoteUsed && (
                <button className={socialButton(config, true)} type="button" onClick={() => void actions.nightAction(`save:${witchVictim.id}`).then(() => setMessage(t('werewolf.actionSubmitted')))}>
                  <Shield className="size-4" />
                  {t('werewolf.useAntidoteTarget')}
                  <PlayerRefLabel player={witchVictim} room={room} />
                </button>
              )}
              {!room.werewolf.witchPoisonUsed && livingTargets.map(player => (
                <button key={player.id} className={socialButton(config)} type="button" onClick={() => void actions.nightAction(`poison:${player.id}`).then(() => setMessage(t('werewolf.actionSubmitted')))}>
                  <Skull className="size-4" />
                  {t('werewolf.usePoisonTarget')}
                  <PlayerRefLabel player={player} room={room} />
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
              <PlayerRefLabel player={player} room={room} />
            </button>
          ))}
        </Panel>
      )
    }

    if (room.phase === 'hunter') {
      const canShoot = you.id === room.werewolf.hunterPendingId
      const selectedHunterPlayer = room.players.find(player => player.id === selectedHunterTarget)
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.hunterShot')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">
            {canShoot
              ? t('werewolf.chooseHunterTarget')
              : (
                  <span className="inline-flex flex-wrap items-center gap-2">
                    {t('werewolf.waitHunterPrefix')}
                    {hunterPending ? <PlayerRefLabel player={hunterPending} room={room} /> : '-'}
                  </span>
                )}
          </p>
          {canShoot && livingTargets.map(player => (
            <ChoiceButton key={player.id} config={config} icon={<Skull className="size-4" />} selected={selectedHunterTarget === player.id} onClick={() => setSelectedHunterTarget(player.id)}>
              <PlayerRefLabel player={player} room={room} />
            </ChoiceButton>
          ))}
          {canShoot && (
            <button className={cn(socialButton(config), selectedHunterTarget === 'skip' && 'ring-2 ring-[#fff8e8]')} type="button" onClick={() => setSelectedHunterTarget('skip')}>
              {t('werewolf.skipHunter')}
            </button>
          )}
          {canShoot && (
            <ConfirmChoiceButton
              config={config}
              disabled={!selectedHunterTarget}
              label={t('social.confirmAction')}
              selectedLabel={selectedHunterTarget === 'skip' ? t('werewolf.skipHunter') : selectedHunterPlayer ? <PlayerRefLabel player={selectedHunterPlayer} room={room} /> : undefined}
              onConfirm={() => void actions.hunterShot(selectedHunterTarget === 'skip' ? '' : selectedHunterTarget).then(() => {
                setMessage(t('werewolf.hunterSubmitted'))
              })}
            />
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
      const hasVoted = Boolean(yourWerewolfVote?.confirmed)
      const activeWerewolfVoteTarget = selectedWerewolfVote || yourWerewolfVote?.targetId || ''
      const selectedPlayer = room.players.find(player => player.id === activeWerewolfVoteTarget)
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('werewolf.exileVote')}</h2>
          {hasVoted && <SubmittedNotice config={config} label={t('werewolf.votedCanChange')} />}
          {livingTargets.map(player => (
            <ChoiceButton
              key={player.id}
              config={config}
              icon={<Vote className="size-4" />}
              selected={activeWerewolfVoteTarget === player.id}
              onClick={() => {
                setSelectedWerewolfVote(player.id)
                void actions.werewolfVote(player.id, false)
              }}
            >
              <PlayerRefLabel player={player} room={room} />
            </ChoiceButton>
          ))}
          <ConfirmChoiceButton
            config={config}
            disabled={!activeWerewolfVoteTarget || hasVoted}
            label={hasVoted ? t('werewolf.voted') : t('social.confirmVote')}
            selectedLabel={selectedPlayer ? <PlayerRefLabel player={selectedPlayer} room={room} /> : undefined}
            onConfirm={() => void actions.werewolfVote(activeWerewolfVoteTarget, true).then(() => setMessage(t('werewolf.voted')))}
          />
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
            onKeyDown={handleDescriptionKeyDown}
            onChange={event => setDescription(event.target.value)}
          />
          <button
            className={socialButton(config, true)}
            disabled={!isCurrentUndercoverSpeaker || !description.trim()}
            type="button"
            onClick={() => void submitUndercoverDescription()}
          >
            {t('undercover.submitDescription')}
          </button>
        </Panel>
      )
    }

    if (room.phase === 'undercover_vote') {
      const yourUndercoverVote = room.undercover.votes[you.id]
      const hasVoted = Boolean(yourUndercoverVote?.confirmed)
      const activeUndercoverVoteTarget = selectedUndercoverVote || yourUndercoverVote?.targetId || ''
      const selectedPlayer = room.players.find(player => player.id === activeUndercoverVoteTarget)
      return (
        <Panel config={config}>
          <h2 className="text-xl font-black">{t('undercover.voteTitle')}</h2>
          <p className="text-sm leading-6 text-[#fff8e8]/76">{t('undercover.voteHint')}</p>
          {hasVoted && <SubmittedNotice config={config} label={t('undercover.voted')} />}
          {livingTargets.map(player => (
            <ChoiceButton
              key={player.id}
              config={config}
              icon={<Vote className="size-4" />}
              selected={activeUndercoverVoteTarget === player.id}
              onClick={() => {
                setSelectedUndercoverVote(player.id)
                void actions.undercoverVote(player.id, false)
              }}
            >
              {player.name}
            </ChoiceButton>
          ))}
          <ConfirmChoiceButton
            config={config}
            disabled={!activeUndercoverVoteTarget || hasVoted}
            label={hasVoted ? t('undercover.voted') : t('social.confirmVote')}
            selectedLabel={selectedPlayer?.name}
            onConfirm={() => void actions.undercoverVote(activeUndercoverVoteTarget, true).then(() => setMessage(t('undercover.voted')))}
          />
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
    const selectedVoteLabel = selectedTeamVote === undefined ? undefined : selectedTeamVote ? t('avalon.approve') : t('avalon.reject')
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.teamVote')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{room.avalon.team.map(id => room.players.find(player => player.id === id)?.name).filter(Boolean).join(' / ')}</p>
        {teamVoteSubmitted && <SubmittedNotice config={config} label={t('avalon.voted')} />}
        <div className="grid grid-cols-2 gap-2">
          <ChoiceButton config={config} disabled={teamVoteSubmitted} selected={selectedTeamVote === true} onClick={() => setSelectedTeamVote(true)}>{t('avalon.approve')}</ChoiceButton>
          <ChoiceButton config={config} disabled={teamVoteSubmitted} selected={selectedTeamVote === false} onClick={() => setSelectedTeamVote(false)}>{t('avalon.reject')}</ChoiceButton>
        </div>
        {!teamVoteSubmitted && (
          <ConfirmChoiceButton
            config={config}
            disabled={selectedTeamVote === undefined}
            label={t('social.confirmVote')}
            selectedLabel={selectedVoteLabel}
            onConfirm={() => void actions.teamVote(Boolean(selectedTeamVote)).then(() => {
              setTeamVoteSubmitted(true)
              setMessage(t('avalon.voted'))
            })}
          />
        )}
      </Panel>
    )
  }

  if (room.phase === 'quest') {
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.quest')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{onQuest ? t('avalon.playQuestCard') : t('avalon.waitQuest')}</p>
        {onQuest && (
          <>
            {questSubmitted && <SubmittedNotice config={config} label={t('avalon.questSubmitted')} />}
            <div className="grid grid-cols-2 gap-2">
              <ChoiceButton config={config} disabled={questSubmitted} selected={selectedQuestCard === 'success'} onClick={() => setSelectedQuestCard('success')}>{t('avalon.successCard')}</ChoiceButton>
              <ChoiceButton config={config} disabled={questSubmitted || you.alignment !== 'evil'} selected={selectedQuestCard === 'fail'} onClick={() => setSelectedQuestCard('fail')}>{t('avalon.failCard')}</ChoiceButton>
            </div>
            {!questSubmitted && (
              <ConfirmChoiceButton
                config={config}
                disabled={!selectedQuestCard}
                label={t('social.confirmAction')}
                selectedLabel={selectedQuestCard ? t(selectedQuestCard === 'success' ? 'avalon.successCard' : 'avalon.failCard') : undefined}
                onConfirm={() => {
                  if (!selectedQuestCard) {
                    return
                  }
                  void actions.playQuest(selectedQuestCard).then(() => {
                    setQuestSubmitted(true)
                    setMessage(t('avalon.questSubmitted'))
                  })
                }}
              />
            )}
          </>
        )}
      </Panel>
    )
  }

  if (room.phase === 'assassination') {
    const selectedPlayer = room.players.find(player => player.id === selectedAssassinationTarget)
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('avalon.assassination')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{isAssassin ? t('avalon.chooseMerlin') : t('avalon.waitAssassin')}</p>
        {isAssassin && room.players.filter(player => player.alignment === 'good' || !player.visibleToYou).map(player => (
          <ChoiceButton key={player.id} config={config} icon={<Skull className="size-4" />} selected={selectedAssassinationTarget === player.id} onClick={() => setSelectedAssassinationTarget(player.id)}>
            {player.name}
          </ChoiceButton>
        ))}
        {isAssassin && (
          <ConfirmChoiceButton
            config={config}
            disabled={!selectedAssassinationTarget}
            label={t('social.confirmAction')}
            selectedLabel={selectedPlayer?.name}
            onConfirm={() => void actions.assassinate(selectedAssassinationTarget).then(() => setMessage(t('avalon.assassinated')))}
          />
        )}
      </Panel>
    )
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}

function ChoiceButton({
  children,
  config,
  disabled = false,
  icon,
  onClick,
  selected,
}: {
  children: ReactNode
  config: typeof GAME_COPY[SocialGameSlug]
  disabled?: boolean
  icon?: ReactNode
  onClick: () => void
  selected: boolean
}) {
  return (
    <button
      className={cn(socialButton(config), selected && 'ring-2 ring-[#fff8e8] ring-offset-2 ring-offset-black/40')}
      disabled={disabled}
      type="button"
      onClick={onClick}
    >
      {icon}
      {children}
    </button>
  )
}

function ConfirmChoiceButton({
  config,
  disabled,
  label,
  onConfirm,
  selectedLabel,
}: {
  config: typeof GAME_COPY[SocialGameSlug]
  disabled: boolean
  label: string
  onConfirm: () => void
  selectedLabel?: ReactNode
}) {
  const { t } = useI18n()
  return (
    <div className="mt-1 grid gap-2 rounded-lg border border-white/12 bg-black/18 p-2">
      <p className="flex min-h-5 flex-wrap items-center gap-1.5 text-xs font-black text-[#fff8e8]/65">
        {selectedLabel
          ? (
              <>
                {t('social.selectedChoice', { name: '' })}
                {selectedLabel}
              </>
            )
          : t('social.selectBeforeConfirm')}
      </p>
      <button className={socialButton(config, true)} disabled={disabled} type="button" onClick={onConfirm}>
        <Vote className="size-4" />
        {label}
      </button>
    </div>
  )
}

function SubmittedNotice({ config, label }: { config: typeof GAME_COPY[SocialGameSlug], label: string }) {
  return (
    <div className={cn('rounded-lg px-3 py-2 text-sm font-black', config.primary)}>
      {label}
    </div>
  )
}

function DeadPlayerPanel({
  actions,
  config,
  game,
  isHost,
  room,
  setMessage,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  isHost: boolean
  room: SocialRoom
  setMessage: (message: string) => void
}) {
  const { t } = useI18n()
  const canHostStartVote = game === 'werewolf' && room.phase === 'day' && isHost

  return (
    <Panel config={config}>
      <Skull className="size-6 text-[#ff9aa8]" />
      <h2 className="text-xl font-black">{t('social.outSpectatorTitle')}</h2>
      <p className="text-sm leading-6 text-[#fff8e8]/76">{t('social.outSpectatorHint')}</p>
      {canHostStartVote && (
        <div className="grid gap-2 rounded-lg border border-white/12 bg-black/20 p-2">
          <p className="text-xs font-bold leading-5 text-[#fff8e8]/64">{t('social.outHostControlHint')}</p>
          <button className={socialButton(config, true)} type="button" onClick={() => void actions.advanceDay().then(() => setMessage(t('werewolf.voteStarted')))}>
            <Vote className="size-4" />
            {t('werewolf.startVote')}
          </button>
        </div>
      )}
    </Panel>
  )
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
  return (
    <div className={cn('grid content-start gap-3 overflow-auto pr-1', compact ? 'sm:grid-cols-2 xl:grid-cols-3' : 'sm:grid-cols-2')}>
      {room.players.map(player => (
        <SocialPlayerCard
          key={player.id}
          accent={playerAccent(room, player.id)}
          actions={actions}
          config={config}
          isHost={isHost}
          llmModel={llmModel}
          player={player}
          room={room}
        />
      ))}
    </div>
  )
}

function SocialPlayerCard({
  accent,
  actions,
  config,
  isHost,
  llmModel,
  player,
  room,
}: {
  accent: PlayerAccent
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  llmModel: string
  player: SocialPlayer
  room: SocialRoom
}) {
  const { t } = useI18n()
  const isSelf = player.id === room.youPlayerId
  const playerStatusLabel = player.ai ? (llmModel ? `AI: ${llmModel}` : t('common.ai')) : player.roomRole === 'host' ? t('common.host') : player.connected ? t('common.online') : t('common.offline')
  const canRemove = isHost && room.phase === 'lobby' && player.roomRole !== 'host'
  const canNote = Boolean(room.youPlayerId && !isSelf)
  const cardAccent = player.alive
    ? { border: accent.border, glow: accent.soft, side: accent.solid }
    : { border: 'rgba(148,163,184,0.72)', glow: 'rgba(15,23,42,0.28)', side: '#94a3b8' }

  return (
    <article
      className={cn('relative grid gap-2 rounded-lg border bg-black/26 p-3', !player.alive && 'bg-slate-950/34')}
      style={{
        borderColor: cardAccent.border,
        boxShadow: `inset 3px 0 0 ${cardAccent.side}, 0 16px 34px rgba(0,0,0,0.18), 0 0 26px ${cardAccent.glow}`,
      }}
    >
      {!player.alive && (
        <span aria-hidden className="pointer-events-none absolute inset-0 rounded-lg bg-[#7f1d1d]/24" />
      )}
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2 pt-0.5">
          <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
          <span
            className="inline-flex min-h-5 shrink-0 items-center rounded-md border px-1.5 text-[0.68rem] font-black leading-none"
            style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
          >
            {playerNumberLabel(room, player.id)}
          </span>
          <strong className="min-w-0 truncate text-base" style={{ color: accent.text, textShadow: `0 0 16px ${accent.soft}` }}>{player.name}</strong>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <span className="inline-flex min-h-6 max-w-32 items-center truncate rounded-full bg-[#fff8e8] px-2 text-xs font-black leading-none text-[#171411]" title={playerStatusLabel}>
            {playerStatusLabel}
          </span>
          {canRemove && (
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

      {isSelf && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-white/10 bg-white/6 p-1">
          <PlayerNameEditor buttonClassName="text-[#fff8e8]" className="min-w-[180px]" name={player.name} onSave={actions.renamePlayer} />
          <SpeechButton onSend={actions.say} />
        </div>
      )}

      <div className="relative h-0">
        <SpeechBubble
          align="center"
          anchorClassName="left-1/2 -translate-x-1/2"
          placement="below"
          speech={latestSpeechForPlayer(room.speeches, player.id)}
          speakerName={player.name}
          speakerStyle={{ color: accent.ink }}
        />
      </div>

      <SocialPlayerBadges player={player} />

      {canNote && (
        <PlayerNoteEditor className="mt-1" note={player.note} onSave={note => actions.updatePlayerNote(player.id, note)} />
      )}
    </article>
  )
}

function SocialPlayerBadges({ player }: { player: SocialPlayer }) {
  const { t } = useI18n()
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className={cn('inline-flex min-h-6 items-center rounded-full px-2 text-xs font-black leading-none', player.alive ? 'bg-emerald-200 text-emerald-950' : 'bg-zinc-300 text-zinc-950')}>
        {player.alive ? t('social.alive') : t('social.out')}
      </span>
      {player.visibleToYou && player.role
        ? <RoleBadge dead={!player.alive} role={player.role} />
        : <span className="inline-flex min-h-6 items-center rounded-full bg-white/12 px-2 text-xs font-black leading-none">{t('social.hiddenRole')}</span>}
    </div>
  )
}

function TableLogLine({ entry, room }: { entry: SocialRoom['log'][number], room: SocialRoom }) {
  const speaker = logSpeaker(room, entry.text)

  if (!speaker) {
    return <p className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/76">{entry.text}</p>
  }

  const accent = playerAccent(room, speaker.id)
  const rest = entry.text.slice(speaker.name.length)

  return (
    <p className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/78">
      <span
        className="mr-1.5 inline-flex min-h-6 items-center rounded-md border px-2 align-baseline text-xs font-black leading-none"
        style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
      >
        {playerNumberLabel(room, speaker.id)}
        <span className="mx-1 opacity-65">·</span>
        {speaker.name}
      </span>
      <span>{rest}</span>
    </p>
  )
}

function logSpeaker(room: SocialRoom, text: string) {
  return [...room.players]
    .sort((left, right) => right.name.length - left.name.length)
    .find(player => text === player.name
      || text.startsWith(`${player.name} `)
      || text.startsWith(`${player.name}:`)
      || text.startsWith(`${player.name}：`))
}

function playerAccent(room: SocialRoom, playerId?: string): PlayerAccent {
  const playerIndex = playerOrderIndex(room, playerId)
  return PLAYER_COLOR_PALETTE[playerIndex % PLAYER_COLOR_PALETTE.length]
}

function playerNumberLabel(room: SocialRoom, playerId?: string) {
  return `${playerOrderIndex(room, playerId) + 1}号`
}

function playerOrderIndex(room: SocialRoom, playerId?: string) {
  return Math.max(0, room.players.findIndex(player => player.id === playerId))
}

function PlayerRefLabel({ className, player, room }: { className?: string, player: SocialPlayer, room: SocialRoom }) {
  const accent = playerAccent(room, player.id)
  return (
    <span className={cn('inline-flex min-w-0 items-center gap-1.5 align-middle', !player.alive && 'opacity-70', className)}>
      <span
        className="inline-flex min-h-5 shrink-0 items-center rounded-md border px-1.5 text-[0.68rem] font-black leading-none"
        style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
      >
        {playerNumberLabel(room, player.id)}
      </span>
      <span className="min-w-0 truncate font-black" style={{ color: accent.text, textShadow: `0 0 14px ${accent.soft}` }}>
        {player.name}
      </span>
    </span>
  )
}

function SelfIntel({ config, game, room, you }: { config: typeof GAME_COPY[SocialGameSlug], game: SocialGameSlug, room: SocialRoom, you?: SocialPlayer }) {
  const { t } = useI18n()
  const visiblePlayers = room.players.filter(player => player.visibleToYou && player.id !== you?.id && player.role)
  const seerChecks = Object.entries(room.werewolf.seerChecks ?? {})
  const yourWord = undercoverWord(room, you)
  const shouldCenterIntel = game === 'undercover'
  return (
    <section className={cn('rounded-lg border p-4', config.panel, shouldCenterIntel && 'text-center')}>
      <h2 className={cn('text-xl font-black', config.accent)}>{t('social.yourIntel')}</h2>
      <div className={cn('mt-3 flex flex-wrap gap-2', shouldCenterIntel && 'justify-center')}>
        {you?.role ? <RoleBadge dead={you.alive === false} role={you.role} size="large" /> : <span className="inline-flex min-h-8 items-center rounded-full bg-white/12 px-3 text-sm font-black leading-none">{t('social.hiddenRole')}</span>}
        {you?.alive === false && (
          <span className="inline-flex min-h-8 items-center rounded-full bg-[#3f1720] px-3 text-sm font-black leading-none text-[#ffd6df] ring-1 ring-[#ff9aa8]/45">
            {t('social.out')}
          </span>
        )}
      </div>
      <p className="mt-3 text-sm leading-6 text-[#fff8e8]/72">
        {rolePlayHint(game, you)}
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
              <PlayerRefLabel player={player} room={room} />
              <RoleBadge role={player.role!} />
            </span>
          ))}
        </div>
      )}
      {game === 'werewolf' && seerChecks.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {seerChecks.map(([playerId, alignment]) => {
            const checkedPlayer = room.players.find(player => player.id === playerId)
            return (
              <span key={playerId} className={cn('inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-black', alignmentClass(alignment))}>
                {checkedPlayer ? <PlayerRefLabel player={checkedPlayer} room={room} /> : t('common.player')}
                {' · '}
                {t(`social.alignments.${alignment}`)}
              </span>
            )
          })}
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

function rolePlayHint(game: SocialGameSlug, player?: SocialPlayer) {
  if (!player?.role) {
    return '先观察局势，等身份揭晓后按你的阵营目标行动。'
  }

  if (game === 'werewolf') {
    return werewolfRoleHint(player.role)
  }
  if (game === 'avalon') {
    return avalonRoleHint(player.role)
  }
  return undercoverRoleHint(player.role)
}

function werewolfRoleHint(role: SocialRole) {
  switch (role) {
    case 'werewolf':
      return '你是狼人。夜里和狼队一起找关键好人下手，白天要伪装成好人、带偏投票。'
    case 'seer':
      return '你是预言家。每晚查验一名玩家阵营，白天用发言把信息传出去，同时别太早暴露自己。'
    case 'witch':
      return '你是女巫。你有一瓶解药和一瓶毒药，各只能用一次；救人、毒人都要看局势。'
    case 'guard':
      return '你是守卫。每晚保护一名玩家，尽量预判狼刀目标，避免连续死守同一个思路。'
    case 'hunter':
      return '你是猎人。正常白天找狼；如果死亡触发开枪机会，尽量带走你最怀疑的玩家。'
    case 'idiot':
      return '你是白痴。首次被放逐会翻牌免死，但之后不能投票；发言时可以更大胆地找狼。'
    default:
      return '你是村民。没有夜间技能，白天主要靠发言、投票和观察行为找出狼人。'
  }
}

function avalonRoleHint(role: SocialRole) {
  switch (role) {
    case 'merlin':
      return '你是梅林。你知道邪恶阵营的大致身份，要引导正义获胜，但不能让刺客看出你是梅林。'
    case 'assassin':
      return '你是刺客。和邪恶阵营一起破坏任务；如果正义完成三次任务，你还可以刺杀梅林翻盘。'
    case 'minion':
      return '你是爪牙。帮邪恶阵营混入任务队伍、制造怀疑，必要时掩护刺客判断梅林。'
    default:
      return '你是忠臣。你不知道谁好谁坏，要通过组队、投票和任务结果判断阵营。'
  }
}

function undercoverRoleHint(role: SocialRole) {
  switch (role) {
    case 'undercover':
      return '你是卧底。你的词和多数人不一样，描述时要贴近大家但别暴露差异。'
    case 'blank':
      return '你是白板。你没有词，要从别人描述里推断主题，再用模糊但可信的线索混进去。'
    default:
      return '你是平民。描述自己的词但不要直接说出来，同时观察谁的描述和大家不太一样。'
  }
}

function RoleList({ game }: { game: SocialGameSlug }) {
  return (
    <div className="flex flex-wrap gap-2">
      {ROLES_BY_GAME[game].map(role => <RoleBadge key={role} role={role} />)}
    </div>
  )
}

function RoleBadge({ dead = false, role, size = 'normal' }: { dead?: boolean, role: SocialRole, size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <span className={cn('inline-flex items-center rounded-full font-black leading-none', size === 'large' ? 'min-h-8 px-3 text-sm' : 'min-h-6 px-2 text-xs', dead ? deadRoleClass() : alignmentClass(ROLE_ALIGNMENT[role]))}>
      {roleLabel(role, t)}
    </span>
  )
}

function deadRoleClass() {
  return 'bg-[#242933] text-[#cbd5e1] ring-1 ring-[#ff9aa8]/42 shadow-[inset_0_-2px_0_rgba(255,154,168,0.28)]'
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

function SocialShell({
  children,
  config,
  game,
  phase,
}: {
  children: ReactNode
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  phase?: SocialRoom['phase']
}) {
  const { t } = useI18n()
  const shellTheme = socialShellTheme(config, game, phase)
  return (
    <main className={cn('relative min-h-svh overflow-y-auto text-[#fff8e8] transition-colors duration-700 ease-in-out', shellTheme.bg)}>
      <div className={cn('pointer-events-none absolute inset-0 transition-[background,opacity] duration-700 ease-in-out', shellTheme.overlay)} />
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

function socialShellTheme(config: typeof GAME_COPY[SocialGameSlug], game: SocialGameSlug, phase?: SocialRoom['phase']) {
  if (game !== 'werewolf') {
    return {
      bg: config.bg,
      overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(255,209,102,0.12),transparent_28%),radial-gradient(circle_at_88%_12%,rgba(115,171,191,0.16),transparent_26%)]',
    }
  }

  if (phase === 'night') {
    return {
      bg: 'bg-[#050912]',
      overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(96,165,250,0.22),transparent_30%),radial-gradient(circle_at_88%_12%,rgba(45,212,191,0.10),transparent_24%),linear-gradient(180deg,rgba(0,0,0,0.10),rgba(0,0,0,0.34))]',
    }
  }

  if (phase === 'day' || phase === 'vote' || phase === 'hunter') {
    return {
      bg: 'bg-[#31465a]',
      overlay: 'bg-[radial-gradient(circle_at_18%_0%,rgba(255,209,102,0.34),transparent_30%),radial-gradient(circle_at_84%_10%,rgba(125,211,252,0.24),transparent_28%),linear-gradient(180deg,rgba(255,248,232,0.06),rgba(8,16,24,0.18))]',
    }
  }

  return {
    bg: config.bg,
    overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(255,209,102,0.14),transparent_28%),radial-gradient(circle_at_88%_12%,rgba(115,171,191,0.18),transparent_26%)]',
  }
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
