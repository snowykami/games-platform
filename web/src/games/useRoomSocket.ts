import { useEffect, useMemo, useRef, useState } from 'react'

interface UseRoomSocketOptions {
  enabled: boolean
  onMessage: (event: MessageEvent) => void
  onReconnect?: () => void | Promise<void>
  url: string
}

const RECONNECT_DELAYS_MS = [500, 1000, 2000, 5000]
const PING_INTERVAL_MS = 5000

export type RoomConnectionState = 'connected' | 'connecting' | 'disconnected' | 'reconnecting'

export interface RoomConnectionInfo {
  latencyMs?: number
  state: RoomConnectionState
}

export function useRoomSocket({ enabled, onMessage, onReconnect, url }: UseRoomSocketOptions) {
  const socketRef = useRef<WebSocket | null>(null)
  const hasConnectedRef = useRef(false)
  const pingTimerRef = useRef<number | undefined>(undefined)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimerRef = useRef<number | undefined>(undefined)
  const [connectionState, setConnectionState] = useState<RoomConnectionState>(enabled ? 'connecting' : 'disconnected')
  const [latencyMs, setLatencyMs] = useState<number>()

  function clearPingTimer() {
    if (pingTimerRef.current !== undefined) {
      window.clearInterval(pingTimerRef.current)
      pingTimerRef.current = undefined
    }
  }

  useEffect(() => {
    if (!enabled) {
      return undefined
    }

    let disposed = false

    function clearReconnectTimer() {
      if (reconnectTimerRef.current !== undefined) {
        window.clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = undefined
      }
    }

    function sendPing(socket: WebSocket) {
      if (socket.readyState !== WebSocket.OPEN) {
        return
      }
      socket.send(JSON.stringify({ payload: { sentAt: Date.now() }, type: 'ping' }))
    }

    function startPing(socket: WebSocket) {
      clearPingTimer()
      sendPing(socket)
      // eslint-disable-next-line react/web-api-no-leaked-interval
      pingTimerRef.current = window.setInterval(sendPing, PING_INTERVAL_MS, socket)
    }

    function connect() {
      clearReconnectTimer()
      if (disposed) {
        return
      }

      const socket = new WebSocket(url)
      socketRef.current = socket

      function detachSocketHandlers() {
        socket.onopen = null
        socket.onmessage = null
        socket.onclose = null
        socket.onerror = null
      }

      function handleOpen() {
        const shouldRefresh = hasConnectedRef.current
        hasConnectedRef.current = true
        reconnectAttemptRef.current = 0
        setConnectionState('connected')
        startPing(socket)
        if (shouldRefresh) {
          void onReconnect?.()
        }
      }

      function handleClose() {
        if (socketRef.current === socket) {
          socketRef.current = null
        }
        clearPingTimer()
        detachSocketHandlers()

        if (disposed) {
          return
        }

        setConnectionState('reconnecting')
        setLatencyMs(undefined)
        const delay = RECONNECT_DELAYS_MS[Math.min(reconnectAttemptRef.current, RECONNECT_DELAYS_MS.length - 1)]
        reconnectAttemptRef.current += 1
        reconnectTimerRef.current = window.setTimeout(connect, delay)
      }

      function handleError() {
        socket.close()
      }

      function handleMessage(event: MessageEvent) {
        const data = JSON.parse(String(event.data))
        if (data.type === 'pong') {
          const sentAt = Number(data.payload?.sentAt)
          if (Number.isFinite(sentAt)) {
            setLatencyMs(Math.max(0, Date.now() - sentAt))
          }
          return
        }
        onMessage(event)
      }

      socket.onopen = handleOpen
      socket.onmessage = handleMessage
      socket.onclose = handleClose
      socket.onerror = handleError
    }

    connect()

    return () => {
      disposed = true
      clearPingTimer()
      clearReconnectTimer()
      hasConnectedRef.current = false
      reconnectAttemptRef.current = 0
      socketRef.current?.close()
      socketRef.current = null
      setConnectionState('disconnected')
      setLatencyMs(undefined)
    }
  }, [enabled, onMessage, onReconnect, url])

  const connection = useMemo<RoomConnectionInfo>(() => ({ latencyMs, state: connectionState }), [connectionState, latencyMs])

  return useMemo(() => ({ connection, socketRef }), [connection, socketRef])
}
