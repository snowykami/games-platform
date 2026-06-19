import type { GameDefinition } from '@/games/registry'
import { z } from 'zod'
import { localGames } from '@/games/registry'

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
    const response = await fetch('/api/games')

    if (!response.ok) {
      throw new Error(`Failed to load games: ${response.status}`)
    }

    const data = gamesResponseSchema.parse(await response.json())
    return data.games
  }
  catch {
    return localGames
  }
}
