import type { ReactNode } from 'react'
import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Check, Skull, X } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { ChoiceButton, ConfirmChoiceButton, SubmittedNotice } from './socialActionControls'
import { PlayerRefLabel } from './socialPlayers'
import { socialButton } from './socialStyle'
import { Panel } from './socialUi'

export function AvalonActionPanel({
  actions,
  config,
  room,
  setMessage,
  you,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  room: SocialRoom
  setMessage: (message: string) => void
  you: SocialPlayer
}) {
  const { t } = useI18n()
  const [selectedTeam, setSelectedTeam] = useState<string[]>(room.avalon.team)
  const [selectedTeamVote, setSelectedTeamVote] = useState<boolean>()
  const [teamVoteSubmitted, setTeamVoteSubmitted] = useState(false)
  const [selectedQuestCard, setSelectedQuestCard] = useState<'success' | 'fail'>()
  const [questSubmitted, setQuestSubmitted] = useState(false)
  const [selectedAssassinationTarget, setSelectedAssassinationTarget] = useState('')
  const alivePlayers = room.players.filter(player => player.alive)
  const onQuest = room.avalon.team.includes(you.id)
  const isLeader = room.avalon.leaderId === you.id
  const isAssassin = you.role === 'assassin'
  const teamPlayers = room.avalon.team
    .map(id => room.players.find(player => player.id === id))
    .filter((player): player is SocialPlayer => Boolean(player))

  function toggleTeam(playerId: string) {
    setSelectedTeam(selectedTeam.includes(playerId)
      ? selectedTeam.filter(id => id !== playerId)
      : selectedTeam.length < room.avalon.requiredTeam ? [...selectedTeam, playerId] : selectedTeam)
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
              <PlayerRefLabel player={player} room={room} />
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
        <div className="flex flex-wrap gap-2 text-sm leading-6 text-[#fff8e8]/76">
          {teamPlayers.map(player => <PlayerRefLabel key={player.id} player={player} room={room} />)}
        </div>
        {teamVoteSubmitted && <SubmittedNotice config={config} label={t('avalon.voted')} />}
        <div className="grid grid-cols-2 gap-2">
          <BinaryChoiceButton disabled={teamVoteSubmitted} selected={selectedTeamVote === true} tone="positive" onClick={() => setSelectedTeamVote(true)}>
            <Check className="size-4" />
            {t('avalon.approve')}
          </BinaryChoiceButton>
          <BinaryChoiceButton disabled={teamVoteSubmitted} selected={selectedTeamVote === false} tone="negative" onClick={() => setSelectedTeamVote(false)}>
            <X className="size-4" />
            {t('avalon.reject')}
          </BinaryChoiceButton>
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
              <BinaryChoiceButton disabled={questSubmitted} selected={selectedQuestCard === 'success'} tone="positive" onClick={() => setSelectedQuestCard('success')}>
                <Check className="size-4" />
                {t('avalon.successCard')}
              </BinaryChoiceButton>
              <BinaryChoiceButton disabled={questSubmitted || you.alignment !== 'evil'} selected={selectedQuestCard === 'fail'} tone="negative" onClick={() => setSelectedQuestCard('fail')}>
                <X className="size-4" />
                {t('avalon.failCard')}
              </BinaryChoiceButton>
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
            <PlayerRefLabel player={player} room={room} />
          </ChoiceButton>
        ))}
        {isAssassin && (
          <ConfirmChoiceButton
            config={config}
            disabled={!selectedAssassinationTarget}
            label={t('social.confirmAction')}
            selectedLabel={selectedPlayer ? <PlayerRefLabel player={selectedPlayer} room={room} /> : undefined}
            onConfirm={() => void actions.assassinate(selectedAssassinationTarget).then(() => setMessage(t('avalon.assassinated')))}
          />
        )}
      </Panel>
    )
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}

function BinaryChoiceButton({
  children,
  disabled = false,
  onClick,
  selected,
  tone,
}: {
  children: ReactNode
  disabled?: boolean
  onClick: () => void
  selected: boolean
  tone: 'positive' | 'negative'
}) {
  const toneClass = tone === 'positive'
    ? 'border-emerald-300/42 bg-emerald-500/14 text-emerald-100 hover:bg-emerald-300 hover:text-emerald-950'
    : 'border-rose-300/42 bg-rose-500/14 text-rose-100 hover:bg-rose-300 hover:text-rose-950'

  return (
    <button
      className={cn('inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border px-3 text-sm font-black transition disabled:cursor-not-allowed disabled:opacity-45', toneClass, selected && 'ring-2 ring-[#fff8e8] ring-offset-2 ring-offset-black/40')}
      disabled={disabled}
      type="button"
      onClick={onClick}
    >
      {children}
    </button>
  )
}
