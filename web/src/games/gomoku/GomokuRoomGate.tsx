import type { FormEvent, ReactNode } from 'react'
import type { AILevel } from '@/games/ai'
import { ArrowLeft, Bot, Copy, DoorOpen, Plus, Sparkles, UserMinus } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { getAICapabilities, getAILevelLabel } from '@/games/ai'
import { AILevelBadgeSelect } from '@/games/AILevelBadgeSelect'
import { AILevelPicker } from '@/games/AILevelPicker'
import { ContinueRoomEntry } from '@/games/ContinueRoomEntry'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { useCurrentRoom } from '@/games/useCurrentRoom'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { GomokuPage } from './GomokuPage'
import { createGomokuRoom, getCurrentGomokuRoom } from './online'
import { useGomokuRoom } from './useGomokuRoom'

interface GomokuRoomGateProps {
  roomId?: string
}

export function GomokuRoomGate({ roomId }: GomokuRoomGateProps) {
  const navigate = useNavigate()
  const { locale, t, ta } = useI18n()
  const { actions, error, isLoading, room } = useGomokuRoom(roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('room.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [aiLevel, setAILevel] = useState<AILevel>('normal')
  const [llmEnabled, setLLMEnabled] = useState(false)
  const [llmModel, setLLMModel] = useState('')
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const loadCurrentRoom = useCallback(() => getCurrentGomokuRoom(), [])
  const { currentRoom } = useCurrentRoom(!roomId, loadCurrentRoom)

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
      setLLMModel(capabilities.model ?? '')
      if (!capabilities.llmEnabled && aiLevel === 'ai') {
        setAILevel('normal')
      }
    })
  }, [aiLevel])

  if (roomId && room && room.phase !== 'lobby') {
    return <GomokuPage roomId={roomId} />
  }

  async function createRoom() {
    setMessage(t('room.defaultMessage'))
    try {
      const nextRoom = await createGomokuRoom()

      navigate(`/games/gomoku?room=${nextRoom.id}`)
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

    navigate(`/games/gomoku?room=${encodeURIComponent(normalizedCode)}`)
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
    navigate(`/games/gomoku?room=${encodeURIComponent(currentRoom.id)}`)
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
      <GomokuShell>
        <section className="grid min-h-[min(560px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="gomoku-panel relative grid content-end overflow-hidden p-5 sm:p-6">
            <div className="pointer-events-none absolute right-5 top-5 grid size-44 grid-cols-5 gap-2 opacity-20">
              {Array.from({ length: 25 }, (_, index) => (
                <span key={index} className={cn('rounded-full', index % 2 === 0 ? 'bg-[#0d1512]' : 'bg-[#f4f0e4]')} />
              ))}
            </div>
            <div className="relative max-w-xl">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">{t('gomoku.lobbyTitle')}</h2>
              <p className="mt-3 text-sm leading-7 text-[#f4f0e4]/78 sm:text-base">{t('gomoku.lobbyDescription')}</p>
            </div>
          </div>

          <form className="gomoku-panel grid content-start gap-4 p-5 sm:p-6" onSubmit={joinRoom}>
            <h2 className="text-2xl font-black tracking-normal">{t('gomoku.enterTitle')}</h2>
            <button className="gomoku-button gomoku-button-primary w-full" type="button" onClick={createRoom}>
              <Plus className="size-4" />
              {t('common.createAndEnter')}
            </button>
            {currentRoom && (
              <ContinueRoomEntry
                buttonClassName="gomoku-button w-full"
                className="border-white/22 bg-[#0b1110]/52 text-[#f4f0e4]"
                room={currentRoom}
                onEnter={enterCurrentRoom}
              />
            )}
            <label className="grid gap-2 text-sm font-black" htmlFor="gomoku-room-code">
              {t('common.roomCode')}
              <input
                id="gomoku-room-code"
                className="min-h-11 rounded-lg border border-white/35 bg-[#101714]/55 px-3 uppercase text-[#f4f0e4] outline-none focus:ring-2 focus:ring-[#f4f0e4]"
                placeholder="GMK42"
                value={joinCode}
                onChange={event => setJoinCode(event.target.value)}
              />
            </label>
            <button className="gomoku-button w-full" type="submit">
              <DoorOpen className="size-4" />
              {t('common.joinRoom')}
            </button>
            <p className="min-h-6 text-sm font-bold text-[#f4f0e4]/75">{message}</p>
          </form>
        </section>
      </GomokuShell>
    )
  }

  return (
    <GomokuShell>
      <section className="grid min-h-[min(600px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="gomoku-panel grid grid-rows-[auto_minmax(0,1fr)_auto] gap-4 p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black text-[#f4f0e4]/65">ROOM</p>
              <h2 className="text-2xl font-black tracking-normal">
                {t('common.room')}
                {' '}
                {roomId}
              </h2>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className="gomoku-button" type="button" onClick={copyLink}>
                <Copy className="size-4" />
                {t('common.copyLink')}
              </button>
              <div className="grid min-w-[250px] grid-cols-[minmax(120px,1fr)_auto_auto] items-end gap-2">
                <AILevelPicker level={aiLevel} llmEnabled={llmEnabled} llmModel={llmModel} palette="gomoku" onChange={setAILevel} />
                <button
                  className={cn('gomoku-button', pendingAI && 'loading')}
                  disabled={pendingAI || !isHost || !room || room.players.length >= 2}
                  type="button"
                  onClick={addAIPlayer}
                >
                  <Bot className="size-4" />
                  {pendingAI ? t('room.addingAI') : `${t('room.addAI')} (${getAILevelLabel(aiLevel, locale)})`}
                </button>
                <button className="gomoku-button gomoku-button-primary" disabled={!isHost || !room || room.players.length < 2} type="button" onClick={startGame}>
                  {t('common.startGame')}
                </button>
              </div>
            </div>
          </div>

          <div className="grid content-start gap-3 overflow-auto pr-1 sm:grid-cols-2">
            {isLoading && <p className="text-sm font-bold text-[#f4f0e4]/70">{t('room.connecting')}</p>}
            {room?.players.map(player => (
              <article key={player.id} className="relative rounded-lg border border-white/22 bg-[#0b1110]/58 p-4 shadow-[0_16px_34px_rgba(0,0,0,0.18)]">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
                    <strong className="truncate text-lg">{player.name}</strong>
                    {player.id === room.youPlayerId && <PlayerNameEditor buttonClassName="text-[#f4f0e4]" name={player.name} onSave={actions.renamePlayer} />}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {player.id === room.youPlayerId && <SpeechButton palette="gomoku" onSend={actions.say} />}
                    {player.ai
                      ? (
                          <AILevelBadgeSelect
                            disabled={!isHost || room.phase !== 'lobby'}
                            level={player.ai.level}
                            llmEnabled={llmEnabled}
                            llmModel={llmModel}
                            palette="gomoku"
                            onChange={level => void actions.updateAI(player.id, level)}
                          />
                        )
                      : (
                          <span className="rounded-full bg-[#f4f0e4] px-2 py-0.5 text-xs font-black text-[#101714]">
                            {player.role === 'host' ? t('common.host') : player.connected ? t('common.online') : t('common.offline')}
                          </span>
                        )}
                    {isHost && room.phase === 'lobby' && player.role !== 'host' && (
                      <button
                        aria-label={t('common.removePlayer')}
                        className="gomoku-button min-h-7 px-2"
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
                <p className="mt-2 min-h-10 text-sm leading-6 text-[#f4f0e4]/72">
                  {player.ai?.personality ?? (player.kind === 'guest' ? t('xiangqi.guestReady') : t('xiangqi.oidcReady'))}
                </p>
              </article>
            ))}
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <p className="text-sm font-bold text-[#f4f0e4]/75">{error ?? message}</p>
          </div>
        </div>

        <aside className="gomoku-panel grid content-start gap-3 p-5">
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            {t('gomoku.rulesTitle')}
          </h2>
          {ta('gomoku.rules').map(rule => <RuleLine key={rule}>{rule}</RuleLine>)}
        </aside>
      </section>
    </GomokuShell>
  )
}

function GomokuShell({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  return (
    <main className="min-h-svh overflow-y-auto bg-[#101714] text-[#f4f0e4]">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
        <header className="flex items-end justify-between gap-4">
          <div>
            <p className="mb-1 text-xs font-black tracking-normal text-[#f4f0e4]/75">ONLINE GOMOKU BOARD</p>
            <h1 className="text-[clamp(40px,8vw,84px)] font-black leading-[0.82] tracking-normal [text-shadow:0_7px_0_rgba(0,0,0,0.32)]">
              {t('gomoku.title')}
            </h1>
          </div>
          <Link
            className="inline-grid min-h-10 place-items-center rounded-full border border-white/36 bg-[#0b1110]/55 px-4 text-sm font-bold text-[#f4f0e4] transition hover:bg-[#0b1110]/72"
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
  return <p className="rounded-lg bg-[#0b1110]/58 px-3 py-2 text-sm leading-6 text-[#f4f0e4]/78">{children}</p>
}
