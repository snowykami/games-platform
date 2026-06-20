import type { Locale } from './messages'

export function getCurrentLocale(): Locale {
  return detectLocale()
}

export function detectLocale(): Locale {
  const stored = window.localStorage.getItem('liteyuki.locale')
  if (stored === 'zh' || stored === 'en') {
    document.documentElement.lang = stored === 'zh' ? 'zh-CN' : 'en'
    return stored
  }
  const browserLocale = window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  document.documentElement.lang = browserLocale === 'zh' ? 'zh-CN' : 'en'
  return browserLocale
}
