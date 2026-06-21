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
import { AlignmentBadge, HiddenRoleBadge, RoleBadge, StatusBadge } from './socialUi'

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
  const players = [...room.players].sort((left, right) => comparePlayerOrder(room, left, right))
  return (
    <div className={cn('grid content-start gap-3 overflow-auto pr-1', compact ? 'sm:grid-cols-2 xl:grid-cols-3' : 'sm:grid-cols-2', className)}>
      {players.map(player => (
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
  const numberLabel = playerNumberLabel(player)
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
          {numberLabel && (
            <span
              className="inline-flex min-h-5 shrink-0 items-center rounded-md border px-1.5 text-[0.68rem] font-black leading-none"
              style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
            >
              {numberLabel}
            </span>
          )}
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

      <SocialPlayerBadges player={player} room={room} />

      {canNote && (
        <PlayerNoteEditor className="mt-1" note={player.note} onSave={note => actions.updatePlayerNote(player.id, note)} />
      )}
    </article>
  )
}

function SocialPlayerBadges({ player, room }: { player: SocialPlayer, room: SocialRoom }) {
  const checkedAlignment = room.game === 'werewolf' ? room.werewolf.seerChecks?.[player.id] : undefined
  return (
    <div className="flex flex-wrap items-center gap-2">
      <StatusBadge state={player.alive ? 'alive' : 'out'} />
      {player.visibleToYou && player.role
        ? <RoleBadge dead={!player.alive} role={player.role} />
        : checkedAlignment
          ? <AlignmentBadge alignment={checkedAlignment} />
          : <HiddenRoleBadge />}
    </div>
  )
}

export function TableLogLine({ entry, room }: { entry: SocialRoom['log'][number], room: SocialRoom }) {
  const speaker = entry.playerId ? room.players.find(player => player.id === entry.playerId) : undefined

  if (!speaker) {
    return <p className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/76">{entry.text}</p>
  }

  const accent = playerAccent(room, speaker.id)
  const numberLabel = playerNumberLabel(speaker)
  const rest = entry.playerName && entry.text.startsWith(entry.playerName)
    ? entry.text.slice(entry.playerName.length)
    : entry.text

  return (
    <p className="rounded-lg bg-black/24 px-3 py-2 text-sm font-bold leading-6 text-[#fff8e8]/78">
      <span
        className="mr-1.5 inline-flex min-h-6 items-center rounded-md border px-2 align-baseline text-xs font-black leading-none"
        style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
      >
        {numberLabel && (
          <>
            {numberLabel}
            <span className="mx-1 opacity-65">·</span>
          </>
        )}
        {speaker.name}
      </span>
      <span>{rest}</span>
    </p>
  )
}

function playerAccent(room: SocialRoom, playerId?: string): PlayerAccent {
  const playerIndex = playerVisualIndex(room, playerId)
  return PLAYER_COLOR_PALETTE[playerIndex % PLAYER_COLOR_PALETTE.length]
}

function playerNumberLabel(player?: SocialPlayer) {
  const seat = playerSeatIndex(player)
  return seat >= 0 ? `${seat + 1}号` : ''
}

function playerSeatIndex(player?: SocialPlayer) {
  return player && Number.isFinite(player.seat) && player.seat >= 0 ? player.seat : -1
}

function playerVisualIndex(room: SocialRoom, playerId?: string) {
  const player = room.players.find(item => item.id === playerId)
  const seat = playerSeatIndex(player)
  if (seat >= 0) {
    return seat
  }
  return Math.max(0, room.players.findIndex(item => item.id === playerId))
}

function comparePlayerOrder(room: SocialRoom, left: SocialPlayer, right: SocialPlayer) {
  const leftSeat = playerSeatIndex(left)
  const rightSeat = playerSeatIndex(right)
  if (leftSeat >= 0 && rightSeat >= 0 && leftSeat !== rightSeat) {
    return leftSeat - rightSeat
  }
  if (leftSeat >= 0 && rightSeat < 0) {
    return -1
  }
  if (leftSeat < 0 && rightSeat >= 0) {
    return 1
  }
  return room.players.indexOf(left) - room.players.indexOf(right)
}

export function PlayerRefLabel({ className, player, room }: { className?: string, player: SocialPlayer, room: SocialRoom }) {
  const accent = playerAccent(room, player.id)
  const numberLabel = playerNumberLabel(player)
  return (
    <span className={cn('inline-flex min-w-0 items-center gap-1.5 align-middle', !player.alive && 'opacity-70', className)}>
      {numberLabel && (
        <span
          className="inline-flex min-h-5 shrink-0 items-center rounded-md border px-1.5 text-[0.68rem] font-black leading-none"
          style={{ backgroundColor: accent.soft, borderColor: accent.border, color: accent.text }}
        >
          {numberLabel}
        </span>
      )}
      <span className="min-w-0 truncate font-black" style={{ color: accent.text, textShadow: `0 0 14px ${accent.soft}` }}>
        {player.name}
      </span>
    </span>
  )
}
