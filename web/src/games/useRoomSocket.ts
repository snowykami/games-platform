import { useEffect, useRef } from 'react'

interface UseRoomSocketOptions {
  enabled: boolean
  onMessage: (event: MessageEvent) => void
  onReconnect?: () => void | Promise<void>
  url: string
}

const RECONNECT_DELAYS_MS = [500, 1000, 2000, 5000]

export function useRoomSocket({ enabled, onMessage, onReconnect, url }: UseRoomSocketOptions) {
  const socketRef = useRef<WebSocket | null>(null)
  const hasConnectedRef = useRef(false)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimerRef = useRef<number | undefined>(undefined)

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
        if (shouldRefresh) {
          void onReconnect?.()
        }
      }

      function handleClose() {
        if (socketRef.current === socket) {
          socketRef.current = null
        }
        detachSocketHandlers()

        if (disposed) {
          return
        }

        const delay = RECONNECT_DELAYS_MS[Math.min(reconnectAttemptRef.current, RECONNECT_DELAYS_MS.length - 1)]
        reconnectAttemptRef.current += 1
        reconnectTimerRef.current = window.setTimeout(connect, delay)
      }

      function handleError() {
        socket.close()
      }

      socket.onopen = handleOpen
      socket.onmessage = onMessage
      socket.onclose = handleClose
      socket.onerror = handleError
    }

    connect()

    return () => {
      disposed = true
      clearReconnectTimer()
      hasConnectedRef.current = false
      reconnectAttemptRef.current = 0
      socketRef.current?.close()
      socketRef.current = null
    }
  }, [enabled, onMessage, onReconnect, url])

  return socketRef
}
