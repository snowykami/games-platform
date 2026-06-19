import type { GameDefinition } from '@/games/registry'
import { DoorOpen, Gamepad2, LogIn, ShieldCheck, Sparkles, UserRound, Users } from 'lucide-react'
import { Link } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { Badge } from '@/shared/components/ui/badge'
import { Button } from '@/shared/components/ui/button'

interface GameCatalogPageProps {
  games: GameDefinition[]
  isLoading: boolean
}

const GAME_COVERS: Record<string, string> = {
  gomoku: '/game-covers/gomoku.webp',
  mahjong: '/game-covers/mahjong.webp',
  uno: '/game-covers/uno.webp',
  xiangqi: '/game-covers/xiangqi.webp',
}

export function GameCatalogPage({ games, isLoading }: GameCatalogPageProps) {
  const { isLoading: isAuthLoading, user } = useAuth()
  const availableCount = games.filter(game => game.status === 'available').length

  return (
    <main className="min-h-svh bg-[#f7f4ee] text-[#191611]">
      <section className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_12%,rgba(229,67,46,0.16),transparent_30%),radial-gradient(circle_at_82%_0%,rgba(33,129,112,0.16),transparent_26%),linear-gradient(180deg,#fffaf0_0%,#f5efe5_100%)]" />
        <div className="relative mx-auto flex max-w-7xl flex-col gap-5 px-4 py-8 sm:px-6 sm:py-10 lg:px-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <div className="mb-4 inline-flex items-center gap-2 rounded-full bg-[#191611] px-3 py-1.5 text-sm font-bold text-[#fff8e8] shadow-[0_10px_28px_rgba(25,22,17,0.12)]">
                <Gamepad2 className="size-4" />
                Liteyuki Games
              </div>
              <h1 className="text-4xl font-black tracking-normal text-[#191611] sm:text-5xl">
                小游戏总览
              </h1>
              <p className="mt-3 max-w-2xl text-base font-medium leading-7 text-[#62594d]">
                先从一个清晰的大厅开始，逐步接入本地与联机游戏。当前页面由统一游戏注册表驱动。
              </p>
            </div>
            <div className="grid gap-3 sm:w-[420px]">
              <UserPanel isLoading={isAuthLoading} user={user} />
              <div className="grid grid-cols-2 gap-3">
                <Metric label="已开放" value={availableCount} />
                <Metric label="规划中" value={games.length - availableCount} />
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-7 sm:px-6 lg:px-8">
        {isLoading && (
          <div className="mb-4 rounded-lg bg-[#fffaf0] px-4 py-3 text-sm font-semibold text-[#62594d] shadow-[0_10px_30px_rgba(25,22,17,0.08)]">
            正在同步后端游戏列表，若后端未启动会使用本地注册表。
          </div>
        )}

        <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
          {games.map(game => (
            <GameCard key={game.slug} game={game} />
          ))}
        </div>
      </section>
    </main>
  )
}

function UserPanel({
  isLoading,
  user,
}: {
  isLoading: boolean
  user?: {
    displayName: string
    kind: 'guest' | 'oidc'
    role: 'admin' | 'player'
  }
}) {
  const shouldShowLogin = !user || user.kind === 'guest'

  return (
    <div className="flex min-h-16 flex-col gap-3 rounded-xl bg-[#fffaf0]/86 p-3 shadow-[0_16px_40px_rgba(25,22,17,0.12)] ring-1 ring-[#191611]/10 backdrop-blur sm:flex-row sm:items-center sm:justify-between">
      <div className="flex min-w-0 items-center gap-3">
        <span className="grid size-10 shrink-0 place-items-center rounded-lg bg-[#e9f4ef] text-[#1d806d]">
          {user?.role === 'admin' ? <ShieldCheck className="size-5" /> : <UserRound className="size-5" />}
        </span>
        <div className="min-w-0">
          <p className="text-xs font-bold text-[#756b5e]">当前用户</p>
          <div className="mt-1 flex min-w-0 flex-wrap items-center gap-2">
            <strong className="truncate text-sm text-[#191611]">
              {isLoading ? '同步中...' : user?.displayName ?? '未登录'}
            </strong>
            {user && <Badge variant={user.kind === 'guest' ? 'secondary' : 'success'}>{user.kind === 'guest' ? '游客' : 'OIDC'}</Badge>}
            {user?.role === 'admin' && <Badge variant="outline">管理员</Badge>}
          </div>
        </div>
      </div>

      {shouldShowLogin && (
        <Button asChild className="min-h-10 shrink-0" size="sm" variant="secondary">
          <Link to="/login?next=/">
            <LogIn className="size-4" />
            登录
          </Link>
        </Button>
      )}
    </div>
  )
}

function GameCard({ game }: { game: GameDefinition }) {
  const isAvailable = game.status === 'available'
  const cover = GAME_COVERS[game.slug] ?? GAME_COVERS.uno

  return (
    <article className="group relative min-h-[360px] overflow-hidden rounded-xl bg-[#191611] shadow-[0_22px_60px_rgba(25,22,17,0.16)] ring-1 ring-[#191611]/10 transition duration-300 hover:-translate-y-1 hover:shadow-[0_28px_70px_rgba(25,22,17,0.24)]">
      <img
        alt=""
        className="absolute inset-0 size-full object-cover transition duration-500 group-hover:scale-105"
        src={cover}
      />
      <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(10,8,6,0.05)_0%,rgba(10,8,6,0.35)_34%,rgba(10,8,6,0.9)_100%)]" />
      <div className="absolute inset-x-0 top-0 flex items-start justify-between gap-3 p-4">
        <span className="inline-flex items-center gap-2 rounded-full bg-[#fff8e8]/92 px-3 py-1.5 text-xs font-black text-[#191611] shadow-[0_10px_24px_rgba(0,0,0,0.16)]">
          <Sparkles className="size-3.5 text-[#c43d28]" />
          {isAvailable ? '可玩' : '规划中'}
        </span>
        <span className="rounded-full bg-[#191611]/55 px-3 py-1.5 text-xs font-black text-[#fff8e8] backdrop-blur">
          {game.supportsOnline ? '联机' : '本地'}
        </span>
      </div>

      <div className="relative z-10 flex min-h-[360px] flex-col justify-end p-4">
        <div className="mb-3 flex flex-wrap gap-2">
          {game.tags.map(tag => (
            <span key={tag} className="rounded-full bg-[#fff8e8]/14 px-2.5 py-1 text-xs font-bold text-[#fff8e8] ring-1 ring-[#fff8e8]/18 backdrop-blur">
              {tag}
            </span>
          ))}
        </div>

        <h2 className="text-2xl font-black text-[#fff8e8]">{game.title}</h2>
        <p className="mt-2 min-h-[72px] text-sm font-semibold leading-6 text-[#fff8e8]/78">
          {game.description}
        </p>

        <div className="mt-4 grid grid-cols-2 gap-2 text-sm font-bold text-[#fff8e8]/86">
          <div className="flex items-center gap-2 rounded-lg bg-[#fff8e8]/12 px-3 py-2 backdrop-blur">
            <Users className="size-4" />
            {game.minPlayers}
            -
            {game.maxPlayers}
            人
          </div>
          <div className="flex items-center gap-2 rounded-lg bg-[#fff8e8]/12 px-3 py-2 backdrop-blur">
            <DoorOpen className="size-4" />
            {game.supportsOnline ? '联机' : '本地'}
          </div>
        </div>

        {isAvailable
          ? (
              <Button asChild className="mt-4 min-h-11" variant="secondary">
                <Link to={`/games/${game.slug}`}>进入游戏</Link>
              </Button>
            )
          : (
              <span className="mt-4 inline-flex min-h-11 items-center justify-center rounded-lg bg-[#fff8e8]/18 px-4 text-sm font-black text-[#fff8e8]/76 ring-1 ring-[#fff8e8]/18 backdrop-blur">
                即将开放
              </span>
            )}
      </div>
    </article>
  )
}

function Metric({ label, value }: { label: string, value: number }) {
  return (
    <div className="rounded-xl bg-[#fffaf0]/74 p-4 shadow-[0_16px_40px_rgba(25,22,17,0.1)] ring-1 ring-[#191611]/10 backdrop-blur">
      <div className="text-3xl font-black text-[#191611]">{value}</div>
      <div className="mt-1 text-sm font-bold text-[#756b5e]">{label}</div>
    </div>
  )
}
