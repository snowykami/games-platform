export interface GameDefinition {
  slug: string
  title: string
  description: string
  minPlayers: number
  maxPlayers: number
  supportsOnline: boolean
  supportsLocal: boolean
  status: 'available' | 'coming-soon'
  tags: string[]
}

export const localGames: GameDefinition[] = [
  {
    slug: 'uno',
    title: 'Uno',
    description: '',
    minPlayers: 2,
    maxPlayers: 4,
    supportsOnline: true,
    supportsLocal: true,
    status: 'available',
    tags: [],
  },
  {
    slug: 'gomoku',
    title: 'Gomoku',
    description: '',
    minPlayers: 2,
    maxPlayers: 2,
    supportsOnline: true,
    supportsLocal: true,
    status: 'available',
    tags: [],
  },
  {
    slug: 'xiangqi',
    title: 'Xiangqi',
    description: '',
    minPlayers: 2,
    maxPlayers: 2,
    supportsOnline: true,
    supportsLocal: true,
    status: 'available',
    tags: [],
  },
  {
    slug: 'mahjong',
    title: 'Mahjong',
    description: '',
    minPlayers: 4,
    maxPlayers: 4,
    supportsOnline: true,
    supportsLocal: true,
    status: 'available',
    tags: [],
  },
]

export function findGame(slug: string) {
  return localGames.find(game => game.slug === slug)
}
