import type { SocialGameSlug, SocialRoom, WerewolfRoleConfig } from './online'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { sendRoomSocketMessage } from '../roomSocket'
import { useRoomSocket } from '../useRoomSocket'
import { joinSocialRoom, parseSocialRoom } from './online'

export function useSocialRoom(game: SocialGameSlug, roomId: string | undefined, godView = false) {
  const { t } = useI18n()
  const [room, setRoom] = useState<SocialRoom>()
  const [error, setError] = useState<string>()
  const [isLoading, setIsLoading] = useState(Boolean(roomId))

  const refresh = useCallback(async () => {
    if (!roomId) {
      return
    }

    setIsLoading(true)
    setError(undefined)
    try {
      setRoom(await joinSocialRoom(game, roomId, { godView }))
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.loadingFailed'))
    }
    finally {
      setIsLoading(false)
    }
  }, [game, godView, roomId, t])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const handleSocketMessage = useCallback((event: MessageEvent) => {
    const data = JSON.parse(String(event.data))
    if (data.type === 'room.state') {
      setRoom(parseSocialRoom(data.room))
      setError(undefined)
    }
    if (data.type === 'error') {
      setError(data.error)
    }
  }, [])

  const { connection, socketRef } = useRoomSocket({
    enabled: Boolean(roomId),
    onMessage: handleSocketMessage,
    onReconnect: refresh,
    url: roomId ? createWebSocketURL(game, roomId, godView) : '',
  })

  const send = useCallback(async (type: string, payload?: unknown) => {
    try {
      setError(undefined)
      if (!(await sendRoomSocketMessage(socketRef.current, type, payload))) {
        setError(t('room.operationFailed'))
      }
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.operationFailed'))
    }
  }, [socketRef, t])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? send('room.add_ai', { level }) : Promise.resolve(),
    advanceDay: () => roomId ? send('room.advance_day') : Promise.resolve(),
    assassinate: (targetId: string) => roomId ? send('room.assassinate', { targetId }) : Promise.resolve(),
    hunterShot: (targetId: string) => roomId ? send('room.hunter_shot', { targetId }) : Promise.resolve(),
    nightAction: (actionId: string) => roomId ? send('room.night_action', { actionId }) : Promise.resolve(),
    playQuest: (card: 'success' | 'fail') => roomId ? send('room.quest', { card }) : Promise.resolve(),
    proposeTeam: (team: string[]) => roomId ? send('room.team', { team }) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? send('room.remove_player', { playerId }) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? send('room.rename', { name }) : Promise.resolve(),
    say: (text: string) => roomId ? send('room.speech', { text }) : Promise.resolve(),
    start: () => roomId ? send('room.start') : Promise.resolve(),
    teamVote: (approve: boolean) => roomId ? send('room.team_vote', { approve }) : Promise.resolve(),
    undercoverConfig: (domainIds: string[], includeBlank: boolean) => roomId ? send('room.undercover_config', { domainIds, includeBlank }) : Promise.resolve(),
    undercoverDescribe: (text: string) => roomId ? send('room.describe', { text }) : Promise.resolve(),
    undercoverVote: (targetId: string, confirmed: boolean) => roomId ? send('room.undercover_vote', { targetId, confirmed }) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? send('room.update_ai', { playerId, level }) : Promise.resolve(),
    updatePlayerNote: (playerId: string, note: string) => roomId ? send('room.note', { playerId, note }) : Promise.resolve(),
    updateWerewolfRoles: (config: WerewolfRoleConfig) => roomId ? send('room.werewolf_roles', { config }) : Promise.resolve(),
    wolfSpeech: (text: string) => roomId ? send('room.wolf_speech', { text }) : Promise.resolve(),
    werewolfVote: (targetId: string, confirmed: boolean) => roomId ? send('room.werewolf_vote', { targetId, confirmed }) : Promise.resolve(),
  }), [refresh, roomId, send])

  return { actions, connection, error, isLoading, room, setRoom }
}

function createWebSocketURL(game: SocialGameSlug, roomId: string, godView: boolean) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = new URL(`${protocol}//${window.location.host}/ws/${game}`)
  url.searchParams.set('room', roomId)
  if (godView) {
    url.searchParams.set('godView', '1')
  }
  return url.toString()
}
