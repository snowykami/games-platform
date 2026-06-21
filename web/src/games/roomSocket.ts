export async function sendRoomSocketMessage(socket: WebSocket | null, type: string, payload?: unknown) {
  if (!socket) {
    return false
  }

  if (socket.readyState === WebSocket.CONNECTING) {
    const opened = await waitForSocketOpen(socket, 2000)
    if (!opened) {
      return false
    }
  }

  if (socket.readyState !== WebSocket.OPEN) {
    return false
  }

  socket.send(JSON.stringify(payload === undefined ? { type } : { type, payload }))
  return true
}

function waitForSocketOpen(socket: WebSocket, timeoutMs: number) {
  return new Promise<boolean>((resolve) => {
    if (socket.readyState === WebSocket.OPEN) {
      resolve(true)
      return
    }

    const timeout = window.setTimeout(() => {
      cleanup()
      resolve(false)
    }, timeoutMs)

    function cleanup() {
      window.clearTimeout(timeout)
      socket.removeEventListener('open', handleOpen)
      socket.removeEventListener('close', handleClosed)
      socket.removeEventListener('error', handleClosed)
    }

    function handleOpen() {
      cleanup()
      resolve(true)
    }

    function handleClosed() {
      cleanup()
      resolve(false)
    }

    socket.addEventListener('open', handleOpen, { once: true })
    socket.addEventListener('close', handleClosed, { once: true })
    socket.addEventListener('error', handleClosed, { once: true })
  })
}
