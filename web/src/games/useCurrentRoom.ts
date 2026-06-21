import { useEffect, useState } from 'react'

export function useCurrentRoom<T>(enabled: boolean, load: () => Promise<T | null>) {
  const [room, setRoom] = useState<T | null>(null)

  useEffect(() => {
    let active = true
    if (!enabled) {
      return () => {
        active = false
      }
    }

    void load()
      .then((nextRoom) => {
        if (active) {
          setRoom(nextRoom)
        }
      })
      .catch(() => {
        if (active) {
          setRoom(null)
        }
      })

    return () => {
      active = false
    }
  }, [enabled, load])

  return { currentRoom: enabled ? room : null }
}
