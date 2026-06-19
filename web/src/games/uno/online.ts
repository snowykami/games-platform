import type { UnoCard, UnoColor } from './types'
import { z } from 'zod'

export type UnoPhase = 'lobby' | 'playing' | 'finished'

export interface UnoOnlinePlayer {
  id: string
  userId: string
  name: string
  role: 'host' | 'player'
  kind: 'guest' | 'oidc' | 'ai'
  isAI: boolean
  connected: boolean
  ai?: {
    name: string
    personality: string
  }
  hand?: UnoCard[]
  handCount: number
}

export interface UnoOnlineRoom {
  id: string
  hostUserId: string
  phase: UnoPhase
  players: UnoOnlinePlayer[]
  topCard?: UnoCard
  drawPileCount: number
  currentPlayerId?: string
  direction: 1 | -1
  activeColor?: UnoColor
  winnerId?: string
  log: Array<{ id: string, text: string }>
}

const cardSchema = z.object({
  id: z.string(),
  color: z.enum(['red', 'yellow', 'green', 'blue', 'wild']),
  kind: z.enum(['number', 'skip', 'reverse', 'draw-two', 'wild', 'wild-draw-four']),
  value: z.number().optional(),
})

const roomSchema: z.ZodType<UnoOnlineRoom> = z.object({
  id: z.string(),
  hostUserId: z.string(),
  phase: z.enum(['lobby', 'playing', 'finished']),
  players: z.array(z.object({
    id: z.string(),
    userId: z.string(),
    name: z.string(),
    role: z.enum(['host', 'player']),
    kind: z.enum(['guest', 'oidc', 'ai']),
    isAI: z.boolean(),
    connected: z.boolean(),
    ai: z.object({
      name: z.string(),
      personality: z.string(),
    }).optional(),
    hand: z.array(cardSchema).optional(),
    handCount: z.number(),
  })),
  topCard: cardSchema.optional(),
  drawPileCount: z.number(),
  currentPlayerId: z.string().optional(),
  direction: z.union([z.literal(1), z.literal(-1)]),
  activeColor: z.enum(['red', 'yellow', 'green', 'blue']).optional(),
  winnerId: z.string().optional(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
})

const roomResponseSchema = z.object({ room: roomSchema })

export async function createUnoRoom() {
  return requestRoom('/api/uno/rooms', { method: 'POST' })
}

export async function joinUnoRoom(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function getUnoRoom(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}`)
}

export async function addUnoAI(roomId: string) {
  return requestRoom(`/api/uno/rooms/${encodeURIComponent(roomId)}/ai`, { method: 'POST' })
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

async function requestRoom(input: RequestInfo | URL, init?: RequestInit) {
  const response = await fetch(input, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `请求失败：${response.status}`)
  }

  return roomResponseSchema.parse(await response.json()).room
}
