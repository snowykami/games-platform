import type { ReactNode } from 'react'
import type { AuthUser } from './types'
import { useEffect, useMemo, useState } from 'react'
import { getMe, loginAsGuest, logoutSession } from './api'
import { AuthContext } from './AuthContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser>()
  const [isLoading, setIsLoading] = useState(true)

  async function refresh() {
    setIsLoading(true)
    try {
      setUser(await getMe())
    }
    finally {
      setIsLoading(false)
    }
  }

  async function loginGuest() {
    const guestUuid = getOrCreateGuestUuid()
    const nextUser = await loginAsGuest(guestUuid)
    setUser(nextUser)
    return nextUser
  }

  async function logout() {
    await logoutSession()
    setUser(undefined)
  }

  useEffect(() => {
    void refresh()
  }, [])

  const value = useMemo(() => ({ isLoading, loginGuest, logout, refresh, user }), [isLoading, user])

  return <AuthContext value={value}>{children}</AuthContext>
}

function getOrCreateGuestUuid() {
  const key = 'games-platform.guest-uuid'
  const existing = window.localStorage.getItem(key)
  if (existing) {
    return existing
  }

  const next = crypto.randomUUID()
  window.localStorage.setItem(key, next)
  return next
}
