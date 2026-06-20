import type { MahjongOnlineRoom } from './online'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { addMahjongAI, claimMahjong, discardMahjongTile, drawMahjongTile, joinMahjongRoom, sayMahjong, selfDrawMahjong, skipMahjongClaims, startMahjongRoom, updateMahjongAI } from './online'

export function useMahjongRoom(roomId: string | undefined) {
  const [room, setRoom] = useState<MahjongOnlineRoom>()
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

  const run = useCallback(async (action: () => Promise<MahjongOnlineRoom>) => {
    try {
      setError(undefined)
      const nextRoom = await action()
      setRoom(nextRoom)
    }
    catch (err) {
      setError(err instanceof Error ? err.message : '操作失败。')
    }
  }, [])

  const actions = useMemo(() => ({
    addAI: (level: string) => roomId ? run(() => addMahjongAI(roomId, level)) : Promise.resolve(),
    claim: (claimId: string) => roomId ? run(() => claimMahjong(roomId, claimId)) : Promise.resolve(),
    discard: (tileId: string) => roomId ? run(() => discardMahjongTile(roomId, tileId)) : Promise.resolve(),
    draw: () => roomId ? run(() => drawMahjongTile(roomId)) : Promise.resolve(),
    refresh,
    say: (text: string) => roomId ? run(() => sayMahjong(roomId, text)) : Promise.resolve(),
    selfDraw: () => roomId ? run(() => selfDrawMahjong(roomId)) : Promise.resolve(),
    skipClaims: () => roomId ? run(() => skipMahjongClaims(roomId)) : Promise.resolve(),
    start: () => roomId ? run(() => startMahjongRoom(roomId)) : Promise.resolve(),
    updateAI: (playerId: string, level: string) => roomId ? run(() => updateMahjongAI(roomId, playerId, level)) : Promise.resolve(),
  }), [refresh, roomId, run])

  return { actions, error, isLoading, room, setRoom }
}

function createWebSocketURL(roomId: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/mahjong?room=${encodeURIComponent(roomId)}`
}
