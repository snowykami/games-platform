import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Skull } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { ChoiceButton, ConfirmChoiceButton, SubmittedNotice } from './socialActionControls'
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
