import type { FormEvent, ReactNode } from 'react'
import type { AILevel } from '@/games/ai'
import { ArrowLeft, Bot, Copy, DoorOpen, Plus, Sparkles } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { getAICapabilities, getAILevelLabel } from '@/games/ai'
import { AILevelBadgeSelect } from '@/games/AILevelBadgeSelect'
import { AILevelPicker } from '@/games/AILevelPicker'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { latestSpeechForPlayer } from '@/games/speech'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { createUnoRoom } from './online'
import { UnoPage } from './UnoPage'
import { useUnoRoom } from './useUnoRoom'

interface UnoRoomGateProps {
  roomId?: string
}

const UNO_VARIANTS = [
  'classic',
  'stacking',
  'party',
  'all-wild',
  'flip',
  'no-mercy',
  'wild-plus',
]

const UNO_THEMES = [
  'classic',
  'neon',
  'anime-collab',
]

export function UnoRoomGate({ roomId }: UnoRoomGateProps) {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { locale, t, ta } = useI18n()
  const { actions, error, isLoading, room } = useUnoRoom(roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('uno.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [aiLevel, setAILevel] = useState<AILevel>('normal')
  const [llmEnabled, setLLMEnabled] = useState(false)
  const [variantKey, setVariantKey] = useState('classic')
  const [themeKey, setThemeKey] = useState('classic')
  const isHost = Boolean(user && room?.hostUserId === user.id)

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
      if (!capabilities.llmEnabled && aiLevel === 'ai') {
        setAILevel('normal')
      }
    })
  }, [aiLevel])

  if (roomId && room && room.phase !== 'lobby') {
    return <UnoPage roomId={roomId} />
  }

  async function createRoom() {
    setMessage(t('room.defaultMessage'))
    try {
      const nextRoom = await createUnoRoom({ themeKey, variantKey })

      navigate(`/games/uno?room=${nextRoom.id}`)
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

    navigate(`/games/uno?room=${encodeURIComponent(normalizedCode)}`)
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
      await actions.addAI(aiLevel)
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
      <UnoShell>
        <section className="grid min-h-0 gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="uno-panel relative grid content-end overflow-hidden p-5 sm:p-6">
            <div className="pointer-events-none absolute -right-8 top-2 rotate-[-12deg] text-[clamp(120px,24vw,270px)] font-black leading-none text-white/20">
              UNO
            </div>
            <div className="relative max-w-xl">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">{t('uno.lobbyTitle')}</h2>
              <p className="mt-3 text-sm leading-7 text-[#fff8e8]/80 sm:text-base">{t('uno.lobbyDescription')}</p>
            </div>
          </div>

          <form className="uno-panel grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-4 p-5 sm:p-6" onSubmit={joinRoom}>
            <div className="grid gap-4">
              <h2 className="text-2xl font-black tracking-normal">{t('uno.enterTitle')}</h2>
              <button className="uno-button uno-button-primary w-full" type="button" onClick={createRoom}>
                <Plus className="size-4" />
                {t('common.createAndEnter')}
              </button>
            </div>

            <div className="grid min-h-0 content-start gap-4 overflow-y-auto pr-1">
              <OptionGroup label={t('uno.variantType')} options={UNO_VARIANTS.map(key => unoVariantOption(key, t))} value={variantKey} onChange={setVariantKey} />
              <OptionGroup label={t('uno.themeType')} options={UNO_THEMES.map(key => unoThemeOption(key, t))} value={themeKey} onChange={setThemeKey} />
            </div>

            <div className="grid gap-3">
              <label className="grid gap-2 text-sm font-black" htmlFor="uno-room-code">
                {t('common.roomCode')}
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
                {t('common.joinRoom')}
              </button>
              <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{message}</p>
            </div>
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
                {t('common.room')}
                {' '}
                {roomId}
              </h2>
              <p className="mt-1 text-xs font-bold text-[#fff8e8]/60">
                {variantLabel(room?.variantKey, t)}
                {' / '}
                {themeLabel(room?.themeKey, t)}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className="uno-button" type="button" onClick={copyLink}>
                <Copy className="size-4" />
                {t('common.copyLink')}
              </button>
              <div className="grid min-w-[250px] grid-cols-[minmax(120px,1fr)_auto] items-end gap-2">
                <AILevelPicker level={aiLevel} llmEnabled={llmEnabled} onChange={setAILevel} />
                <button
                  className={cn('uno-button', pendingAI && 'loading')}
                  disabled={pendingAI || !isHost || !room || room.players.length >= 10}
                  type="button"
                  onClick={addAIPlayer}
                >
                  <Bot className="size-4" />
                  {pendingAI ? t('room.addingAI') : `${t('room.addAI')} (${getAILevelLabel(aiLevel, locale)})`}
                </button>
              </div>
            </div>
          </div>

          <div className="grid content-start gap-3 overflow-auto pr-1 sm:grid-cols-2">
            {isLoading && <p className="text-sm font-bold text-[#fff8e8]/70">{t('room.connecting')}</p>}
            {room?.players.map(player => (
              <article key={player.id} className="rounded-lg border border-white/25 bg-[#090807]/50 p-4 shadow-[0_16px_34px_rgba(0,0,0,0.2)]">
                <div className="flex items-center justify-between gap-3">
                  <strong className="truncate text-lg">{player.name}</strong>
                  <div className="flex shrink-0 items-center gap-2">
                    {player.userId === user?.id && <SpeechButton onSend={actions.say} />}
                    {player.ai
                      ? (
                          <AILevelBadgeSelect
                            disabled={!isHost || room.phase !== 'lobby'}
                            level={player.ai.level}
                            llmEnabled={llmEnabled}
                            onChange={level => void actions.updateAI(player.id, level)}
                          />
                        )
                      : (
                          <span className="rounded-full bg-[#fff8e8] px-2 py-0.5 text-xs font-black text-[#171411]">
                            {player.role === 'host' ? t('common.host') : player.connected ? t('common.online') : t('common.offline')}
                          </span>
                        )}
                  </div>
                </div>
                <SpeechBubble className="mt-3" text={latestSpeechForPlayer(room.speeches, player.id)?.text} />
                <p className="mt-2 min-h-10 text-sm leading-6 text-[#fff8e8]/72">
                  {player.ai?.personality ?? (player.kind === 'guest' ? t('xiangqi.guestReady') : t('xiangqi.oidcReady'))}
                </p>
              </article>
            ))}
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <button className="uno-button uno-button-primary sm:w-40" disabled={!isHost || !room || room.players.length < 2} type="button" onClick={startGame}>
              {t('common.startGame')}
            </button>
            <p className="text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
          </div>
        </div>

        <aside className="uno-panel grid content-start gap-3 p-5">
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            {t('uno.aiSystem')}
          </h2>
          {ta('uno.rules').map(rule => <RuleLine key={rule}>{rule}</RuleLine>)}
        </aside>
      </section>
    </UnoShell>
  )
}

function UnoShell({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  return (
    <main className="h-svh overflow-hidden bg-[#15110e] text-[#fff8e8]">
      <div className="mx-auto grid h-full w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
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
            {t('common.backToLobby')}
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

function variantLabel(key: string | undefined, t: (key: string) => string) {
  return key ? unoVariantOption(key, t).name : t('uno.classicVariant')
}

function themeLabel(key: string | undefined, t: (key: string) => string) {
  return key ? unoThemeOption(key, t).name : t('uno.classicTheme')
}

function unoVariantOption(key: string, t: (key: string) => string) {
  const normalizedKey = key.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase())
  return {
    description: t(`uno.variants.${normalizedKey}.description`),
    key,
    name: t(`uno.variants.${normalizedKey}.name`),
  }
}

function unoThemeOption(key: string, t: (key: string) => string) {
  const normalizedKey = key.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase())
  return {
    description: t(`uno.themes.${normalizedKey}.description`),
    key,
    name: t(`uno.themes.${normalizedKey}.name`),
  }
}
