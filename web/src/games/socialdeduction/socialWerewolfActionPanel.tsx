import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Shield, Skull, Vote } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { ChoiceButton, ConfirmChoiceButton, SubmittedNotice } from './socialActionControls'
import { PlayerRefLabel } from './socialPlayers'
import { socialButton } from './socialStyle'
import { Panel } from './socialUi'

export function WerewolfActionPanel({
  actions,
  config,
  isHost,
  room,
  setMessage,
  you,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  room: SocialRoom
  setMessage: (message: string) => void
  you: SocialPlayer
}) {
  const { t } = useI18n()
  const [selectedWerewolfVote, setSelectedWerewolfVote] = useState('')
  const [selectedHunterTarget, setSelectedHunterTarget] = useState('')
  const livingTargets = room.players.filter(player => player.alive && player.id !== you.id)
  const yourWerewolfVote = room.werewolf.votes[you.id]
  const hunterPending = room.players.find(player => player.id === room.werewolf.hunterPendingId)

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

  return <Panel config={config}>{t('social.waiting')}</Panel>
}
