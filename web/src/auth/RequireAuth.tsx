import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import { useI18n } from '@/i18n/context'
import { useAuth } from './AuthContext'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { isLoading, user } = useAuth()
  const { t } = useI18n()
  const location = useLocation()

  if (isLoading) {
    return (
      <main className="grid min-h-svh place-items-center bg-background px-4 text-foreground">
        <p className="text-sm font-semibold text-muted-foreground">{t('auth.checking')}</p>
      </main>
    )
  }

  if (!user) {
    const next = `${location.pathname}${location.search}`
    return <Navigate replace to={`/login?next=${encodeURIComponent(next)}`} />
  }

  if (user.banned) {
    return (
      <main className="grid min-h-svh place-items-center bg-background px-4 text-foreground">
        <section className="w-full max-w-md rounded-lg border bg-card p-6 text-card-foreground shadow-sm">
          <h1 className="text-2xl font-bold">{t('auth.bannedTitle')}</h1>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">{t('auth.bannedDescription')}</p>
        </section>
      </main>
    )
  }

  return children
}
