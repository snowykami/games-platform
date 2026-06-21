import type { ReactNode } from 'react'
import type { SocialGameSlug } from './online'
import type { GAME_COPY } from './socialTheme'
import { Skull, Vote } from 'lucide-react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { usePendingAction } from '../usePendingAction'
import { socialButton } from './socialStyle'
import { Panel } from './socialUi'

export function ChoiceButton({
  children,
  config,
  disabled = false,
  icon,
  onClick,
  selected,
}: {
  children: ReactNode
  config: typeof GAME_COPY[SocialGameSlug]
  disabled?: boolean
  icon?: ReactNode
  onClick: () => void
  selected: boolean
}) {
  return (
    <button
      className={cn(socialButton(config), selected && 'ring-2 ring-[#fff8e8] ring-offset-2 ring-offset-black/40')}
      disabled={disabled}
      type="button"
      onClick={onClick}
    >
      {icon}
      {children}
    </button>
  )
}

export function ConfirmChoiceButton({
  config,
  disabled,
  label,
  onConfirm,
  selectedLabel,
}: {
  config: typeof GAME_COPY[SocialGameSlug]
  disabled: boolean
  label: string
  onConfirm: () => void | Promise<void>
  selectedLabel?: ReactNode
}) {
  const { t } = useI18n()
  const pending = usePendingAction()
  const isSubmitting = pending.isPending('confirm')

  return (
    <div className="mt-1 grid gap-2 rounded-lg border border-white/12 bg-black/18 p-2">
      <p className="flex min-h-5 flex-wrap items-center gap-1.5 text-xs font-black text-[#fff8e8]/65">
        {selectedLabel
          ? (
              <>
                {t('social.selectedChoice', { name: '' })}
                {selectedLabel}
              </>
            )
          : t('social.selectBeforeConfirm')}
      </p>
      <button className={socialButton(config, true)} disabled={disabled || isSubmitting} type="button" onClick={() => void pending.run('confirm', onConfirm, { releaseOnSettle: false })}>
        <Vote className="size-4" />
        {isSubmitting ? t('common.syncing') : label}
      </button>
    </div>
  )
}

export function SubmittedNotice({ config, label }: { config: typeof GAME_COPY[SocialGameSlug], label: string }) {
  return (
    <div className={cn('rounded-lg px-3 py-2 text-sm font-black', config.primary)}>
      {label}
    </div>
  )
}

export function DeadPlayerPanel({
  config,
}: {
  config: typeof GAME_COPY[SocialGameSlug]
}) {
  const { t } = useI18n()

  return (
    <Panel config={config}>
      <Skull className="size-6 text-[#ff9aa8]" />
      <h2 className="text-xl font-black">{t('social.outSpectatorTitle')}</h2>
      <p className="text-sm leading-6 text-[#fff8e8]/76">{t('social.outSpectatorHint')}</p>
    </Panel>
  )
}
