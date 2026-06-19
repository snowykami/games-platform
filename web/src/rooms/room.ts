export function getRoomFromSearch(search: string) {
  const params = new URLSearchParams(search)
  const room = params.get('room')?.trim()

  return room || undefined
}

export function buildRoomUrl(slug: string, room: string) {
  const url = new URL(window.location.href)
  url.pathname = `/games/${slug}`
  url.search = ''
  url.searchParams.set('room', room)

  return url.toString()
}
