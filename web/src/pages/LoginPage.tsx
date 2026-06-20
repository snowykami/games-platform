import type { OIDCProvider } from '@/auth/types'
import { Bot, DoorOpen } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { getOIDCProviders } from '@/auth/api'
import { useAuth } from '@/auth/AuthContext'
import { useI18n } from '@/i18n/context'
import { useDocumentMeta } from '@/shared/brand/meta'
import { Button } from '@/shared/components/ui/button'

export function LoginPage() {
  const { loginGuest, user } = useAuth()
  const { t } = useI18n()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [oidcProviders, setOIDCProviders] = useState<OIDCProvider[]>([])
  const [message, setMessage] = useState(() => t('auth.loginMessage'))
  const next = safeNext(searchParams.get('next'))
  useDocumentMeta(t('common.login'))

  useEffect(() => {
    void getOIDCProviders().then(providers => setOIDCProviders(providers.filter(provider => provider.enabled)))
  }, [])

  useEffect(() => {
    if (user?.kind === 'oidc') {
      navigate(next, { replace: true })
    }
  }, [navigate, next, user])

  async function handleGuestLogin() {
    setIsSubmitting(true)
    setMessage(t('auth.creatingGuestSession'))
    try {
      await loginGuest()
      navigate(next, { replace: true })
    }
    catch (error) {
      setMessage(error instanceof Error ? error.message : t('auth.loginFailed'))
    }
    finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="grid min-h-svh place-items-center bg-[radial-gradient(circle_at_top_left,rgba(215,47,47,0.18),transparent_32%),radial-gradient(circle_at_bottom_right,rgba(39,104,199,0.18),transparent_36%),#100d0b] px-4 text-[#fff8e8]">
      <section className="w-full max-w-md rounded-lg border border-white/20 bg-[#15110e]/82 p-6 shadow-[0_28px_90px_rgba(0,0,0,0.38)] backdrop-blur">
        <div className="flex items-center gap-3">
          <span className="grid size-12 place-items-center rounded-lg bg-[#fff8e8] text-[#15110e]">
            <img alt="" className="size-7" src="/favicon.svg" />
          </span>
          <div>
            <p className="text-xs font-black text-[#fff8e8]/65">LITEYUKI GAMES</p>
            <h1 className="text-3xl font-black tracking-normal">{t('auth.loginTitle')}</h1>
          </div>
        </div>

        <p className="mt-4 text-sm leading-7 text-[#fff8e8]/75">
          {t('auth.loginDescription')}
        </p>

        <div className="mt-5 grid gap-3">
          <Button className="min-h-12 justify-start text-base" disabled={isSubmitting} variant="secondary" onClick={handleGuestLogin}>
            <DoorOpen className="size-5" />
            {t('auth.guestLogin')}
          </Button>
          {oidcProviders.map(provider => (
            <Button key={provider.key} asChild className="min-h-12 justify-start text-base text-[#191611]" variant="secondary">
              <a href={`/api/auth/oidc/${encodeURIComponent(provider.key)}/login?returnTo=${encodeURIComponent(next)}`}>
                <Bot className="size-5" />
                {provider.displayName}
              </a>
            </Button>
          ))}
          {oidcProviders.length === 0 && (
            <Button className="min-h-12 justify-start text-base" disabled variant="outline">
              <Bot className="size-5" />
              {t('auth.oidcNotConfigured')}
            </Button>
          )}
        </div>

        <p className="mt-4 min-h-6 text-sm font-bold text-[#fff8e8]/70">{message}</p>
      </section>
    </main>
  )
}

function safeNext(next: string | null) {
  if (!next || !next.startsWith('/') || next.startsWith('//')) {
    return '/'
  }
  return next
}
