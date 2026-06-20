export function getUnoVariantLabel(key: string | undefined, t: (key: string) => string) {
  return key ? getUnoVariantOption(key, t).name : t('uno.classicVariant')
}

export function getUnoThemeLabel(key: string | undefined, t: (key: string) => string) {
  return key ? getUnoThemeOption(key, t).name : t('uno.classicTheme')
}

export function getUnoVariantOption(key: string, t: (key: string) => string) {
  const normalizedKey = key.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase())
  return {
    description: t(`uno.variants.${normalizedKey}.description`),
    key,
    name: t(`uno.variants.${normalizedKey}.name`),
  }
}

export function getUnoThemeOption(key: string, t: (key: string) => string) {
  const normalizedKey = key.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase())
  return {
    description: t(`uno.themes.${normalizedKey}.description`),
    key,
    name: t(`uno.themes.${normalizedKey}.name`),
  }
}
