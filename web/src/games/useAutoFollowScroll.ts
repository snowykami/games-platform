import { useCallback, useLayoutEffect, useRef } from 'react'

const BOTTOM_THRESHOLD_PX = 24

export function useAutoFollowScroll<T extends HTMLElement>() {
  const containerRef = useRef<T | null>(null)
  const shouldFollowRef = useRef(true)

  const handleScroll = useCallback(() => {
    const container = containerRef.current
    if (!container) {
      return
    }

    const distanceFromBottom = container.scrollHeight - container.clientHeight - container.scrollTop
    shouldFollowRef.current = distanceFromBottom <= BOTTOM_THRESHOLD_PX
  }, [])

  useLayoutEffect(() => {
    const container = containerRef.current
    if (!container || !shouldFollowRef.current) {
      return
    }

    container.scrollTop = container.scrollHeight
  })

  return { containerRef, handleScroll }
}
