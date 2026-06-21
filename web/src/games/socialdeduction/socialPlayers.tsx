import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY, PlayerAccent } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { UserMinus } from 'lucide-react'
import { SpeechBubble, SpeechButton } from '@/games/GameSpeech'
import { PlayerNameEditor } from '@/games/PlayerNameEditor'
import { PlayerNoteEditor } from '@/games/PlayerNoteEditor'
import { PlayerStatusDot } from '@/games/PlayerStatusDot'
import { latestSpeechForPlayer } from '@/games/speech'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { socialIconButton } from './socialStyle'
import { PLAYER_COLOR_PALETTE } from './socialTheme'
import { RoleBadge } from './socialUi'

type SocialActions = ReturnType<typeof useSocialRoom>['actions']

export function PlayerGrid({
  actions,
  className,
  compact = false,
  config,
  isHost,
  llmModel,
  room,
}: {
  actions: SocialActions
  className?: string
  compact?: boolean
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  llmModel: string
  room: SocialRoom
}) {
  return (
    <div className={cn('grid content-start gap-3 overflow-auto pr-1', compact ? 'sm:grid-cols-2 xl:grid-cols-3' : 'sm:grid-cols-2', className)}>
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
  actions: SocialActions
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

export function TableLogLine({ entry, room }: { entry: SocialRoom['log'][number], room: SocialRoom }) {
  const speaker = entry.playerId ? room.players.find(player => player.id === entry.playerId) : undefined

  if (!speaker) {
    return <p className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/76">{entry.text}</p>
  }

  const accent = playerAccent(room, speaker.id)
  const rest = entry.playerName && entry.text.startsWith(entry.playerName)
    ? entry.text.slice(entry.playerName.length)
    : entry.text

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

export function PlayerRefLabel({ className, player, room }: { className?: string, player: SocialPlayer, room: SocialRoom }) {
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
