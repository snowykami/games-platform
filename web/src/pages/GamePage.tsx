import { ArrowLeft } from 'lucide-react'
import { Link, useParams, useSearchParams } from 'react-router'
import { GomokuRoomGate } from '@/games/gomoku/GomokuRoomGate'
import { MahjongRoomGate } from '@/games/mahjong/MahjongRoomGate'
import { findGame } from '@/games/registry'
import { UnoRoomGate } from '@/games/uno/UnoRoomGate'
import { XiangqiRoomGate } from '@/games/xiangqi/XiangqiRoomGate'
import { useI18n } from '@/i18n/context'
import { gameIconHref, useDocumentMeta } from '@/shared/brand/meta'
import { Button } from '@/shared/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/shared/components/ui/card'

export function GamePage() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const { t } = useI18n()
  const roomId = searchParams.get('room')?.trim() || undefined
  const game = slug ? findGame(slug) : undefined
  useDocumentMeta(game?.title ?? t('catalog.notFoundTitle'), gameIconHref(game?.slug))

  if (!game) {
    return (
      <CenteredState
        description={t('catalog.notFoundDescription')}
        title={t('catalog.notFoundTitle')}
      />
    )
  }

  if (game.slug === 'uno') {
    return <UnoRoomGate roomId={roomId} />
  }

  if (game.slug === 'gomoku') {
    return <GomokuRoomGate roomId={roomId} />
  }

  if (game.slug === 'mahjong') {
    return <MahjongRoomGate roomId={roomId} />
  }

  if (game.slug === 'xiangqi') {
    return <XiangqiRoomGate roomId={roomId} />
  }

  return (
    <CenteredState
      title={t('catalog.comingSoonTitle')}
      description={t('catalog.comingSoonDescription', { title: game.title })}
    />
  )
}

function CenteredState({ title, description }: { title: string, description: string }) {
  const { t } = useI18n()
  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
          <Button asChild className="mt-4 w-fit" variant="outline">
            <Link to="/">
              <ArrowLeft className="size-4" />
              {t('catalog.back')}
            </Link>
          </Button>
        </CardHeader>
      </Card>
    </main>
  )
}
