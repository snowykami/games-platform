import type { ReactNode } from 'react'
import type { ClaimOption, MahjongPlayer, MahjongTile, WinResult } from './types'
import type { AILevel } from '@/games/ai'
import type { GameSpeechEntry } from '@/games/speech'
import { ArrowLeft, CircleDot, Hand, RefreshCw, ScrollText, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import { decideWithAI, getAICapabilities } from '@/games/ai'
import { AILevelBadgeSelect } from '@/games/AILevelBadgeSelect'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { latestSpeechForPlayer } from '@/games/speech'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import {
  claimOption,
  createMahjongGame,
  declareSelfDraw,
  discardTile,
  drawTile,
  formatWind,
  getCurrentPlayer,
  getHumanPlayer,
  getSelfDrawWinResult,
  playBotDiscard,
  playBotTurn,
  skipClaims,
} from './engine'
import { CHINESE_OFFICIAL_RULESET, formatTile } from './scoring'

export function MahjongPage() {
  const { t } = useI18n()
  const [state, setState] = useState(() => createMahjongGame())
  const [speeches, setSpeeches] = useState<GameSpeechEntry[]>([])
  const [aiLevel, setAILevel] = useState<AILevel>('normal')
  const [llmEnabled, setLLMEnabled] = useState(false)
  const human = getHumanPlayer(state)
  const currentPlayer = getCurrentPlayer(state)
  const winner = state.players.find(player => player.id === state.winnerId)
  const selfDrawResult = useMemo(() => getSelfDrawWinResult(state, human.id), [human.id, state])
  const humanClaims = state.claimOptions.filter(option => option.playerId === human.id)
  const lastLog = state.log.at(-1)?.text
  const isHumanTurn = state.phase === 'playing' && currentPlayer.id === human.id
  const canHumanDraw = isHumanTurn && !state.hasDrawn
  const canHumanDiscard = isHumanTurn && state.hasDrawn

  useEffect(() => {
    void getAICapabilities().then((capabilities) => {
      setLLMEnabled(capabilities.llmEnabled)
      if (!capabilities.llmEnabled && aiLevel === 'ai') {
        setAILevel('normal')
      }
    })
  }, [aiLevel])

  const runBotTurn = useCallback(async () => {
    const bot = getCurrentPlayer(state)
    if (aiLevel !== 'ai' || !state.hasDrawn) {
      setState(current => playBotTurn(current))
      return
    }

    try {
      const decision = await decideWithAI({
        actions: bot.hand.map(tile => ({
          id: `discard:${tile.id}`,
          label: t('mahjong.discardAction', { tile: formatTile(tile) }),
        })),
        game: 'mahjong',
        level: 'ai',
        personality: t('mahjong.aiPersonality'),
        playerName: bot.name,
        sessionId: `mahjong-${bot.id}`,
        state: {
          discards: state.players.map(player => ({ name: player.name, discards: player.discards.map(formatTile) })),
          hand: bot.hand.map(formatTile),
          recentSpeech: speeches.slice(-8),
          roundWind: state.roundWind,
          wallCount: state.wall.length,
        },
      })
      const tileID = decision.actionId.replace('discard:', '')
      if (bot.hand.some(tile => tile.id === tileID)) {
        recordSpeech(bot, decision.speech)
        setState(current => playBotDiscard(current, tileID))
        return
      }
    }
    catch {
      // Fallback to deterministic bot below.
    }

    setState(current => playBotTurn(current))
  }, [aiLevel, speeches, state, t])

  useEffect(() => {
    if (state.phase !== 'playing' || getCurrentPlayer(state).isHuman) {
      return undefined
    }

    const timer = window.setTimeout(() => {
      void runBotTurn()
    }, state.hasDrawn ? 620 : 460)

    return () => window.clearTimeout(timer)
  }, [runBotTurn, state])

  function handleRestart() {
    setState(createMahjongGame())
  }

  function handleDraw() {
    setState(current => drawTile(current, human.id))
  }

  function handleDiscard(tile: MahjongTile) {
    setState(current => discardTile(current, human.id, tile.id))
  }

  function handleSelfDraw() {
    setState(current => declareSelfDraw(current, human.id))
  }

  function handleClaim(option: ClaimOption) {
    setState(current => claimOption(current, option.id))
  }

  function handleSkipClaims() {
    setState(current => skipClaims(current, human.id))
  }

  function recordSpeech(player: MahjongPlayer, text: string | undefined) {
    const nextText = text?.trim()
    if (!nextText) {
      return
    }
    setSpeeches(current => [
      ...current,
      {
        id: `speech-${Date.now()}-${player.id}`,
        playerId: player.id,
        playerName: player.name,
        text: [...nextText].slice(0, 120).join(''),
        spokenAt: new Date().toISOString(),
      },
    ].slice(-18))
  }

  return (
    <main className="min-h-svh bg-[#1b342b] text-[#fff8e8]">
      <div className="mx-auto grid min-h-svh w-[min(1380px,calc(100vw-24px))] grid-rows-[auto_auto_minmax(0,1fr)_auto] gap-3 py-3">
        <header className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="mb-0.5 text-xs font-black text-[#ffd166]">CHINESE OFFICIAL MAHJONG</p>
            <h1 className="text-sm font-black leading-none tracking-normal text-[#fff8e8]/92 sm:text-base">
              {t('mahjong.title')}
            </h1>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Link className="mahjong-action" to="/">
              <ArrowLeft className="size-4" />
              {t('mahjong.back')}
            </Link>
            <button className="mahjong-action mahjong-action-primary" type="button" onClick={handleRestart}>
              <RefreshCw className="size-4" />
              {t('mahjong.restart')}
            </button>
          </div>
        </header>

        <section className="flex min-h-0 flex-wrap items-center gap-2 rounded-lg border border-[#d8b66a]/35 bg-[#10251f]/80 p-2 shadow-[0_18px_46px_rgba(0,0,0,0.22)]">
          <StatusPill icon={<ScrollText className="size-4" />}>{state.ruleset.name}</StatusPill>
          <StatusPill>
            {state.ruleset.minFan}
            {t('mahjong.minFan', { fan: state.ruleset.minFan })}
          </StatusPill>
          <StatusPill>
            {t('mahjong.roundWind')}
            {formatWind(state.roundWind)}
          </StatusPill>
          <StatusPill>
            {t('mahjong.wall')}
            {state.wall.length}
          </StatusPill>
          <StatusPill>
            {t('mahjong.turn')}
            {currentPlayer.name}
          </StatusPill>
          <label className="ml-auto inline-flex min-h-9 items-center gap-2 rounded-lg border border-[#d8b66a]/35 bg-[#fff8e8]/10 px-3 text-sm font-black text-[#fff8e8]">
            {t('mahjong.rules')}
            <select className="bg-transparent text-[#fff8e8] outline-none" value={CHINESE_OFFICIAL_RULESET.id} disabled>
              <option className="text-[#10251f]" value={CHINESE_OFFICIAL_RULESET.id}>{CHINESE_OFFICIAL_RULESET.name}</option>
            </select>
          </label>
        </section>

        <section className="grid min-h-[560px] gap-3 overflow-hidden rounded-lg border border-[#d8b66a]/45 bg-[radial-gradient(circle_at_center,rgba(255,248,232,0.12),transparent_42%),linear-gradient(135deg,#173b31,#10251f)] p-3 shadow-[inset_0_0_80px_rgba(0,0,0,0.28),0_24px_70px_rgba(0,0,0,0.25)] lg:grid-cols-[240px_minmax(0,1fr)_240px] lg:grid-rows-[150px_minmax(0,1fr)_180px]">
          <PlayerPanel aiLevel={aiLevel} className="lg:col-start-2 lg:row-start-1" currentPlayerId={currentPlayer.id} llmEnabled={llmEnabled} player={state.players[2]} speech={latestSpeechForPlayer(speeches, state.players[2].id)} onAILevelChange={setAILevel} />
          <PlayerPanel aiLevel={aiLevel} className="lg:col-start-1 lg:row-start-2" currentPlayerId={currentPlayer.id} llmEnabled={llmEnabled} player={state.players[3]} speech={latestSpeechForPlayer(speeches, state.players[3].id)} onAILevelChange={setAILevel} />

          <div className="relative grid min-h-[270px] place-items-center rounded-lg border border-[#d8b66a]/30 bg-[#0f211c]/70 p-3 lg:col-start-2 lg:row-start-2">
            <div className="absolute inset-4 rounded-lg border border-[#d8b66a]/25" />
            <div className="z-10 grid w-full max-w-xl gap-3 text-center">
              <div className="mx-auto grid size-24 place-items-center rounded-full border-4 border-[#d8b66a] bg-[#fff8e8] text-[#143128] shadow-[0_18px_50px_rgba(0,0,0,0.26)]">
                <CircleDot className="size-10" strokeWidth={2.8} />
              </div>
              <div>
                <strong className="text-xl">{state.phase === 'finished' ? (winner ? t('mahjong.win', { name: winner.name }) : t('mahjong.drawGame')) : t('mahjong.acting', { name: currentPlayer.name })}</strong>
                <p className="mt-1 text-sm font-bold text-[#fff8e8]/72">{lastLog ?? CHINESE_OFFICIAL_RULESET.description}</p>
              </div>
              <ScorePanel result={state.phase === 'finished' ? state.winResult : selfDrawResult} />
            </div>
          </div>

          <PlayerPanel aiLevel={aiLevel} className="lg:col-start-3 lg:row-start-2" currentPlayerId={currentPlayer.id} llmEnabled={llmEnabled} player={state.players[1]} speech={latestSpeechForPlayer(speeches, state.players[1].id)} onAILevelChange={setAILevel} />

          <section className="relative grid min-h-0 gap-3 rounded-lg border border-[#d8b66a]/35 bg-[#081914]/82 p-3 lg:col-span-3 lg:row-start-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <strong className="text-lg">{t('mahjong.yourHand')}</strong>
                <p className="text-sm font-bold text-[#fff8e8]/70">
                  {state.phase === 'claiming' ? t('mahjong.claimPrompt') : canHumanDiscard ? t('mahjong.discardPrompt') : canHumanDraw ? t('mahjong.drawPrompt') : t('mahjong.waitPrompt', { name: currentPlayer.name })}
                </p>
                <SpeechBubble speech={latestSpeechForPlayer(speeches, human.id)} />
              </div>
              <div className="flex flex-wrap gap-2">
                <SpeechButton palette="mahjong" onSend={text => recordSpeech(human, text)} />
                <button className="mahjong-action" disabled={!canHumanDraw} type="button" onClick={handleDraw}>
                  <Hand className="size-4" />
                  {t('mahjong.drawTile')}
                </button>
                <button className="mahjong-action mahjong-action-primary" disabled={!selfDrawResult.canWin} type="button" onClick={handleSelfDraw}>
                  <Sparkles className="size-4" />
                  {t('mahjong.selfDraw')}
                </button>
              </div>
            </div>

            {humanClaims.length > 0 && (
              <div className="flex flex-wrap items-center gap-2 rounded-lg bg-[#fff8e8]/10 p-2">
                {humanClaims.map(option => (
                  <button key={option.id} className="mahjong-action mahjong-action-primary" type="button" onClick={() => handleClaim(option)}>
                    {claimLabel(option, t)}
                  </button>
                ))}
                <button className="mahjong-action" type="button" onClick={handleSkipClaims}>{t('mahjong.skip')}</button>
              </div>
            )}

            <div className="flex min-h-24 flex-wrap items-end gap-1.5 overflow-y-auto pb-1">
              {human.hand.map(tile => (
                <button
                  key={tile.id}
                  className="p-0 transition hover:-translate-y-1 disabled:cursor-not-allowed disabled:opacity-55"
                  disabled={!canHumanDiscard}
                  title={canHumanDiscard ? t('mahjong.discardAction', { tile: formatTile(tile) }) : formatTile(tile)}
                  type="button"
                  onClick={() => handleDiscard(tile)}
                >
                  <TileView tile={tile} />
                </button>
              ))}
            </div>
          </section>
        </section>

        <p className="pb-1 text-xs font-bold text-[#fff8e8]/65">
          {CHINESE_OFFICIAL_RULESET.description}
        </p>
      </div>
    </main>
  )
}

function PlayerPanel({
  aiLevel,
  className,
  currentPlayerId,
  llmEnabled,
  onAILevelChange,
  player,
  speech,
}: {
  aiLevel: AILevel
  className?: string
  currentPlayerId: string
  llmEnabled: boolean
  onAILevelChange: (level: AILevel) => void
  player: MahjongPlayer
  speech?: GameSpeechEntry
}) {
  const { t } = useI18n()
  const isCurrent = currentPlayerId === player.id

  return (
    <article className={cn('relative grid min-h-32 content-between rounded-lg border border-[#d8b66a]/35 bg-[#081914]/72 p-3 shadow-[0_18px_40px_rgba(0,0,0,0.18)]', isCurrent && 'ring-2 ring-[#ffd166]', className)}>
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className="flex items-center gap-2">
            <strong>{player.name}</strong>
            <span className="rounded-full bg-[#ffd166] px-2 py-0.5 text-xs font-black text-[#143128]">
              {formatWind(player.wind)}
              {t('mahjong.windSuffix')}
            </span>
          </div>
          <p className="mt-1 text-xs font-bold text-[#fff8e8]/65">
            {t('mahjong.hand')}
            {' '}
            {player.hand.length}
            {' '}
            {t('mahjong.tiles')}
          </p>
        </div>
        {player.isHuman
          ? <span className="text-xs font-black text-[#ffd166]">SELF</span>
          : (
              <div className="grid justify-items-end gap-2">
                <AILevelBadgeSelect level={aiLevel} llmEnabled={llmEnabled} palette="mahjong" onChange={onAILevelChange} />
                <OpponentTiles count={player.hand.length} />
              </div>
            )}
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

function TileView({ tile }: { tile: MahjongTile }) {
  return (
    <span className="mahjong-tile">
      <span className={cn('text-[11px] font-black', tileColor(tile))}>{tileSuitLabel(tile)}</span>
      <strong className={cn('text-2xl leading-none', tileColor(tile))}>{tileMainLabel(tile)}</strong>
    </span>
  )
}

function MiniTile({ tile }: { tile: MahjongTile }) {
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

function StatusPill({ children, icon }: { children: ReactNode, icon?: ReactNode }) {
  return (
    <span className="inline-flex min-h-9 items-center gap-2 rounded-lg border border-[#d8b66a]/35 bg-[#fff8e8]/10 px-3 text-sm font-black">
      {icon}
      {children}
    </span>
  )
}

function ScorePanel({ result }: { result?: WinResult }) {
  const { t } = useI18n()
  if (!result || result.patterns.length === 0) {
    return (
      <div className="mx-auto rounded-lg bg-[#081914]/70 px-3 py-2 text-sm font-bold text-[#fff8e8]/70">
        {t('mahjong.noPatterns')}
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-full rounded-lg bg-[#081914]/70 px-3 py-2 text-sm font-bold text-[#fff8e8]">
      <span className={result.canWin ? 'text-[#ffd166]' : 'text-[#fff8e8]/70'}>
        {result.fan}
        {' '}
        {t('mahjong.fan')}
      </span>
      <span className="mx-2 text-[#fff8e8]/35">|</span>
      {result.patterns.map(pattern => `${pattern.name}${pattern.fan}`).join(' / ')}
    </div>
  )
}

function claimLabel(option: ClaimOption, t: (key: string, params?: Record<string, string | number>) => string) {
  if (option.kind === 'hu') {
    return t('mahjong.claimWin', { fan: option.winResult?.fan ?? 0 })
  }
  if (option.kind === 'peng') {
    return t('mahjong.claimPong', { tile: formatTile(option.tile) })
  }

  return t('mahjong.claimChi', { tiles: [...option.tilesFromHand, option.tile].sort((left, right) => left.code.localeCompare(right.code)).map(formatTile).join('') })
}

function tileMainLabel(tile: MahjongTile) {
  if (tile.rank) {
    return tile.rank
  }

  return formatTile(tile)
}

function tileSuitLabel(tile: MahjongTile) {
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

function tileColor(tile: MahjongTile) {
  if (tile.kind === 'characters' || tile.dragon === 'red') {
    return 'text-[#b63b2f]'
  }
  if (tile.kind === 'bamboo' || tile.dragon === 'green') {
    return 'text-[#177554]'
  }

  return 'text-[#173047]'
}
