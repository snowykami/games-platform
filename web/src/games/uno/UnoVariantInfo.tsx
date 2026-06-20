import type { MouseEvent } from 'react'
import { Info } from 'lucide-react'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { getUnoVariantOption } from './variantInfo'

export function UnoVariantInfoButton({ className, variantKey }: { className?: string, variantKey?: string }) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const variant = getUnoVariantOption(variantKey ?? 'classic', t)

  function keepDialogOpen(event: MouseEvent<HTMLDivElement>) {
    event.stopPropagation()
  }

  return (
    <>
      <button
        aria-label={t('uno.variantInfo')}
        className={cn(
          'inline-flex min-h-8 items-center justify-center gap-1.5 rounded-full border border-white/28 bg-[#141310]/50 px-2.5 text-xs font-black text-[#fff8e8] transition hover:border-[#f3c33c] hover:bg-[#f3c33c]/12',
          className,
        )}
        type="button"
        onClick={() => setOpen(true)}
      >
        <Info className="size-4" />
        <span>{variant.name}</span>
      </button>
      {open && (
        <div
          className="fixed inset-0 z-50 grid place-items-center bg-[#090807]/62 px-4 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-label={t('uno.variantInfo')}
          onClick={() => setOpen(false)}
        >
          <div
            className="w-[min(420px,calc(100vw-32px))] rounded-lg border border-white/25 bg-[#14110e] p-5 text-[#fff8e8] shadow-[0_28px_80px_rgba(0,0,0,0.45)]"
            onClick={keepDialogOpen}
          >
            <div className="flex items-center gap-2">
              <Info className="size-5 text-[#f3c33c]" />
              <strong className="text-xl">{variant.name}</strong>
            </div>
            <p className="mt-3 text-sm leading-7 text-[#fff8e8]/78">{variant.description}</p>
            <p className="mt-4 text-xs font-black text-[#fff8e8]/55">{t('uno.variantCloseHint')}</p>
          </div>
        </div>
      )}
    </>
  )
}
