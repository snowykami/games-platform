import type { XiangqiOnlineRoom } from './online'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { sendRoomSocketMessage } from '../roomSocket'
import { useRoomSocket } from '../useRoomSocket'
import { joinXiangqiRoom } from './online'

export function useXiangqiRoom(roomId: string | undefined) {
  const { t } = useI18n()
  const [room, setRoom] = useState<XiangqiOnlineRoom>()
  const [error, setError] = useState<string>()
  const [isLoading, setIsLoading] = useState(Boolean(roomId))

  const refresh = useCallback(async () => {
    if (!roomId) {
      return
    }

    setIsLoading(true)
    setError(undefined)
    try {
      setRoom(await joinXiangqiRoom(roomId))
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.loadingFailed'))
    }
    finally {
      setIsLoading(false)
    }
  }, [roomId, t])

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
        setError(t('room.operationFailed'))
      }
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.operationFailed'))
    }
  }, [socketRef, t])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? send('room.add_ai', { level }) : Promise.resolve(),
    move: (pieceId: string, x: number, y: number) => roomId ? send('room.move', { pieceId, x, y }) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? send('room.remove_player', { playerId }) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? send('room.rename', { name }) : Promise.resolve(),
    say: (text: string) => roomId ? send('room.speech', { text }) : Promise.resolve(),
    start: () => roomId ? send('room.start') : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? send('room.update_ai', { playerId, level }) : Promise.resolve(),
  }), [refresh, roomId, send])

  return { actions, connection, error, isLoading, room }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/xiangqi?room=${encodeURIComponent(roomId)}`
}
