import type { UnoOnlineRoom } from './online'
import type { UnoColor } from './types'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { addUnoAI, callUno, catchUno, drawUnoCard, joinUnoRoom, playUnoCard, removeUnoPlayer, sayUno, startUnoRoom, updateUnoAI } from './online'

export function useUnoRoom(roomId: string | undefined) {
  const { t } = useI18n()
  const [room, setRoom] = useState<UnoOnlineRoom>()
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
      setRoom(await joinUnoRoom(roomId))
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

  const run = useCallback(async (action: () => Promise<UnoOnlineRoom>) => {
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
    addAI: (level: string) => roomId ? run(() => addUnoAI(roomId, level)) : Promise.resolve(),
    callUno: () => roomId ? run(() => callUno(roomId)) : Promise.resolve(),
    catchUno: (targetId: string) => roomId ? run(() => catchUno(roomId, targetId)) : Promise.resolve(),
    draw: () => roomId ? run(() => drawUnoCard(roomId)) : Promise.resolve(),
    play: (cardId: string, color: UnoColor) => roomId ? run(() => playUnoCard(roomId, cardId, color)) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? run(() => removeUnoPlayer(roomId, playerId)) : Promise.resolve(),
    say: (text: string) => roomId ? run(() => sayUno(roomId, text)) : Promise.resolve(),
    start: () => roomId ? run(() => startUnoRoom(roomId)) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateUnoAI(roomId, playerId, level)) : Promise.resolve(),
  }), [refresh, roomId, run])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/uno?room=${encodeURIComponent(roomId)}`
}
