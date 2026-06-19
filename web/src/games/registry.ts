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
    description: '轻量本地 Uno 原型，先验证卡牌规则、回合流转和页面结构。',
    minPlayers: 2,
    maxPlayers: 4,
    supportsOnline: true,
    supportsLocal: true,
    status: 'available',
    tags: ['卡牌', '回合制', '首个原型'],
  },
  {
    slug: 'gomoku',
    title: '五子棋',
    description: '经典双人棋类，适合作为首个完整联机规则验证目标。',
    minPlayers: 2,
    maxPlayers: 2,
    supportsOnline: true,
    supportsLocal: true,
    status: 'coming-soon',
    tags: ['棋类', '双人'],
  },
  {
    slug: 'xiangqi',
    title: '象棋',
    description: '中国象棋，后续通过独立适配器接入。',
    minPlayers: 2,
    maxPlayers: 2,
    supportsOnline: true,
    supportsLocal: true,
    status: 'coming-soon',
    tags: ['棋类', '策略'],
  },
  {
    slug: 'mahjong',
    title: '麻将',
    description: '多人牌桌游戏，规则复杂度较高，后续分阶段实现。',
    minPlayers: 4,
    maxPlayers: 4,
    supportsOnline: true,
    supportsLocal: false,
    status: 'coming-soon',
    tags: ['牌桌', '多人'],
  },
]

export function findGame(slug: string) {
  return localGames.find(game => game.slug === slug)
}
