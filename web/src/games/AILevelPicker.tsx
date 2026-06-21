import type { AILevel } from './ai'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { AI_LEVELS, getAILevelDescription, getAILevelLabel } from './ai'

interface AILevelPickerProps {
  className?: string
  level: string
  llmEnabled: boolean
  onChange: (level: AILevel) => void
  palette?: 'dark' | 'gomoku' | 'xiangqi'
}

export function AILevelPicker({ className, level, llmEnabled, onChange, palette = 'dark' }: AILevelPickerProps) {
  const { locale, t } = useI18n()

  return (
    <label className={cn('grid min-w-40 gap-1 text-xs font-black', paletteClass(palette, 'frame'), className)}>
      <span>{t('common.aiStrength')}</span>
      <select
        className={cn('min-h-10 rounded-lg border px-3 text-sm font-black outline-none transition', paletteClass(palette, 'button'))}
        title={getAILevelDescription(level as AILevel, locale)}
        value={level}
        onChange={event => onChange(event.target.value as AILevel)}
      >
        {AI_LEVELS.map(option => (
          <option key={option} className="text-[#171411]" disabled={option === 'ai' && !llmEnabled} value={option}>
            {getAILevelLabel(option, locale)}
          </option>
        ))}
      </select>
    </label>
  )
}

function paletteClass(palette: NonNullable<AILevelPickerProps['palette']>, part: 'button' | 'frame') {
  const styles = {
    dark: {
      button: 'border-white/18 bg-[#141310]/48 text-[#fff8e8] hover:border-[#fff8e8]/60',
      frame: 'text-[#fff8e8]',
    },
    gomoku: {
      button: 'border-white/18 bg-[#101714]/55 text-[#f4f0e4] hover:border-[#f4f0e4]/60',
      frame: 'text-[#f4f0e4]',
    },
    xiangqi: {
      button: 'border-[#fff8e8]/18 bg-[#10100d]/58 text-[#fff8e8] hover:border-[#f2d59a]/70',
      frame: 'text-[#fff8e8]',
    },
  }
  return styles[palette][part]
}
