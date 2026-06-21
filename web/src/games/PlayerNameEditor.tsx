import { Check, Pencil } from 'lucide-react'
import { useState } from 'react'
import { usePendingAction } from '@/games/usePendingAction'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'

interface PlayerNameEditorProps {
  buttonClassName?: string
  className?: string
  name: string
  onSave: (name: string) => Promise<void>
}

export function PlayerNameEditor({ buttonClassName, className, name, onSave }: PlayerNameEditorProps) {
  const { t } = useI18n()
  const [draft, setDraft] = useState(name)
  const [editing, setEditing] = useState(false)
  const pending = usePendingAction()
  const isSaving = pending.isPending('save')

  if (!editing) {
    return (
      <button
        aria-label={t('common.editName')}
        className={cn('inline-grid size-8 shrink-0 place-items-center rounded-lg border border-white/20 bg-white/10 transition hover:bg-white/20', buttonClassName)}
        title={t('common.editName')}
        type="button"
        onClick={() => {
          setDraft(name)
          setEditing(true)
        }}
      >
        <Pencil className="size-4" />
      </button>
    )
  }

  async function saveName() {
    const nextName = draft.trim()
    if (!nextName || isSaving) {
      return
    }

    await pending.run('save', async () => {
      await onSave(nextName)
      setEditing(false)
    })
  }

  return (
    <form
      className={cn('grid min-w-[150px] grid-cols-[minmax(0,1fr)_auto] items-center gap-1', className)}
      onSubmit={(event) => {
        event.preventDefault()
        void saveName()
      }}
    >
      <input
        aria-label={t('common.displayName')}
        className="min-h-8 min-w-0 rounded-lg border border-white/25 bg-black/25 px-2 text-sm font-bold outline-none focus:ring-2 focus:ring-white/50"
        maxLength={24}
        value={draft}
        onChange={event => setDraft(event.currentTarget.value)}
      />
      <button
        aria-label={t('common.saveName')}
        className={cn('inline-grid size-8 place-items-center rounded-lg border border-white/20 bg-white/14 transition hover:bg-white/24 disabled:cursor-not-allowed disabled:opacity-50', buttonClassName)}
        disabled={isSaving || !draft.trim()}
        title={isSaving ? t('common.syncing') : t('common.saveName')}
        type="submit"
      >
        {isSaving ? <span className="text-xs font-black">...</span> : <Check className="size-4" />}
      </button>
    </form>
  )
}
