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
import { createXiangqiRoom, getCurrentXiangqiRoom } from './online'
import { useXiangqiRoom } from './useXiangqiRoom'
import { XiangqiPage } from './XiangqiPage'

interface XiangqiRoomGateProps {
  roomId?: string
}

export function XiangqiRoomGate({ roomId }: XiangqiRoomGateProps) {
  const navigate = useNavigate()
  const { locale, t, ta } = useI18n()
  const { actions, error, isLoading, room } = useXiangqiRoom(roomId)
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState(() => t('room.defaultMessage'))
  const [pendingAI, setPendingAI] = useState(false)
  const [aiLevel, setAILevel] = useState<AILevel>('normal')
  const [llmEnabled, setLLMEnabled] = useState(false)
  const isHost = Boolean(room?.hostPlayerId && room.hostPlayerId === room.youPlayerId)
  const loadCurrentRoom = useCallback(() => getCurrentXiangqiRoom(), [])
  const { currentRoom } = useCurrentRoom(!roomId, loadCurrentRoom)

  useEffect(() => {
    let active = true
    void getAICapabilities().then((capabilities) => {
      if (!active) {
        return
      }
      setLLMEnabled(capabilities.llmEnabled)
      setAILevel(current => current === 'ai' && !capabilities.llmEnabled ? 'normal' : current)
    })
    return () => {
      active = false
    }
  }, [])

  if (roomId && room && room.phase !== 'lobby') {
    return <XiangqiPage roomId={roomId} />
  }

  async function createRoom() {
    setMessage(t('room.defaultMessage'))
    try {
      const nextRoom = await createXiangqiRoom()

      navigate(`/games/xiangqi?room=${nextRoom.id}`)
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

    navigate(`/games/xiangqi?room=${encodeURIComponent(normalizedCode)}`)
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
    navigate(`/games/xiangqi?room=${encodeURIComponent(currentRoom.id)}`)
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
      <XiangqiShell>
        <section className="grid min-h-[min(560px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="xiangqi-panel relative grid content-end overflow-hidden p-5 sm:p-6">
            <div className="pointer-events-none absolute right-3 top-2 text-[clamp(120px,22vw,250px)] font-black leading-none text-[#fff8e8]/10">象</div>
            <div className="relative max-w-xl">
              <h2 className="text-3xl font-black tracking-normal sm:text-4xl">{t('xiangqi.lobbyTitle')}</h2>
              <p className="mt-3 text-sm leading-7 text-[#fff8e8]/78 sm:text-base">
                {t('xiangqi.lobbyDescription')}
              </p>
            </div>
          </div>

          <form className="xiangqi-panel grid content-start gap-4 p-5 sm:p-6" onSubmit={joinRoom}>
            <h2 className="text-2xl font-black tracking-normal">{t('xiangqi.enterTitle')}</h2>
            <button className="xiangqi-button xiangqi-button-primary w-full" type="button" onClick={createRoom}>
              <Plus className="size-4" />
              {t('common.createAndEnter')}
            </button>
            {currentRoom && (
              <ContinueRoomEntry
                buttonClassName="xiangqi-button w-full"
                className="border-[#fff8e8]/22 bg-[#10100d]/52 text-[#fff8e8]"
                room={currentRoom}
                onEnter={enterCurrentRoom}
              />
            )}
            <label className="grid gap-2 text-sm font-black" htmlFor="xiangqi-room-code">
              {t('common.roomCode')}
              <input
                id="xiangqi-room-code"
                className="min-h-11 rounded-lg border border-[#fff8e8]/35 bg-[#10100d]/55 px-3 uppercase text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#f2d59a]"
                placeholder="XQ42"
                value={joinCode}
                onChange={event => setJoinCode(event.target.value)}
              />
            </label>
            <button className="xiangqi-button w-full" type="submit">
              <DoorOpen className="size-4" />
              {t('common.joinRoom')}
            </button>
            <p className="min-h-6 text-sm font-bold text-[#fff8e8]/75">{message}</p>
          </form>
        </section>
      </XiangqiShell>
    )
  }

  return (
    <XiangqiShell>
      <section className="grid min-h-[min(600px,calc(100svh-150px))] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="xiangqi-panel grid grid-rows-[auto_minmax(0,1fr)_auto] gap-4 p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black text-[#f2d59a]/80">ROOM</p>
              <h2 className="text-2xl font-black tracking-normal">
                {t('common.room')}
                {' '}
                {roomId}
              </h2>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className="xiangqi-button" type="button" onClick={copyLink}>
                <Copy className="size-4" />
                {t('common.copyLink')}
              </button>
              <div className="grid min-w-[250px] grid-cols-[minmax(120px,1fr)_auto_auto] items-end gap-2">
                <AILevelPicker level={aiLevel} llmEnabled={llmEnabled} palette="xiangqi" onChange={setAILevel} />
                <button
                  className={cn('xiangqi-button', pendingAI && 'loading')}
                  disabled={pendingAI || !isHost || !room || room.players.length >= 2}
                  type="button"
                  onClick={addAIPlayer}
                >
                  <Bot className="size-4" />
                  {pendingAI ? t('room.addingAI') : `${t('room.addAI')} (${getAILevelLabel(aiLevel, locale)})`}
                </button>
                <button className="xiangqi-button xiangqi-button-primary" disabled={!isHost || !room || room.players.length < 2} type="button" onClick={startGame}>
                  {t('common.startGame')}
                </button>
              </div>
            </div>
          </div>

          <div className="grid content-start gap-3 overflow-auto pr-1 sm:grid-cols-2">
            {isLoading && <p className="text-sm font-bold text-[#fff8e8]/70">{t('room.connecting')}</p>}
            {room?.players.map(player => (
              <article key={player.id} className="relative rounded-lg border border-[#fff8e8]/18 bg-[#10100d]/58 p-4 shadow-[0_16px_34px_rgba(0,0,0,0.18)]">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <PlayerStatusDot connected={player.connected} disconnectedAt={player.disconnectedAt} />
                    <strong className="truncate text-lg">{player.name}</strong>
                    {player.id === room.youPlayerId && <PlayerNameEditor buttonClassName="text-[#fff8e8]" name={player.name} onSave={actions.renamePlayer} />}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {player.id === room.youPlayerId && <SpeechButton palette="xiangqi" onSend={actions.say} />}
                    {player.ai
                      ? (
                          <AILevelBadgeSelect
                            disabled={!isHost || room.phase !== 'lobby'}
                            level={player.ai.level}
                            llmEnabled={llmEnabled}
                            palette="xiangqi"
                            onChange={level => void actions.updateAI(player.id, level)}
                          />
                        )
                      : (
                          <span className="rounded-full bg-[#fff8e8] px-2 py-0.5 text-xs font-black text-[#202018]">
                            {player.role === 'host' ? t('common.host') : player.connected ? t('common.online') : t('common.offline')}
                          </span>
                        )}
                    {isHost && room.phase === 'lobby' && player.role !== 'host' && (
                      <button
                        aria-label={t('common.removePlayer')}
                        className="xiangqi-button min-h-7 px-2"
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
                <p className="mt-2 min-h-10 text-sm leading-6 text-[#fff8e8]/72">
                  {player.ai?.personality ?? (player.kind === 'guest' ? t('xiangqi.guestReady') : t('xiangqi.oidcReady'))}
                </p>
              </article>
            ))}
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <p className="text-sm font-bold text-[#fff8e8]/75">{error ?? message}</p>
          </div>
        </div>

        <aside className="xiangqi-panel grid content-start gap-3 p-5">
          <h2 className="flex items-center gap-2 text-xl font-black">
            <Sparkles className="size-5" />
            {t('xiangqi.rulesTitle')}
          </h2>
          {ta('xiangqi.rules').map(rule => <RuleLine key={rule}>{rule}</RuleLine>)}
        </aside>
      </section>
    </XiangqiShell>
  )
}

function XiangqiShell({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  return (
    <main className="min-h-svh overflow-y-auto bg-[#202018] text-[#fff8e8]">
      <div className="mx-auto grid min-h-svh w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3">
        <header className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="mb-1 text-xs font-black tracking-normal text-[#f2d59a]/80">ONLINE XIANGQI ROOM</p>
            <h1 className="text-[clamp(40px,8vw,84px)] font-black leading-[0.82] tracking-normal text-[#fff8e8]">
              {t('xiangqi.title')}
            </h1>
          </div>
          <Link
            className="inline-grid min-h-10 place-items-center rounded-full border border-[#fff8e8]/36 bg-[#10100d]/55 px-4 text-sm font-bold text-[#fff8e8] transition hover:bg-[#10100d]/72"
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
  return <p className="rounded-lg bg-[#10100d]/58 px-3 py-2 text-sm leading-6 text-[#fff8e8]/78">{children}</p>
}
