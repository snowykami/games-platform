import type { UnoOnlineRoom } from './online'
import type { UnoColor } from './types'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { addUnoAI, drawUnoCard, joinUnoRoom, playUnoCard, startUnoRoom } from './online'

export function useUnoRoom(roomId: string | undefined) {
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
      setError(err instanceof Error ? err.message : '房间加载失败')
    }
    finally {
      setIsLoading(false)
    }
  }, [roomId])

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

  async function run(action: () => Promise<UnoOnlineRoom>) {
    try {
      setError(undefined)
      const nextRoom = await action()
      setRoom(nextRoom)
    }
    catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    }
  }

  const actions = useMemo(() => ({
    addAI: () => roomId ? run(() => addUnoAI(roomId)) : Promise.resolve(),
    draw: () => roomId ? run(() => drawUnoCard(roomId)) : Promise.resolve(),
    play: (cardId: string, color: UnoColor) => roomId ? run(() => playUnoCard(roomId, cardId, color)) : Promise.resolve(),
    refresh,
    start: () => roomId ? run(() => startUnoRoom(roomId)) : Promise.resolve(),
  }), [refresh, roomId])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/uno?room=${encodeURIComponent(roomId)}`
}
