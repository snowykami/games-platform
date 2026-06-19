import type { GameDefinition } from '@/games/registry'
import { DoorOpen, Gamepad2, Users } from 'lucide-react'
import { Link } from 'react-router'
import { Badge } from '@/shared/components/ui/badge'
import { Button } from '@/shared/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/components/ui/card'

interface GameCatalogPageProps {
  games: GameDefinition[]
  isLoading: boolean
}

export function GameCatalogPage({ games, isLoading }: GameCatalogPageProps) {
  const availableCount = games.filter(game => game.status === 'available').length

  return (
    <main className="min-h-svh bg-background">
      <section className="border-b bg-card">
        <div className="mx-auto flex max-w-7xl flex-col gap-5 px-4 py-8 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <div className="mb-3 inline-flex items-center gap-2 rounded-md border bg-background px-3 py-1 text-sm text-muted-foreground">
                <Gamepad2 className="size-4" />
                Snowy Games Platform
              </div>
              <h1 className="text-3xl font-semibold tracking-normal text-foreground sm:text-4xl">
                小游戏总览
              </h1>
              <p className="mt-3 text-base leading-7 text-muted-foreground">
                先从一个清晰的大厅开始，逐步接入本地与联机游戏。当前页面由统一游戏注册表驱动。
              </p>
            </div>
            <div className="grid grid-cols-2 gap-3 sm:w-80">
              <Metric label="已开放" value={availableCount} />
              <Metric label="规划中" value={games.length - availableCount} />
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {isLoading && (
          <div className="mb-4 rounded-md border bg-card px-4 py-3 text-sm text-muted-foreground">
            正在同步后端游戏列表，若后端未启动会使用本地注册表。
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {games.map(game => (
            <GameCard key={game.slug} game={game} />
          ))}
        </div>
      </section>
    </main>
  )
}

function GameCard({ game }: { game: GameDefinition }) {
  const isAvailable = game.status === 'available'

  return (
    <Card className="flex min-h-72 flex-col">
      <CardHeader>
        <div className="mb-3 flex items-start justify-between gap-3">
          <div className="flex size-11 items-center justify-center rounded-md bg-primary/10 text-primary">
            <Gamepad2 className="size-5" />
          </div>
          <Badge variant={isAvailable ? 'success' : 'secondary'}>
            {isAvailable ? '可玩' : '规划中'}
          </Badge>
        </div>
        <CardTitle className="text-xl">{game.title}</CardTitle>
        <CardDescription>{game.description}</CardDescription>
      </CardHeader>
      <CardContent className="mt-auto flex flex-col gap-4">
        <div className="flex flex-wrap gap-2">
          {game.tags.map(tag => <Badge key={tag} variant="outline">{tag}</Badge>)}
        </div>

        <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
          <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2">
            <Users className="size-4" />
            {game.minPlayers}
            -
            {game.maxPlayers}
            人
          </div>
          <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2">
            <DoorOpen className="size-4" />
            {game.supportsOnline ? '联机' : '本地'}
          </div>
        </div>

        <Button asChild disabled={!isAvailable} variant={isAvailable ? 'default' : 'secondary'}>
          {isAvailable
            ? <Link to={`/games/${game.slug}`}>进入游戏</Link>
            : <span>即将开放</span>}
        </Button>
      </CardContent>
    </Card>
  )
}

function Metric({ label, value }: { label: string, value: number }) {
  return (
    <div className="rounded-lg border bg-background p-4">
      <div className="text-2xl font-semibold text-foreground">{value}</div>
      <div className="mt-1 text-sm text-muted-foreground">{label}</div>
    </div>
  )
}
