import type { UnoCard, UnoColor } from './types'
import type { GameSpeechEntry } from '@/games/speech'
import { z } from 'zod'
import { fetchWithAuthRedirect } from '@/auth/fetch'

export type UnoPhase = 'lobby' | 'playing' | 'finished'

export interface UnoOnlinePlayer {
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
  hand?: UnoCard[]
  handCount: number
  needsUno: boolean
}

export interface UnoPublicAction {
  seq: number
  type: 'play' | 'draw' | 'effect' | 'win'
  actorId: string
  actorName: string
  targetId?: string
  card?: UnoCard
  count?: number
  message: string
}

export interface UnoOnlineRoom {
  id: string
  hostPlayerId?: string
  youPlayerId?: string
  variantKey: string
  themeKey: string
  phase: UnoPhase
  players: UnoOnlinePlayer[]
  topCard?: UnoCard
  drawPileCount: number
  currentPlayerId?: string
  direction: 1 | -1
  activeColor?: UnoColor
  pendingDrawCount: number
  flipSide: boolean
  rules: {
    stacking: boolean
    sevenZero: boolean
    jumpIn: boolean
    allWild: boolean
    flip: boolean
    noMercy: boolean
  }
  playableCardIds: string[]
  winnerId?: string
  log: Array<{ id: string, text: string }>
  speeches: GameSpeechEntry[]
  actionSeq: number
  recentActions: UnoPublicAction[]
  turnDeadline?: string
  turnRemainingSeconds: number
}

const cardSchema = z.object({
  id: z.string(),
  color: z.enum(['red', 'yellow', 'green', 'blue', 'wild']),
  kind: z.enum(['number', 'skip', 'reverse', 'draw-two', 'wild', 'wild-draw-four', 'wild-draw-six', 'wild-draw-ten', 'flip']),
  value: z.number().optional(),
})

const rulesSchema = z.object({
  stacking: z.boolean(),
  sevenZero: z.boolean(),
  jumpIn: z.boolean(),
  allWild: z.boolean(),
  flip: z.boolean(),
  noMercy: z.boolean(),
})

const speechSchema = z.object({
  id: z.string(),
  playerId: z.string(),
  playerName: z.string(),
  text: z.string(),
  spokenAt: z.string(),
})

const roomSchema: z.ZodType<UnoOnlineRoom> = z.object({
  id: z.string(),
  hostPlayerId: z.string().optional(),
  youPlayerId: z.string().optional(),
  variantKey: z.string(),
  themeKey: z.string(),
  phase: z.enum(['lobby', 'playing', 'finished']),
  players: z.array(z.object({
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
    hand: z.array(cardSchema).optional(),
    handCount: z.number(),
    needsUno: z.boolean(),
  })),
  topCard: cardSchema.optional(),
  drawPileCount: z.number(),
  currentPlayerId: z.string().optional(),
  direction: z.union([z.literal(1), z.literal(-1)]),
  activeColor: z.enum(['red', 'yellow', 'green', 'blue']).optional(),
  pendingDrawCount: z.number(),
  flipSide: z.boolean(),
  rules: rulesSchema,
  playableCardIds: z.array(z.string()),
  winnerId: z.string().optional(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
  speeches: z.array(speechSchema),
  actionSeq: z.number(),
  recentActions: z.array(z.object({
    seq: z.number(),
    type: z.enum(['play', 'draw', 'effect', 'win']),
    actorId: z.string(),
    actorName: z.string(),
    targetId: z.string().optional(),
    card: cardSchema.optional(),
    count: z.number().optional(),
    message: z.string(),
  })),
  turnDeadline: z.string().optional(),
  turnRemainingSeconds: z.number(),
})

const roomResponseSchema = z.object({ room: roomSchema })

export async function createUnoRoom(options: { themeKey: string, variantKey: string }) {
  return requestRoom('/api/uno/rooms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options),
  })
}

export async function joinUnoRoom(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function getUnoRoom(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}`)
}

export async function addUnoAI(roomId: string, level: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/ai`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function updateUnoAI(roomId: string, playerId: string, level: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/ai/${encodeURIComponent(playerId)}`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function removeUnoPlayer(roomId: string, playerId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/players/${encodeURIComponent(playerId)}`, { method: 'DELETE' })
}

export async function sayUno(roomId: string, text: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/speech`, {
    body: JSON.stringify({ text }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function renameUnoPlayer(roomId: string, name: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/name`, {
    body: JSON.stringify({ name }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function startUnoRoom(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/start`, { method: 'POST' })
}

export async function drawUnoCard(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/draw`, { method: 'POST' })
}

export async function playUnoCard(roomId: string, cardId: string, color: UnoColor) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/play`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cardId, color }),
  })
}

export async function callUno(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/uno`, { method: 'POST' })
}

export async function catchUno(roomId: string, targetId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/catch-uno`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetId }),
  })
}

async function requestRoom(input: RequestInfo | URL, init?: RequestInit) {
  const response = await fetchWithAuthRedirect(input, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `Request failed: ${response.status}`)
  }

  return roomResponseSchema.parse(await response.json()).room
}
