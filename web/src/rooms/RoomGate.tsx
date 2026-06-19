import type { FormEvent, ReactNode } from 'react'
import type { GameDefinition } from '@/games/registry'
import { ArrowLeft, Copy, DoorOpen, Play, Plus, Users } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { Badge } from '@/shared/components/ui/badge'
import { Button } from '@/shared/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/components/ui/card'

interface RoomGateProps {
  game: GameDefinition
  onStart: () => void
  roomId?: string
}

export function RoomGate({ game, onStart, roomId }: RoomGateProps) {
  const navigate = useNavigate()
  const [joinCode, setJoinCode] = useState(roomId ?? '')
  const [message, setMessage] = useState('创建房间后复制链接，朋友打开同一链接即可进入准备页。')

  function createRoom() {
    const nextRoomId = createRoomId()

    navigate(`/games/${game.slug}?room=${nextRoomId}`)
    setJoinCode(nextRoomId)
    setMessage('房间已创建，可以复制链接邀请玩家。')
  }

  function joinRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const normalizedCode = joinCode.trim().toUpperCase()

    if (!normalizedCode) {
      setMessage('请输入房间号。')
      return
    }

    navigate(`/games/${game.slug}?room=${encodeURIComponent(normalizedCode)}`)
    setMessage('已进入房间准备页。')
  }

  function copyLink() {
    navigator.clipboard?.writeText(window.location.href)
    setMessage('链接已复制。')
  }

  return (
    <main className="min-h-svh bg-background">
      <section className="border-b bg-card">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-5 sm:px-6 lg:px-8">
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Badge variant="secondary">{game.supportsOnline ? '联机房间' : '本地游戏'}</Badge>
              <Badge variant="outline">
                {game.minPlayers}
                -
                {game.maxPlayers}
                人
              </Badge>
            </div>
            <h1 className="text-2xl font-semibold tracking-normal text-foreground sm:text-3xl">
              进入
              {' '}
              {game.title}
              {' '}
              房间
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              先创建或加入房间，再进入游戏桌。这个入口会作为后续象棋、麻将、Uno 等联机游戏的统一流程。
            </p>
          </div>
          <Button asChild variant="outline">
            <Link to="/">
              <ArrowLeft className="size-4" />
              游戏大厅
            </Link>
          </Button>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-4 px-4 py-5 sm:px-6 md:grid-cols-[minmax(0,1fr)_360px] lg:px-8">
        <Card className="overflow-hidden">
          <CardHeader className="border-b bg-muted/30">
            <CardTitle className="flex items-center gap-2 text-xl">
              <Users className="size-5" />
              {roomId ? `房间 ${roomId}` : '创建或加入房间'}
            </CardTitle>
            <CardDescription>{message}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 p-5">
            {roomId
              ? (
                  <RoomReadyPanel game={game} onCopy={copyLink} onStart={onStart} roomId={roomId} />
                )
              : (
                  <RoomCreateJoinPanel
                    joinCode={joinCode}
                    setJoinCode={setJoinCode}
                    onCreate={createRoom}
                    onJoin={joinRoom}
                  />
                )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>房间规则</CardTitle>
            <CardDescription>所有联机游戏统一先进入房间准备页。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <RuleLine>无房间号：展示创建/加入入口。</RuleLine>
            <RuleLine>有房间号：展示房间准备页和玩家列表。</RuleLine>
            <RuleLine>开始游戏：由房主触发，后续接入 WebSocket 服务端权威状态。</RuleLine>
            <RuleLine>AI 玩家：作为正式玩家类型，可由房主添加。</RuleLine>
          </CardContent>
        </Card>
      </section>
    </main>
  )
}

function RoomCreateJoinPanel({
  joinCode,
  onCreate,
  onJoin,
  setJoinCode,
}: {
  joinCode: string
  onCreate: () => void
  onJoin: (event: FormEvent<HTMLFormElement>) => void
  setJoinCode: (value: string) => void
}) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <div className="rounded-lg border bg-background p-4">
        <div className="mb-4 flex size-11 items-center justify-center rounded-md bg-primary/10 text-primary">
          <Plus className="size-5" />
        </div>
        <h2 className="font-semibold text-foreground">创建房间</h2>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          生成房间号并进入准备页，之后复制链接邀请其他玩家。
        </p>
        <Button className="mt-4 w-full" type="button" onClick={onCreate}>
          创建房间
        </Button>
      </div>

      <form className="rounded-lg border bg-background p-4" onSubmit={onJoin}>
        <div className="mb-4 flex size-11 items-center justify-center rounded-md bg-primary/10 text-primary">
          <DoorOpen className="size-5" />
        </div>
        <h2 className="font-semibold text-foreground">加入房间</h2>
        <label className="mt-3 block text-sm font-medium text-foreground" htmlFor="room-code">
          房间号
        </label>
        <input
          id="room-code"
          className="mt-2 h-10 w-full rounded-md border border-input bg-background px-3 text-sm uppercase text-foreground outline-none focus:ring-2 focus:ring-ring"
          placeholder="ROOM42"
          value={joinCode}
          onChange={event => setJoinCode(event.target.value)}
        />
        <Button className="mt-4 w-full" type="submit" variant="secondary">
          加入房间
        </Button>
      </form>
    </div>
  )
}

function RoomReadyPanel({
  game,
  onCopy,
  onStart,
  roomId,
}: {
  game: GameDefinition
  onCopy: () => void
  onStart: () => void
  roomId: string
}) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-3 sm:grid-cols-3">
        <PlayerSlot label="房主" name="你" />
        <PlayerSlot label="AI" name="北风" />
        <PlayerSlot label="AI" name="南星" />
      </div>

      <div className="flex flex-wrap gap-2 rounded-lg border bg-muted/40 p-3 text-sm text-muted-foreground">
        <span>
          房间号：
          <strong className="text-foreground">{roomId}</strong>
        </span>
        <span>
          游戏：
          <strong className="text-foreground">{game.title}</strong>
        </span>
        <span>
          当前人数：
          <strong className="text-foreground">3</strong>
        </span>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row">
        <Button className="sm:w-40" type="button" variant="outline" onClick={onCopy}>
          <Copy className="size-4" />
          复制链接
        </Button>
        <Button className="sm:w-40" type="button" onClick={onStart}>
          <Play className="size-4" />
          开始游戏
        </Button>
      </div>
    </div>
  )
}

function PlayerSlot({ label, name }: { label: string, name: string }) {
  return (
    <div className="rounded-lg border bg-background p-4">
      <div className="flex items-center justify-between gap-2">
        <strong className="text-foreground">{name}</strong>
        <Badge variant={label === 'AI' ? 'secondary' : 'default'}>{label}</Badge>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">已准备</p>
    </div>
  )
}

function RuleLine({ children }: { children: ReactNode }) {
  return <p className="rounded-md bg-muted px-3 py-2">{children}</p>
}

function createRoomId() {
  return Math.random().toString(36).slice(2, 8).toUpperCase()
}
