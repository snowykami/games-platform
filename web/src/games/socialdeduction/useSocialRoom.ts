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

export function useSocialRoom(game: SocialGameSlug, roomId: string | undefined, godView = false) {
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

  useEffect(() => {
    if (!roomId) {
      return undefined
    }

    const socket = new WebSocket(createWebSocketURL(game, roomId, godView))
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
  }, [game, godView, roomId])

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
    addAI: (level: string) => roomId ? run(() => addSocialAI(game, roomId, level, { godView })) : Promise.resolve(),
    advanceDay: () => roomId ? run(() => advanceWerewolfDay(roomId, { godView })) : Promise.resolve(),
    assassinate: (targetId: string) => roomId ? run(() => assassinateAvalon(roomId, targetId, { godView })) : Promise.resolve(),
    hunterShot: (targetId: string) => roomId ? run(() => werewolfHunterShot(roomId, targetId, { godView })) : Promise.resolve(),
    nightAction: (actionId: string) => roomId ? run(() => werewolfNightAction(roomId, actionId, { godView })) : Promise.resolve(),
    playQuest: (card: 'success' | 'fail') => roomId ? run(() => playAvalonQuest(roomId, card, { godView })) : Promise.resolve(),
    proposeTeam: (team: string[]) => roomId ? run(() => proposeAvalonTeam(roomId, team, { godView })) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? run(() => removeSocialPlayer(game, roomId, playerId, { godView })) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? run(() => renameSocialPlayer(game, roomId, name, { godView })) : Promise.resolve(),
    say: (text: string) => roomId ? run(() => saySocial(game, roomId, text, { godView })) : Promise.resolve(),
    start: () => roomId ? run(() => startSocialRoom(game, roomId, { godView })) : Promise.resolve(),
    teamVote: (approve: boolean) => roomId ? run(() => voteAvalonTeam(roomId, approve, { godView })) : Promise.resolve(),
    undercoverConfig: (presetId: string, includeBlank: boolean) => roomId ? run(() => updateUndercoverConfig(roomId, presetId, includeBlank, { godView })) : Promise.resolve(),
    undercoverDescribe: (text: string) => roomId ? run(() => describeUndercover(roomId, text, { godView })) : Promise.resolve(),
    undercoverVote: (targetId: string, confirmed: boolean) => roomId ? run(() => voteUndercover(roomId, targetId, confirmed, { godView })) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateSocialAI(game, roomId, playerId, level, { godView })) : Promise.resolve(),
    updatePlayerNote: (playerId: string, note: string) => roomId ? run(() => updateSocialPlayerNote(game, roomId, playerId, note, { godView })) : Promise.resolve(),
    updateWerewolfRoles: (config: WerewolfRoleConfig) => roomId ? run(() => updateWerewolfRoles(roomId, config, { godView })) : Promise.resolve(),
    werewolfVote: (targetId: string, confirmed: boolean) => roomId ? run(() => werewolfVote(roomId, targetId, confirmed, { godView })) : Promise.resolve(),
  }), [game, godView, refresh, roomId, run])

  return { actions, error, isLoading, room, setRoom }
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
