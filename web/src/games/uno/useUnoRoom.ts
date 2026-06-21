import type { UnoOnlineRoom } from './online'
import type { UnoColor } from './types'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { sendRoomSocketMessage } from '../roomSocket'
import { joinUnoRoom } from './online'

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
  }, [t])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? send('room.add_ai', { level }) : Promise.resolve(),
    callUno: () => roomId ? send('room.call_uno') : Promise.resolve(),
    catchUno: (targetId: string) => roomId ? send('room.catch_uno', { targetId }) : Promise.resolve(),
    draw: () => roomId ? send('room.draw') : Promise.resolve(),
    play: (cardId: string, color: UnoColor) => roomId ? send('room.play', { cardId, color }) : Promise.resolve(),
    refresh,
    removePlayer: (playerId: string) => roomId ? send('room.remove_player', { playerId }) : Promise.resolve(),
    renamePlayer: (name: string) => roomId ? send('room.rename', { name }) : Promise.resolve(),
    say: (text: string) => roomId ? send('room.speech', { text }) : Promise.resolve(),
    start: () => roomId ? send('room.start') : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? send('room.update_ai', { playerId, level }) : Promise.resolve(),
  }), [refresh, roomId, send])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/uno?room=${encodeURIComponent(roomId)}`
}
