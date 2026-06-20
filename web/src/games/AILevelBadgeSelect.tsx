import type { AILevel } from './ai'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { AI_LEVELS, getAILevelLabel, normalizeAILevel } from './ai'

interface AILevelBadgeSelectProps {
  className?: string
  disabled?: boolean
  level: string | undefined
  llmEnabled: boolean
  onChange?: (level: AILevel) => void
  palette?: 'dark' | 'gomoku' | 'mahjong' | 'xiangqi'
}

export function AILevelBadgeSelect({ className, disabled = false, level, llmEnabled, onChange, palette = 'dark' }: AILevelBadgeSelectProps) {
  const { locale, t } = useI18n()
  const value = normalizeAILevel(level)

  return (
    <select
      aria-label={t('common.aiStrength')}
      className={cn(
        'h-7 rounded-full border px-2 text-xs font-black outline-none transition',
        paletteClass(palette),
        disabled && 'cursor-default appearance-none opacity-100',
        className,
      )}
      disabled={disabled}
      value={value}
      onChange={event => onChange?.(event.target.value as AILevel)}
    >
      {AI_LEVELS.map(option => (
        <option key={option} className="text-[#171411]" disabled={option === 'ai' && !llmEnabled} value={option}>
          {getAILevelLabel(option, locale)}
        </option>
      ))}
    </select>
  )
}

function paletteClass(palette: NonNullable<AILevelBadgeSelectProps['palette']>) {
  const styles = {
    dark: 'border-[#fff8e8]/45 bg-[#fff8e8] text-[#171411] hover:border-[#fff8e8]',
    gomoku: 'border-[#f4f0e4]/45 bg-[#f4f0e4] text-[#101714] hover:border-[#f4f0e4]',
    mahjong: 'border-[#ffd166]/65 bg-[#ffd166] text-[#143128] hover:border-[#fff8e8]',
    xiangqi: 'border-[#f2d59a]/60 bg-[#fff8e8] text-[#202018] hover:border-[#f2d59a]',
  }
  return styles[palette]
}
