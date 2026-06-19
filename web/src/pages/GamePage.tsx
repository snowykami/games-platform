import { ArrowLeft } from 'lucide-react'
import { Link, useParams, useSearchParams } from 'react-router'
import { findGame } from '@/games/registry'
import { UnoRoomGate } from '@/games/uno/UnoRoomGate'
import { Button } from '@/shared/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/shared/components/ui/card'

export function GamePage() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const roomId = searchParams.get('room')?.trim() || undefined
  const game = slug ? findGame(slug) : undefined

  if (!game) {
    return (
      <CenteredState
        description="这个 slug 没有注册到游戏列表里。"
        title="游戏不存在"
      />
    )
  }

  if (game.slug === 'uno') {
    return <UnoRoomGate roomId={roomId} />
  }

  return (
    <CenteredState
      description={`${game.title} 已在总览中注册，但还没有实现具体页面。`}
      title="游戏即将开放"
    />
  )
}

function CenteredState({ title, description }: { title: string, description: string }) {
  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
          <Button asChild className="mt-4 w-fit" variant="outline">
            <Link to="/">
              <ArrowLeft className="size-4" />
              返回总览
            </Link>
          </Button>
        </CardHeader>
      </Card>
    </main>
  )
}
