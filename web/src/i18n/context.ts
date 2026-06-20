import type { Locale } from './messages'
import { createContext, use } from 'react'

export interface I18nContextValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string, params?: Record<string, string | number>) => string
  ta: (key: string) => string[]
}

export const I18nContext = createContext<I18nContextValue | null>(null)

export function useI18n() {
  const context = use(I18nContext)
  if (!context) {
    throw new Error('useI18n must be used inside I18nProvider')
  }
  return context
}
