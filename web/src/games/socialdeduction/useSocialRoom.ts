import type { SocialGameSlug, SocialRoom, WerewolfRoleConfig } from './online'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '@/i18n/context'
import {
  addSocialAI,
  advanceWerewolfDay,
  assassinateAvalon,
  describeUndercover,
  joinSocialRoom,
  parseSocialRoom,
  playAvalonQuest,
  proposeAvalonTeam,
  removeSocialPlayer,
  renameSocialPlayer,
  saySocial,
  startSocialRoom,
  updateSocialAI,
  updateSocialPlayerNote,
  updateUndercoverConfig,
  updateWerewolfRoles,
  voteAvalonTeam,
  voteUndercover,
  werewolfHunterShot,
  werewolfNightAction,
  werewolfVote,
} from './online'

export function useSocialRoom(game: SocialGameSlug, roomId: string | undefined) {
  const { t } = useI18n()
  const [room, setRoom] = useState<SocialRoom>()
  const [error, setError] = useState<string>()
  const [isLoading, setIsLoading] = useState(Boolean(roomId))
  const socketRef = useRef<WebSocket | null>(null)

  const refresh = useCallback(async () => {
    if (!roomId) {
      return
    }

    setIsLoading(true)
    setError(undefined)
    try {
      setRoom(await joinSocialRoom(game, roomId))
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.loadingFailed'))
    }
    finally {
      setIsLoading(false)
    }
  }, [game, roomId, t])

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    if (!roomId) {
      return undefined
    }

    const socket = new WebSocket(createWebSocketURL(game, roomId))
    socketRef.current = socket
    function handleMessage(event: MessageEvent) {
      const data = JSON.parse(String(event.data))
      if (data.type === 'room.state') {
        setRoom(parseSocialRoom(data.room))
        setError(undefined)
      }
      if (data.type === 'error') {
        setError(data.error)
      }
    }

    function handleClose() {
      if (socketRef.current === socket) {
        socketRef.current = null
      }
    }

    socket.addEventListener('message', handleMessage)
    socket.addEventListener('close', handleClose)

    return () => {
      socketRef.current = null
      socket.removeEventListener('message', handleMessage)
      socket.removeEventListener('close', handleClose)
      socket.close()
    }
  }, [game, roomId])

  const run = useCallback(async (action: () => Promise<SocialRoom>) => {
    try {
      setError(undefined)
      const nextRoom = await action()
      setRoom(nextRoom)
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.operationFailed'))
    }
  }, [t])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? run(() => addSocialAI(game, roomId, level)) : Promise.resolve(),
    advanceDay: () => roomId ? run(() => advanceWerewolfDay(roomId)) : Promise.resolve(),
    assassinate: (targetId: string) => roomId ? run(() => assassinateAvalon(roomId, targetId)) : Promise.resolve(),
    hunterShot: (targetId: string) => roomId ? run(() => werewolfHunterShot(roomId, targetId)) : Promise.resolve(),
    nightAction: (actionId: string) => roomId ? run(() => werewolfNightAction(roomId, actionId)) : Promise.resolve(),
    playQuest: (card: 'success' | 'fail') => roomId ? run(() => playAvalonQuest(roomId, card)) : Promise.resolve(),
    proposeTeam: (team: string[]) => roomId ? run(() => proposeAvalonTeam(roomId, team)) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? run(() => removeSocialPlayer(game, roomId, playerId)) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? run(() => renameSocialPlayer(game, roomId, name)) : Promise.resolve(),
    say: (text: string) => roomId ? run(() => saySocial(game, roomId, text)) : Promise.resolve(),
    start: () => roomId ? run(() => startSocialRoom(game, roomId)) : Promise.resolve(),
    teamVote: (approve: boolean) => roomId ? run(() => voteAvalonTeam(roomId, approve)) : Promise.resolve(),
    undercoverConfig: (presetId: string, includeBlank: boolean) => roomId ? run(() => updateUndercoverConfig(roomId, presetId, includeBlank)) : Promise.resolve(),
    undercoverDescribe: (text: string) => roomId ? run(() => describeUndercover(roomId, text)) : Promise.resolve(),
    undercoverVote: (targetId: string) => roomId ? run(() => voteUndercover(roomId, targetId)) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateSocialAI(game, roomId, playerId, level)) : Promise.resolve(),
    updatePlayerNote: (playerId: string, note: string) => roomId ? run(() => updateSocialPlayerNote(game, roomId, playerId, note)) : Promise.resolve(),
    updateWerewolfRoles: (config: WerewolfRoleConfig) => roomId ? run(() => updateWerewolfRoles(roomId, config)) : Promise.resolve(),
    werewolfVote: (targetId: string) => roomId ? run(() => werewolfVote(roomId, targetId)) : Promise.resolve(),
  }), [game, refresh, roomId, run])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(game: SocialGameSlug, roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/${game}?room=${encodeURIComponent(roomId)}`
}
