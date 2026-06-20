import { useEffect, useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'

const OFFLINE_TIMEOUT_MS = 60_000

interface PlayerStatusDotProps {
  className?: string
  connected: boolean
  disconnectedAt?: string
}

export function PlayerStatusDot({ className, connected, disconnectedAt }: PlayerStatusDotProps) {
  const { t } = useI18n()
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (connected || !disconnectedAt) {
      return
    }
    const timer = window.setInterval(() => setNow(Date.now()), 5_000)
    return () => window.clearInterval(timer)
  }, [connected, disconnectedAt])

  const disconnectedTime = disconnectedAt ? Date.parse(disconnectedAt) : Number.NaN
  const isTimedOut = !connected && Number.isFinite(disconnectedTime) && now - disconnectedTime >= OFFLINE_TIMEOUT_MS
  const label = connected ? t('common.onlineStatus') : isTimedOut ? t('common.timedOut') : t('common.shortOffline')

  return (
    <span
      aria-label={label}
      className={cn(
        'inline-block size-2.5 shrink-0 rounded-full ring-2 ring-black/24',
        connected && 'bg-emerald-400 shadow-[0_0_14px_rgba(52,211,153,0.7)]',
        !connected && !isTimedOut && 'bg-amber-300 shadow-[0_0_14px_rgba(252,211,77,0.62)]',
        !connected && isTimedOut && 'bg-zinc-500',
        className,
      )}
      title={label}
    />
  )
}
