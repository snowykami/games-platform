import type { OIDCProvider } from '@/auth/types'
import { Bot, DoorOpen, ShieldCheck } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { getOIDCProviders } from '@/auth/api'
import { useAuth } from '@/auth/AuthContext'
import { Button } from '@/shared/components/ui/button'

export function LoginPage() {
  const { loginGuest, user } = useAuth()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [oidcProviders, setOIDCProviders] = useState<OIDCProvider[]>([])
  const [message, setMessage] = useState('选择游客登录或已配置的 OIDC provider。')
  const next = safeNext(searchParams.get('next'))

  useEffect(() => {
    void getOIDCProviders().then(providers => setOIDCProviders(providers.filter(provider => provider.enabled)))
  }, [])

  useEffect(() => {
    if (user) {
      navigate(next, { replace: true })
    }
  }, [navigate, next, user])

  async function handleGuestLogin() {
    setIsSubmitting(true)
    setMessage('正在创建游客会话...')
    try {
      await loginGuest()
      navigate(next, { replace: true })
    }
    catch (error) {
      setMessage(error instanceof Error ? error.message : '登录失败')
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
            <ShieldCheck className="size-6" />
          </span>
          <div>
            <p className="text-xs font-black text-[#fff8e8]/65">GAMES PLATFORM</p>
            <h1 className="text-3xl font-black tracking-normal">登录后进入房间</h1>
          </div>
        </div>

        <p className="mt-4 text-sm leading-7 text-[#fff8e8]/75">
          分享链接会在登录后继续打开。游客身份会保留对局数据，但其他玩家只会看到游客展示名。
        </p>

        <div className="mt-5 grid gap-3">
          <Button className="min-h-12 justify-start rounded-lg text-base font-black" disabled={isSubmitting} onClick={handleGuestLogin}>
            <DoorOpen className="size-5" />
            游客登录
          </Button>
          {oidcProviders.map(provider => (
            <Button key={provider.key} asChild className="min-h-12 justify-start rounded-lg text-base font-black" variant="outline">
              <a href={`/api/auth/oidc/${encodeURIComponent(provider.key)}/login?returnTo=${encodeURIComponent(next)}`}>
                <Bot className="size-5" />
                {provider.displayName}
              </a>
            </Button>
          ))}
          {oidcProviders.length === 0 && (
            <Button className="min-h-12 justify-start rounded-lg text-base font-black" disabled variant="outline">
              <Bot className="size-5" />
              OIDC 未配置
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
