import { MessageCircle, Send, X } from 'lucide-react'
import { useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'

interface SpeechBubbleProps {
  className?: string
  text?: string
}

interface SpeechButtonProps {
  className?: string
  disabled?: boolean
  onSend: (text: string) => void | Promise<void>
  palette?: 'dark' | 'gomoku' | 'mahjong' | 'xiangqi'
}

export function SpeechBubble({ className, text }: SpeechBubbleProps) {
  if (!text) {
    return null
  }

  return (
    <p className={cn('max-w-52 rounded-lg border border-white/18 bg-[#fff8e8] px-2.5 py-1.5 text-xs font-black leading-5 text-[#171411] shadow-[0_12px_30px_rgba(0,0,0,0.22)]', className)}>
      {text}
    </p>
  )
}

export function SpeechButton({ className, disabled = false, onSend, palette = 'dark' }: SpeechButtonProps) {
  const { t } = useI18n()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const [open, setOpen] = useState(false)
  const [panelPosition, setPanelPosition] = useState({ left: 16, top: 16 })
  const [text, setText] = useState('')

  useLayoutEffect(() => {
    if (!open || !buttonRef.current) {
      return
    }

    const positionPanel = () => {
      const rect = buttonRef.current?.getBoundingClientRect()
      if (!rect) {
        return
      }
      const panelWidth = Math.min(320, window.innerWidth - 24)
      const panelHeight = 214
      const gap = 10
      const left = Math.min(Math.max(12, rect.right - panelWidth), window.innerWidth - panelWidth - 12)
      const hasRoomBelow = rect.bottom + gap + panelHeight <= window.innerHeight - 12
      const top = hasRoomBelow ? rect.bottom + gap : Math.max(12, rect.top - panelHeight - gap)
      setPanelPosition({ left, top })
    }

    const frame = window.requestAnimationFrame(positionPanel)
    window.addEventListener('resize', positionPanel)
    window.addEventListener('scroll', positionPanel, true)
    return () => {
      window.cancelAnimationFrame(frame)
      window.removeEventListener('resize', positionPanel)
      window.removeEventListener('scroll', positionPanel, true)
    }
  }, [open])

  async function send() {
    const nextText = text.trim()
    if (!nextText) {
      return
    }
    await onSend(nextText)
    setText('')
    setOpen(false)
  }

  return (
    <div className={cn('relative inline-flex', className)}>
      <button
        ref={buttonRef}
        aria-label={t('common.speak')}
        className={cn('grid size-7 place-items-center rounded-full border transition', paletteClass(palette, 'button'))}
        disabled={disabled}
        type="button"
        onClick={() => setOpen(current => !current)}
      >
        <MessageCircle className="size-4" />
      </button>
      {open && createPortal(
        <div
          className={cn('fixed z-[9999] grid w-[min(320px,calc(100vw-24px))] gap-2 rounded-lg border p-2 shadow-[0_20px_60px_rgba(0,0,0,0.45)]', paletteClass(palette, 'panel'))}
          style={{ left: panelPosition.left, top: panelPosition.top }}
        >
          <div className="flex items-center justify-between gap-2">
            <strong className="text-xs">{t('common.speak')}</strong>
            <button aria-label={t('common.close')} className="grid size-6 place-items-center rounded-full hover:bg-white/10" type="button" onClick={() => setOpen(false)}>
              <X className="size-3.5" />
            </button>
          </div>
          <textarea
            autoFocus
            className={cn('min-h-24 resize-none rounded-md border bg-black/20 p-2 text-sm font-bold outline-none', paletteClass(palette, 'input'))}
            maxLength={120}
            placeholder={t('common.speechPlaceholder')}
            value={text}
            onChange={event => setText(event.target.value)}
          />
          <button className={cn('inline-flex min-h-9 items-center justify-center gap-2 rounded-md px-3 text-sm font-black', paletteClass(palette, 'send'))} type="button" onClick={send}>
            <Send className="size-4" />
            {t('common.send')}
          </button>
        </div>,
        document.body,
      )}
    </div>
  )
}

function paletteClass(palette: NonNullable<SpeechButtonProps['palette']>, part: 'button' | 'input' | 'panel' | 'send') {
  const styles = {
    dark: {
      button: 'border-white/25 bg-[#141310]/50 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#171411]',
      input: 'border-white/20 text-[#fff8e8]',
      panel: 'border-white/24 bg-[#171411] text-[#fff8e8]',
      send: 'bg-[#fff8e8] text-[#171411]',
    },
    gomoku: {
      button: 'border-white/22 bg-[#0b1110]/60 text-[#f4f0e4] hover:bg-[#f4f0e4] hover:text-[#101714]',
      input: 'border-white/20 text-[#f4f0e4]',
      panel: 'border-white/24 bg-[#101714] text-[#f4f0e4]',
      send: 'bg-[#f4f0e4] text-[#101714]',
    },
    mahjong: {
      button: 'border-[#d8b66a]/50 bg-[#10251f] text-[#fff8e8] hover:bg-[#ffd166] hover:text-[#143128]',
      input: 'border-[#d8b66a]/35 text-[#fff8e8]',
      panel: 'border-[#d8b66a]/45 bg-[#10251f] text-[#fff8e8]',
      send: 'bg-[#ffd166] text-[#143128]',
    },
    xiangqi: {
      button: 'border-[#fff8e8]/22 bg-[#10100d]/60 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#202018]',
      input: 'border-[#fff8e8]/20 text-[#fff8e8]',
      panel: 'border-[#fff8e8]/24 bg-[#202018] text-[#fff8e8]',
      send: 'bg-[#f2d59a] text-[#202018]',
    },
  }
  return styles[palette][part]
}
