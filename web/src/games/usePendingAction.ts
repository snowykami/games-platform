import { useCallback, useEffect, useMemo, useReducer, useRef } from 'react'

interface PendingActionOptions {
  releaseOnSettle?: boolean
}

const DEFAULT_PENDING_TIMEOUT_MS = 10000

export function usePendingAction(timeoutMs = DEFAULT_PENDING_TIMEOUT_MS) {
  const [, refreshPending] = useReducer((version: number) => version + 1, 0)
  const pendingRef = useRef<Set<string>>(new Set())
  const timersRef = useRef<Map<string, number>>(new Map())

  const clear = useCallback((key: string) => {
    const timer = timersRef.current.get(key)
    if (timer !== undefined) {
      window.clearTimeout(timer)
      timersRef.current.delete(key)
    }
    pendingRef.current.delete(key)
    refreshPending()
  }, [refreshPending])

  const clearAll = useCallback(() => {
    timersRef.current.forEach(timer => window.clearTimeout(timer))
    timersRef.current.clear()
    pendingRef.current.clear()
    refreshPending()
  }, [refreshPending])

  const isPending = useCallback((key: string) => pendingRef.current.has(key), [])

  const run = useCallback(async <T>(key: string, action: () => T | Promise<T>, options: PendingActionOptions = {}) => {
    if (pendingRef.current.has(key)) {
      return undefined
    }
    pendingRef.current.add(key)
    refreshPending()
    const timer = window.setTimeout(clear, timeoutMs, key)
    timersRef.current.set(key, timer)

    try {
      return await action()
    }
    finally {
      if (options.releaseOnSettle !== false) {
        clear(key)
      }
    }
  }, [clear, refreshPending, timeoutMs])

  useEffect(() => clearAll, [clearAll])

  return useMemo(() => ({ clear, clearAll, isPending, run }), [clear, clearAll, isPending, run])
}
