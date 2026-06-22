import type { SocialGameSlug, SocialPlayer, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { Send, Shield, Skull, Vote } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { usePendingAction } from '../usePendingAction'
import { ChoiceButton, ConfirmChoiceButton, SubmittedNotice } from './socialActionControls'
import { PlayerRefLabel } from './socialPlayers'
import { socialButton } from './socialStyle'
import { Panel, SocialBadge } from './socialUi'

export function WerewolfActionPanel({
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
  const [selectedNightAction, setSelectedNightAction] = useState('')
  const [wolfMessage, setWolfMessage] = useState('')
  const [selectedWerewolfVote, setSelectedWerewolfVote] = useState('')
  const [selectedHunterTarget, setSelectedHunterTarget] = useState('')
  const pending = usePendingAction()
  const livingTargets = room.players.filter(player => player.alive && player.id !== you.id)
  const yourWerewolfVote = room.werewolf.votes[you.id]
  const hunterPending = room.players.find(player => player.id === room.werewolf.hunterPendingId)

  if (room.phase === 'night') {
    const canAct = ['werewolf', 'seer', 'guard', 'witch'].includes(you.role ?? '')
    const witchVictim = room.players.find(player => player.id === room.werewolf.witchVictimId)
    const nightTargets = werewolfNightTargets(room, you)
    const nightSubmitted = Boolean(room.werewolf.nightActionSubmitted) && you.role !== 'werewolf'
    const canSubmitNightAction = canAct && !nightSubmitted && (you.role !== 'witch' || Boolean(witchVictim))
    const selectedNightPlayerID = selectedNightAction.split(':')[1] ?? ''
    const selectedNightPlayer = room.players.find(player => player.id === selectedNightPlayerID)
    const selectedNightLabel = selectedNightAction === 'skip:witch'
      ? t('werewolf.skipWitch')
      : selectedNightAction === 'skip:wolf'
        ? t('werewolf.skipWolfKill')
        : selectedNightPlayer
          ? (
              <span className="inline-flex flex-wrap items-center gap-1.5">
                {selectedNightAction.startsWith('save:') && t('werewolf.useAntidoteTarget')}
                {selectedNightAction.startsWith('poison:') && t('werewolf.usePoisonTarget')}
                {selectedNightAction.startsWith('target:') && t('werewolf.nightTarget')}
                <PlayerRefLabel player={selectedNightPlayer} room={room} />
              </span>
            )
          : undefined

    function toggleNightAction(actionID: string) {
      setSelectedNightAction(selectedNightAction === actionID ? '' : actionID)
    }

    function submitWolfMessage() {
      const text = wolfMessage.trim()
      if (!text) {
        return
      }
      void pending.run('wolf-speech', () => actions.wolfSpeech(text).then(() => {
        setWolfMessage('')
        setMessage(t('werewolf.wolfMessageSent'))
      }), { releaseOnSettle: false })
    }

    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('werewolf.nightAction')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{nightSubmitted ? t('werewolf.actionSubmitted') : canAct ? t('werewolf.chooseNightTarget') : t('werewolf.noNightAction')}</p>
        {nightSubmitted && <SubmittedNotice config={config} label={t('werewolf.actionSubmitted')} />}
        {you.role === 'werewolf' && (
          <div className="grid min-w-0 gap-2 overflow-hidden rounded-lg border border-white/12 bg-black/18 p-3">
            <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3">
              <h3 className="min-w-0 truncate text-sm font-black text-[#fff8e8]">{t('werewolf.wolfChat')}</h3>
              <span className="shrink-0 text-xs font-black text-[#fff8e8]/58">{t('werewolf.wolfChoices')}</span>
            </div>
            <div className="grid min-w-0 gap-1.5">
              {room.players.filter(player => player.alive && player.role === 'werewolf').map(player => (
                <div key={player.id} className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2 rounded-md bg-black/22 px-2 py-1.5 text-xs font-black text-[#fff8e8]/76">
                  <span className="min-w-0 overflow-hidden">
                    <PlayerRefLabel player={player} room={room} />
                  </span>
                  <span className="max-w-28 truncate text-right" title={werewolfChoiceLabel(room, room.werewolf.wolfNightActions?.[player.id], t('werewolf.skipWolfKill'), t('werewolf.wolfChoicePending'))}>{werewolfChoiceLabel(room, room.werewolf.wolfNightActions?.[player.id], t('werewolf.skipWolfKill'), t('werewolf.wolfChoicePending'))}</span>
                </div>
              ))}
            </div>
            <div className="grid max-h-32 min-w-0 gap-1.5 overflow-y-auto overflow-x-hidden pr-1">
              {(room.werewolf.wolfSpeeches ?? []).map((speech) => {
                const speaker = room.players.find(player => player.id === speech.playerId)
                return (
                  <p key={speech.id} className="min-w-0 break-words rounded-md bg-black/24 px-2 py-1.5 text-xs font-bold leading-5 text-[#fff8e8]/76">
                    {speaker ? <PlayerRefLabel player={speaker} room={room} /> : speech.playerName}
                    <span className="mx-1 text-[#fff8e8]/45">:</span>
                    <span>{speech.text}</span>
                  </p>
                )
              })}
            </div>
            <div className="flex min-w-0 gap-2">
              <textarea
                className="min-h-11 min-w-0 flex-1 resize-none rounded-lg border border-white/12 bg-black/22 px-3 py-2 text-sm font-bold text-[#fff8e8] outline-none placeholder:text-[#fff8e8]/35 focus:border-white/28"
                maxLength={120}
                placeholder={t('werewolf.wolfChatPlaceholder')}
                value={wolfMessage}
                onChange={event => setWolfMessage(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault()
                    submitWolfMessage()
                  }
                }}
              />
              <button className={cn(socialButton(config, true), 'shrink-0 px-3')} disabled={!wolfMessage.trim() || pending.isPending('wolf-speech')} type="button" onClick={submitWolfMessage}>
                <Send className="size-4" />
                <span className="sr-only">{pending.isPending('wolf-speech') ? t('common.syncing') : t('common.send')}</span>
              </button>
            </div>
          </div>
        )}
        {you.role === 'witch' && !nightSubmitted && (
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
            {witchVictim && (
              <>
                {!room.werewolf.witchAntidoteUsed && (
                  <ChoiceButton config={config} icon={<Shield className="size-4" />} selected={selectedNightAction === `save:${witchVictim.id}`} onClick={() => toggleNightAction(`save:${witchVictim.id}`)}>
                    {t('werewolf.useAntidoteTarget')}
                    <PlayerRefLabel player={witchVictim} room={room} />
                  </ChoiceButton>
                )}
                {!room.werewolf.witchPoisonUsed && livingTargets.map(player => (
                  <ChoiceButton key={player.id} config={config} icon={<Skull className="size-4" />} selected={selectedNightAction === `poison:${player.id}`} onClick={() => toggleNightAction(`poison:${player.id}`)}>
                    {t('werewolf.usePoisonTarget')}
                    <PlayerRefLabel player={player} room={room} />
                  </ChoiceButton>
                ))}
                <ChoiceButton config={config} selected={selectedNightAction === 'skip:witch'} onClick={() => toggleNightAction('skip:witch')}>
                  {t('werewolf.skipWitch')}
                </ChoiceButton>
              </>
            )}
          </>
        )}
        {canAct && !nightSubmitted && you.role !== 'witch' && nightTargets.map(player => (
          <ChoiceButton key={player.id} config={config} selected={selectedNightAction === `target:${player.id}`} onClick={() => toggleNightAction(`target:${player.id}`)}>
            <WerewolfNightRelationBadge player={player} you={you} />
            <PlayerRefLabel player={player} room={room} />
          </ChoiceButton>
        ))}
        {you.role === 'werewolf' && (
          <ChoiceButton config={config} selected={selectedNightAction === 'skip:wolf'} onClick={() => toggleNightAction('skip:wolf')}>
            {t('werewolf.skipWolfKill')}
          </ChoiceButton>
        )}
        {canSubmitNightAction && (
          <ConfirmChoiceButton
            config={config}
            disabled={!selectedNightAction}
            label={t('social.confirmAction')}
            selectedLabel={selectedNightLabel}
            onConfirm={() => void actions.nightAction(selectedNightAction).then(() => {
              setSelectedNightAction('')
              setMessage(t('werewolf.actionSubmitted'))
            })}
          />
        )}
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
    const voterCount = room.players.filter(player => player.alive && !room.werewolf.revealedIdiots?.[player.id]).length
    const speakerCount = Object.keys(room.werewolf.daySpeakers ?? {}).length
    return (
      <Panel config={config}>
        <h2 className="text-xl font-black">{t('werewolf.dayDiscussion')}</h2>
        <p className="text-sm leading-6 text-[#fff8e8]/76">{room.werewolf.lastNight || t('werewolf.dayHint')}</p>
        <div className="rounded-lg border border-white/12 bg-white/8 p-3 text-sm font-black text-[#fff8e8]/82">
          {t('werewolf.autoVoteHint', { count: speakerCount, total: voterCount })}
        </div>
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
              if (activeWerewolfVoteTarget === player.id) {
                setSelectedWerewolfVote('')
                void actions.werewolfVote('', false)
                return
              }
              setSelectedWerewolfVote(player.id)
              void actions.werewolfVote(player.id, false)
            }}
          >
            <PlayerRefLabel player={player} room={room} />
          </ChoiceButton>
        ))}
        <ConfirmChoiceButton
          config={config}
          disabled={!activeWerewolfVoteTarget}
          label={t('social.confirmVote')}
          selectedLabel={selectedPlayer ? <PlayerRefLabel player={selectedPlayer} room={room} /> : undefined}
          onConfirm={() => void actions.werewolfVote(activeWerewolfVoteTarget, true).then(() => setMessage(t('werewolf.voted')))}
        />
      </Panel>
    )
  }

  return <Panel config={config}>{t('social.waiting')}</Panel>
}

function werewolfNightTargets(room: SocialRoom, you: SocialPlayer) {
  return room.players.filter((player) => {
    if (!player.alive) {
      return false
    }
    if (you.role === 'seer') {
      return player.id !== you.id
    }
    return true
  })
}

function werewolfChoiceLabel(room: SocialRoom, actionId: string | undefined, skipLabel: string, pendingLabel: string) {
  if (!actionId) {
    return pendingLabel
  }
  if (actionId === 'skip:wolf') {
    return skipLabel
  }
  const player = room.players.find(item => item.id === actionId)
  if (!player) {
    return pendingLabel
  }
  return `${player.seat >= 0 ? `${player.seat + 1}号 · ` : ''}${player.name}`
}

function WerewolfNightRelationBadge({ player, you }: { player: SocialPlayer, you: SocialPlayer }) {
  if (you.role !== 'werewolf') {
    return (
      <SocialBadge className="bg-white/12 text-[#fff8e8]/76 ring-1 ring-white/12">
        目标
      </SocialBadge>
    )
  }
  const relation = player.role === 'werewolf' ? 'friend' : 'enemy'
  return (
    <SocialBadge
      className={cn(
        relation === 'friend'
          ? 'bg-[#123729] text-[#9fffd0] ring-[#36d399]/40'
          : 'bg-[#4a1424] text-[#ffd6df] ring-[#ff7a9a]/45',
      )}
    >
      {relation === 'friend' ? '友' : '敌'}
    </SocialBadge>
  )
}
