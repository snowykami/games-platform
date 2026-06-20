export async function fetchWithAuthRedirect(input: RequestInfo | URL, init?: RequestInit) {
  const response = await fetch(input, init)
  if (response.status === 401 && window.location.pathname !== '/login') {
    const next = `${window.location.pathname}${window.location.search}`
    window.location.assign(`/login?next=${encodeURIComponent(next)}`)
  }
  return response
}
