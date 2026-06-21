import type { GameDefinition } from '@/games/registry'
import { z } from 'zod'
import { fetchWithAuthRedirect } from '@/auth/fetch'
import { localGames } from '@/games/registry'
import { localizeGames } from '@/i18n/gameText'
import { getCurrentLocale } from '@/i18n/locale'

const gameDefinitionSchema = z.object({
  slug: z.string(),
  title: z.string(),
  description: z.string(),
  minPlayers: z.number(),
  maxPlayers: z.number(),
  supportsOnline: z.boolean(),
  supportsLocal: z.boolean(),
  status: z.enum(['available', 'coming-soon']),
  tags: z.array(z.string()),
})

const gamesResponseSchema = z.object({
  games: z.array(gameDefinitionSchema),
})

export async function getGames(): Promise<GameDefinition[]> {
  try {
    const response = await fetch('/api/games', { headers: { 'Accept-Language': getCurrentLocale() } })

    if (!response.ok) {
      throw new Error(`Failed to load games: ${response.status}`)
    }

    const data = gamesResponseSchema.parse(await response.json())
    return localizeGames(data.games, getCurrentLocale())
  }
  catch {
    return localizeGames(localGames, getCurrentLocale())
  }
}

export async function recordGameUsage(slug: string) {
  const response = await fetchWithAuthRedirect(`/api/games/${encodeURIComponent(slug)}/usage`, { method: 'POST' })
  if (!response.ok) {
    throw new Error(`Failed to record game usage: ${response.status}`)
  }
}
