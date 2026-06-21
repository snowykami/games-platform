import { DoorOpen } from 'lucide-react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'

export interface ContinueRoomSummary {
  id: string
  phase: string
  players?: unknown[]
}

interface ContinueRoomEntryProps {
  buttonClassName: string
  className: string
  onEnter: () => void
  room: ContinueRoomSummary
}

export function ContinueRoomEntry({ buttonClassName, className, onEnter, room }: ContinueRoomEntryProps) {
  const { t } = useI18n()
  return (
    <section className={cn('grid gap-3 rounded-lg border p-4', className)}>
      <div className="grid gap-1">
        <p className="text-xs font-black uppercase tracking-normal opacity-70">{t('room.continueTitle')}</p>
        <h3 className="text-xl font-black tracking-normal">
          {t('common.room')}
          {' '}
          {room.id}
        </h3>
        <p className="text-sm font-bold opacity-75">{t('room.continueDescription')}</p>
      </div>
      <div className="flex flex-wrap items-center gap-2 text-xs font-black opacity-80">
        <span className="rounded-full border border-current/25 px-2 py-1 uppercase">{room.phase}</span>
        {room.players && (
          <span className="rounded-full border border-current/25 px-2 py-1">
            {room.players.length}
            {' '}
            {t('common.player')}
          </span>
        )}
      </div>
      <button className={buttonClassName} type="button" onClick={onEnter}>
        <DoorOpen className="size-4" />
        {t('room.continueButton')}
      </button>
    </section>
  )
}
