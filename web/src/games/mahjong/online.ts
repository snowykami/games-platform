import { z } from 'zod'
import { fetchWithAuthRedirect } from '@/auth/fetch'

export interface MahjongSpeechEntry {
  id: string
  playerId: string
  playerName: string
  text: string
  spokenAt: string
}

export type MahjongPhase = 'lobby' | 'playing' | 'claiming' | 'finished'
export type MahjongWind = 'east' | 'south' | 'west' | 'north'
export type MahjongTileKind = 'characters' | 'dots' | 'bamboo' | 'wind' | 'dragon'
export type MahjongMeldKind = 'chow' | 'pung' | 'kong'
export type MahjongClaimKind = 'hu' | 'peng' | 'chi'

export interface MahjongOnlineTile {
  id: string
  code: string
  kind: MahjongTileKind
  rank?: number
  wind?: MahjongWind
  dragon?: 'red' | 'green' | 'white'
}

export interface MahjongOnlineMeld {
  id: string
  kind: MahjongMeldKind
  tiles: MahjongOnlineTile[]
  fromPlayerId?: string
  exposed: boolean
}

export interface MahjongOnlineWinResult {
  canWin: boolean
  fan: number
  patterns: Array<{ name: string, fan: number }>
  reason?: string
}

export interface MahjongOnlinePlayer {
  id: string
  name: string
  role: 'host' | 'player'
  kind: 'guest' | 'oidc' | 'ai'
  isAI: boolean
  connected: boolean
  disconnectedAt?: string
  ai?: {
    name: string
    personality: string
    speechStyle?: string
    level: string
  }
  wind: MahjongWind
  hand: MahjongOnlineTile[]
  handCount: number
  melds: MahjongOnlineMeld[]
  discards: MahjongOnlineTile[]
}

export interface MahjongClaimOption {
  id: string
  playerId: string
  kind: MahjongClaimKind
  tile: MahjongOnlineTile
  tilesFromHand: MahjongOnlineTile[]
  winResult?: MahjongOnlineWinResult
}

export interface MahjongOnlineRoom {
  id: string
  hostPlayerId?: string
  youPlayerId?: string
  phase: MahjongPhase
  players: MahjongOnlinePlayer[]
  wallCount: number
  deadWallCount: number
  currentPlayerId?: string
  dealerId?: string
  roundWind: MahjongWind
  hasDrawn: boolean
  lastDiscard?: {
    tile: MahjongOnlineTile
    playerId: string
  }
  claimOptions: MahjongClaimOption[]
  ruleset: {
    id: string
    name: string
    minFan: number
    description: string
  }
  winnerId?: string
  winResult?: MahjongOnlineWinResult
  log: Array<{ id: string, text: string }>
  speeches: MahjongSpeechEntry[]
  actionSeq: number
  recentActions: Array<{
    seq: number
    type: 'draw' | 'discard' | 'claim' | 'win' | 'start'
    actorId: string
    actorName: string
    targetId?: string
    tile?: MahjongOnlineTile
    message: string
  }>
}

const windSchema = z.enum(['east', 'south', 'west', 'north'])
const tileSchema: z.ZodType<MahjongOnlineTile> = z.object({
  id: z.string(),
  code: z.string(),
  kind: z.enum(['characters', 'dots', 'bamboo', 'wind', 'dragon']),
  rank: z.number().optional(),
  wind: windSchema.optional(),
  dragon: z.enum(['red', 'green', 'white']).optional(),
})
const meldSchema: z.ZodType<MahjongOnlineMeld> = z.object({
  id: z.string(),
  kind: z.enum(['chow', 'pung', 'kong']),
  tiles: z.array(tileSchema),
  fromPlayerId: z.string().optional(),
  exposed: z.boolean(),
})
const winResultSchema: z.ZodType<MahjongOnlineWinResult> = z.object({
  canWin: z.boolean().default(false),
  fan: z.number().default(0),
  patterns: z.array(z.object({ name: z.string(), fan: z.number() })).nullish().transform(patterns => patterns ?? []),
  reason: z.string().optional(),
})
const playerSchema: z.ZodType<MahjongOnlinePlayer> = z.object({
  id: z.string(),
  name: z.string(),
  role: z.enum(['host', 'player']),
  kind: z.enum(['guest', 'oidc', 'ai']),
  isAI: z.boolean(),
  connected: z.boolean(),
  disconnectedAt: z.string().optional(),
  ai: z.object({
    name: z.string(),
    personality: z.string(),
    speechStyle: z.string().optional(),
    level: z.string(),
  }).optional(),
  wind: z.preprocess(value => value === '' ? 'east' : value, windSchema),
  hand: z.array(tileSchema),
  handCount: z.number(),
  melds: z.array(meldSchema),
  discards: z.array(tileSchema),
})
const speechSchema = z.object({
  id: z.string(),
  playerId: z.string(),
  playerName: z.string(),
  text: z.string(),
  spokenAt: z.string(),
})
const claimSchema: z.ZodType<MahjongClaimOption> = z.object({
  id: z.string(),
  playerId: z.string(),
  kind: z.enum(['hu', 'peng', 'chi']),
  tile: tileSchema,
  tilesFromHand: z.array(tileSchema),
  winResult: winResultSchema.optional(),
})
const roomSchema: z.ZodType<MahjongOnlineRoom> = z.object({
  id: z.string(),
  hostPlayerId: z.string().optional(),
  youPlayerId: z.string().optional(),
  phase: z.enum(['lobby', 'playing', 'claiming', 'finished']),
  players: z.array(playerSchema),
  wallCount: z.number(),
  deadWallCount: z.number(),
  currentPlayerId: z.string().optional(),
  dealerId: z.string().optional(),
  roundWind: windSchema,
  hasDrawn: z.boolean(),
  lastDiscard: z.object({ tile: tileSchema, playerId: z.string() }).optional(),
  claimOptions: z.array(claimSchema),
  ruleset: z.object({
    id: z.string(),
    name: z.string(),
    minFan: z.number(),
    description: z.string(),
  }),
  winnerId: z.string().optional(),
  winResult: winResultSchema.optional(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
  speeches: z.array(speechSchema),
  actionSeq: z.number(),
  recentActions: z.array(z.object({
    seq: z.number(),
    type: z.enum(['draw', 'discard', 'claim', 'win', 'start']),
    actorId: z.string(),
    actorName: z.string(),
    targetId: z.string().optional(),
    tile: tileSchema.optional(),
    message: z.string(),
  })),
})

const roomResponseSchema = z.object({ room: roomSchema })

export async function createMahjongRoom() {
  return requestRoom('/api/mahjong/rooms', { method: 'POST' })
}

export async function joinMahjongRoom(roomId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function addMahjongAI(roomId: string, level: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/ai`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function updateMahjongAI(roomId: string, playerId: string, level: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/ai/${encodeURIComponent(playerId)}`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function removeMahjongPlayer(roomId: string, playerId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/players/${encodeURIComponent(playerId)}`, { method: 'DELETE' })
}

export async function sayMahjong(roomId: string, text: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/speech`, {
    body: JSON.stringify({ text }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function renameMahjongPlayer(roomId: string, name: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/name`, {
    body: JSON.stringify({ name }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function startMahjongRoom(roomId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/start`, { method: 'POST' })
}

export async function drawMahjongTile(roomId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/draw`, { method: 'POST' })
}

export async function discardMahjongTile(roomId: string, tileId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/discard`, {
    body: JSON.stringify({ tileId }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function selfDrawMahjong(roomId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/self-draw`, { method: 'POST' })
}

export async function claimMahjong(roomId: string, claimId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/claim`, {
    body: JSON.stringify({ claimId }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function skipMahjongClaims(roomId: string) {
  return requestRoom(`/api/mahjong/rooms/${encodeURIComponent(roomId)}/skip-claims`, { method: 'POST' })
}

async function requestRoom(input: RequestInfo | URL, init?: RequestInit) {
  const response = await fetchWithAuthRedirect(input, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `Request failed: ${response.status}`)
  }

  return roomResponseSchema.parse(await response.json()).room
}
