import type { KeyboardEvent } from 'react'
import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Vote } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { usePendingAction } from '../usePendingAction'
import { ChoiceButton, ConfirmChoiceButton, SubmittedNotice } from './socialActionControls'
import { PlayerRefLabel } from './socialPlayers'
import { socialButton } from './socialStyle'
import { Panel } from './socialUi'

export function UndercoverActionPanel({
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
  const [selectedUndercoverVote, setSelectedUndercoverVote] = useState('')
  const [description, setDescription] = useState('')
  const pending = usePendingAction()
  const livingTargets = room.players.filter(player => player.alive && player.id !== you.id)
  const isCurrentSpeaker = room.undercover.currentSpeakerId === you.id
  const isSubmittingDescription = pending.isPending('describe')

  async function submitDescription() {
    const nextDescription = description.trim()
    if (!isCurrentSpeaker || !nextDescription) {
      return
    }

    await pending.run('describe', () => actions.undercoverDescribe(nextDescription).then(() => {
      setDescription('')
      setMessage(t('undercover.described'))
    }), { releaseOnSettle: false })
  }

  function handleDescriptionKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) {
      return
    }

    event.preventDefault()
    void submitDescription()
  }

  if (room.phase === 'describe') {
    const currentSpeaker = room.players.find(player => player.id === room.undercover.currentSpeakerId)

    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('undercover.describeTitle')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">
          {isCurrentSpeaker ? t('undercover.yourTurnDescribe') : t('undercover.waitDescribe', { name: currentSpeaker?.name ?? '-' })}
        </p>
        <textarea
          className="min-h-24 resize-none rounded-lg border border-white/18 bg-black/28 p-3 text-sm font-bold text-[#fff8e8] outline-none focus:ring-2 focus:ring-[#f4c7ff]"
          disabled={!isCurrentSpeaker || isSubmittingDescription}
          maxLength={80}
          placeholder={t('undercover.describePlaceholder')}
          value={description}
          onKeyDown={handleDescriptionKeyDown}
          onChange={event => setDescription(event.target.value)}
        />
        <button
          className={socialButton(config, true)}
          disabled={!isCurrentSpeaker || !description.trim() || isSubmittingDescription}
          type="button"
          onClick={() => void submitDescription()}
        >
          {isSubmittingDescription ? t('common.syncing') : t('undercover.submitDescription')}
        </button>
      </Panel>
    )
  }

  if (room.phase === 'undercover_vote') {
    const yourUndercoverVote = room.undercover.votes[you.id]
    const hasVoted = Boolean(yourUndercoverVote?.confirmed)
    const activeUndercoverVoteTarget = hasVoted ? yourUndercoverVote?.targetId ?? '' : selectedUndercoverVote || yourUndercoverVote?.targetId || ''
    const selectedPlayer = room.players.find(player => player.id === activeUndercoverVoteTarget)

    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('undercover.voteTitle')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{t('undercover.voteHint')}</p>
        {hasVoted && <SubmittedNotice config={config} label={t('undercover.voted')} />}
        {livingTargets.map(player => (
          <ChoiceButton
            key={player.id}
            config={config}
            icon={<Vote className="size-4" />}
            disabled={hasVoted}
            selected={activeUndercoverVoteTarget === player.id}
            onClick={() => {
              if (hasVoted) {
                return
              }
              setSelectedUndercoverVote(player.id)
              void actions.undercoverVote(player.id, false)
            }}
          >
            <PlayerRefLabel player={player} room={room} />
          </ChoiceButton>
        ))}
        <ConfirmChoiceButton
          config={config}
          disabled={!activeUndercoverVoteTarget || hasVoted}
          label={hasVoted ? t('undercover.voted') : t('social.confirmVote')}
          selectedLabel={selectedPlayer ? <PlayerRefLabel player={selectedPlayer} room={room} /> : undefined}
          onConfirm={() => void actions.undercoverVote(activeUndercoverVoteTarget, true).then(() => setMessage(t('undercover.voted')))}
        />
      </Panel>
    )
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}
