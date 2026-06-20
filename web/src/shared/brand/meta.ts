import { useEffect } from 'react'

const SITE_NAME = 'Liteyuki Games'
const SITE_ICON = '/favicon.svg'

export function formatPageTitle(title?: string) {
  return title ? `${title} | ${SITE_NAME}` : SITE_NAME
}

export function gameIconHref(slug?: string) {
  return slug ? `/game-icons/${slug}.svg` : SITE_ICON
}

export function useDocumentMeta(title?: string, iconHref = SITE_ICON) {
  useEffect(() => {
    document.title = formatPageTitle(title)
    setFavicon(iconHref)
  }, [iconHref, title])
}

function setFavicon(iconHref: string) {
  let icon = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!icon) {
    icon = document.createElement('link')
    icon.rel = 'icon'
    document.head.append(icon)
  }

  icon.type = 'image/svg+xml'
  icon.href = iconHref
}
