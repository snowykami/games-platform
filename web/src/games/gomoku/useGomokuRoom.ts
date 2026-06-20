import type { GomokuOnlineRoom } from './online'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { addGomokuAI, joinGomokuRoom, placeGomokuStone, sayGomoku, startGomokuRoom, updateGomokuAI } from './online'

export function useGomokuRoom(roomId: string | undefined) {
  const { t } = useI18n()
  const [room, setRoom] = useState<GomokuOnlineRoom>()
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
      setRoom(await joinGomokuRoom(roomId))
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

  useEffect(() => {
    if (!roomId) {
      return undefined
    }

    const socket = new WebSocket(createWebSocketURL(roomId))
    socketRef.current = socket
    function handleMessage(event: MessageEvent) {
      const data = JSON.parse(String(event.data))
      if (data.type === 'room.state') {
        setRoom(data.room)
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
  }, [roomId])

  const run = useCallback(async (action: () => Promise<GomokuOnlineRoom>) => {
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
    addAI: (level: string) => roomId ? run(() => addGomokuAI(roomId, level)) : Promise.resolve(),
    place: (x: number, y: number) => roomId ? run(() => placeGomokuStone(roomId, x, y)) : Promise.resolve(),
    refresh,
    say: (text: string) => roomId ? run(() => sayGomoku(roomId, text)) : Promise.resolve(),
    start: () => roomId ? run(() => startGomokuRoom(roomId)) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateGomokuAI(roomId, playerId, level)) : Promise.resolve(),
  }), [refresh, roomId, run])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/gomoku?room=${encodeURIComponent(roomId)}`
}
