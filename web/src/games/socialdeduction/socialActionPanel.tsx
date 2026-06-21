import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Skull } from 'lucide-react'
import { useI18n } from '@/i18n/context'
import { DeadPlayerPanel } from './socialActionControls'
import { AvalonActionPanel } from './socialAvalonActionPanel'
import { Panel } from './socialUi'
import { UndercoverActionPanel } from './socialUndercoverActionPanel'
import { WerewolfActionPanel } from './socialWerewolfActionPanel'

export function ActionPanel({
  actions,
  config,
  game,
  room,
  setMessage,
  you,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  room: SocialRoom
  setMessage: (message: string) => void
  you?: SocialPlayer
}) {
  const { t } = useI18n()

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
    return <DeadPlayerPanel config={config} />
  }

  if (game === 'werewolf') {
    return <WerewolfActionPanel actions={actions} config={config} room={room} setMessage={setMessage} you={you} />
  }

  if (game === 'undercover') {
    return <UndercoverActionPanel actions={actions} config={config} room={room} setMessage={setMessage} you={you} />
  }

  if (game === 'avalon') {
    return <AvalonActionPanel actions={actions} config={config} room={room} setMessage={setMessage} you={you} />
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}
