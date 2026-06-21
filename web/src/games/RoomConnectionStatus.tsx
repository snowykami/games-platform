import type { RoomConnectionInfo } from './useRoomSocket'
import { Wifi } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

interface RoomConnectionStatusProps {
  className?: string
  connection?: RoomConnectionInfo
}

export function RoomConnectionStatus({ className, connection }: RoomConnectionStatusProps) {
  const state = connection?.state ?? 'disconnected'
  const latencyMs = connection?.latencyMs
  const connected = state === 'connected'
  const tier = connected && latencyMs !== undefined
    ? latencyMs <= 50 ? 'fast' : latencyMs <= 150 ? 'slow' : 'bad'
    : 'idle'
  const label = connected
    ? latencyMs === undefined ? '测延迟' : `${latencyMs}ms`
    : state === 'reconnecting' ? '重连中' : state === 'connecting' ? '连接中' : '已断线'

  return (
    <span
      className={cn(
        'inline-flex min-h-7 shrink-0 items-center gap-1.5 rounded-full border px-2 text-[11px] font-black leading-none opacity-82',
        tier === 'fast' && 'border-emerald-300/30 bg-emerald-400/12 text-emerald-100',
        tier === 'slow' && 'border-amber-300/35 bg-amber-400/14 text-amber-100',
        tier === 'bad' && 'border-rose-300/40 bg-rose-400/15 text-rose-100',
        tier === 'idle' && 'border-white/14 bg-white/8 text-white/62',
        className,
      )}
      title="WebSocket 延迟"
    >
      <Wifi className="size-3" />
      {label}
    </span>
  )
}
