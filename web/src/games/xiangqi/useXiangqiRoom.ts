import type { XiangqiOnlineRoom } from './online'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { addXiangqiAI, joinXiangqiRoom, moveXiangqiPiece, removeXiangqiPlayer, sayXiangqi, startXiangqiRoom, updateXiangqiAI } from './online'

export function useXiangqiRoom(roomId: string | undefined) {
  const { t } = useI18n()
  const [room, setRoom] = useState<XiangqiOnlineRoom>()
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

  const run = useCallback(async (action: () => Promise<XiangqiOnlineRoom>) => {
    try {
      setError(undefined)
      setRoom(await action())
    }
    catch (err) {
      setError(err instanceof Error ? err.message : t('room.operationFailed'))
    }
  }, [t])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? run(() => addXiangqiAI(roomId, level)) : Promise.resolve(),
    move: (pieceId: string, x: number, y: number) => roomId ? run(() => moveXiangqiPiece(roomId, pieceId, x, y)) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? run(() => removeXiangqiPlayer(roomId, playerId)) : Promise.resolve(),
    say: (text: string) => roomId ? run(() => sayXiangqi(roomId, text)) : Promise.resolve(),
    start: () => roomId ? run(() => startXiangqiRoom(roomId)) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateXiangqiAI(roomId, playerId, level)) : Promise.resolve(),
  }), [refresh, roomId, run])

  return { actions, error, isLoading, room }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/xiangqi?room=${encodeURIComponent(roomId)}`
}
