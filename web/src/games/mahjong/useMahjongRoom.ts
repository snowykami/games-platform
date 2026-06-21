import type { MahjongOnlineRoom } from './online'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { sendRoomSocketMessage } from '../roomSocket'
import { useRoomSocket } from '../useRoomSocket'
import { joinMahjongRoom } from './online'

export function useMahjongRoom(roomId: string | undefined) {
  const [room, setRoom] = useState<MahjongOnlineRoom>()
  const [error, setError] = useState<string>()
  const [isLoading, setIsLoading] = useState(Boolean(roomId))

  const refresh = useCallback(async () => {
    if (!roomId) {
      return
    }

    setIsLoading(true)
    setError(undefined)
    try {
      setRoom(await joinMahjongRoom(roomId))
    }
    catch (err) {
      setError(err instanceof Error ? err.message : '房间加载失败。')
    }
    finally {
      setIsLoading(false)
    }
  }, [roomId])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const handleSocketMessage = useCallback((event: MessageEvent) => {
    const data = JSON.parse(String(event.data))
    if (data.type === 'room.state') {
      setRoom(data.room)
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
    url: roomId ? createWebSocketURL(roomId) : '',
  })

  const send = useCallback(async (type: string, payload?: unknown) => {
    try {
      setError(undefined)
      if (!(await sendRoomSocketMessage(socketRef.current, type, payload))) {
        setError('操作失败。')
      }
    }
    catch (err) {
      setError(err instanceof Error ? err.message : '操作失败。')
    }
  }, [socketRef])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? send('room.add_ai', { level }) : Promise.resolve(),
    claim: (claimId: string) => roomId ? send('room.claim', { claimId }) : Promise.resolve(),
    discard: (tileId: string) => roomId ? send('room.discard', { tileId }) : Promise.resolve(),
    draw: () => roomId ? send('room.draw') : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? send('room.remove_player', { playerId }) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? send('room.rename', { name }) : Promise.resolve(),
    say: (text: string) => roomId ? send('room.speech', { text }) : Promise.resolve(),
    selfDraw: () => roomId ? send('room.self_draw') : Promise.resolve(),
    skipClaims: () => roomId ? send('room.skip_claims') : Promise.resolve(),
    start: () => roomId ? send('room.start') : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? send('room.update_ai', { playerId, level }) : Promise.resolve(),
  }), [refresh, roomId, send])

  return { actions, connection, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/mahjong?room=${encodeURIComponent(roomId)}`
}
