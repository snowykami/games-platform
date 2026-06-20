import type { ReactNode } from 'react'
import type { I18nContextValue } from './context'
import type { Locale } from './messages'
import { useMemo, useState } from 'react'
import { I18nContext } from './context'
import { detectLocale } from './locale'
import { messages } from './messages'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState(() => detectLocale())

  const value = useMemo<I18nContextValue>(() => ({
    locale,
    setLocale: (nextLocale) => {
      window.localStorage.setItem('liteyuki.locale', nextLocale)
      document.documentElement.lang = nextLocale === 'zh' ? 'zh-CN' : 'en'
      setLocale(nextLocale)
    },
    t: (key, params) => formatMessage(resolveMessage(locale, key), params),
    ta: key => resolveMessageArray(locale, key),
  }), [locale])

  return <I18nContext value={value}>{children}</I18nContext>
}

function resolveMessage(locale: Locale, key: string): string {
  const value = resolveValue(messages[locale], key) ?? resolveValue(messages.en, key)
  return typeof value === 'string' ? value : key
}

function resolveMessageArray(locale: Locale, key: string): string[] {
  const value = resolveValue(messages[locale], key) ?? resolveValue(messages.en, key)
  return Array.isArray(value) ? [...value] : []
}

function resolveValue(source: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((current, part) => {
    if (current && typeof current === 'object' && part in current) {
      return (current as Record<string, unknown>)[part]
    }
    return undefined
  }, source)
}

function formatMessage(message: string, params?: Record<string, string | number>) {
  if (!params) {
    return message
  }
  return message.replace(/\{(\w+)\}/g, (_, key: string) => String(params[key] ?? `{${key}}`))
}
