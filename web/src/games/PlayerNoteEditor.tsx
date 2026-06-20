import { Check } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'

interface PlayerNoteEditorProps {
  className?: string
  note?: string
  onSave: (note: string) => Promise<void>
}

export function PlayerNoteEditor({ className, note = '', onSave }: PlayerNoteEditorProps) {
  const { t } = useI18n()
  const [draft, setDraft] = useState(note)
  const [pending, setPending] = useState(false)

  async function saveNote() {
    if (pending || draft === note) {
      return
    }

    const nextNote = normalizeNoteDraft(draft)
    setPending(true)
    try {
      await onSave(nextNote)
      setDraft(nextNote)
    }
    finally {
      setPending(false)
    }
  }

  return (
    <form
      className={cn('grid grid-cols-[minmax(0,1fr)_auto] items-center gap-1', className)}
      onSubmit={(event) => {
        event.preventDefault()
        void saveNote()
      }}
    >
      <input
        aria-label={t('social.playerNote')}
        className="min-h-8 min-w-0 rounded-lg border border-white/16 bg-black/24 px-2 text-xs font-bold text-[#fff8e8] outline-none placeholder:text-[#fff8e8]/42 focus:ring-2 focus:ring-white/35"
        maxLength={80}
        placeholder={t('social.notePlaceholder')}
        value={draft}
        onChange={event => setDraft(event.currentTarget.value)}
      />
      <button
        aria-label={t('social.saveNote')}
        className="inline-grid size-8 place-items-center rounded-lg border border-white/16 bg-white/10 transition hover:bg-white/20 disabled:cursor-not-allowed disabled:opacity-50"
        disabled={pending || draft === note}
        title={t('social.saveNote')}
        type="submit"
      >
        <Check className="size-4" />
      </button>
    </form>
  )
}

function normalizeNoteDraft(value: string) {
  const note = value.trim().split(/\s+/).filter(Boolean).join(' ')
  return Array.from(note).slice(0, 80).join('')
}
