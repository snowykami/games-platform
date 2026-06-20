import type { GameSpeechEntry } from '@/games/speech'
import { z } from 'zod'

export type GomokuPhase = 'lobby' | 'playing' | 'finished'
export type GomokuStone = 'black' | 'white'

export interface GomokuPlayer {
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
    level: string
  }
  stone?: GomokuStone
}

export interface GomokuMove {
  x: number
  y: number
  stone: GomokuStone
  playerId: string
  playerName: string
  placedAt: string
}

export interface GomokuPoint {
  x: number
  y: number
}

export interface GomokuPublicAction {
  seq: number
  type: 'place' | 'win' | 'draw'
  actorId: string
  actorName: string
  x?: number
  y?: number
  stone?: GomokuStone
  message: string
}

export interface GomokuOnlineRoom {
  id: string
  hostUserId: string
  phase: GomokuPhase
  players: GomokuPlayer[]
  boardSize: number
  moves: GomokuMove[]
  currentPlayerId?: string
  winnerId?: string
  winningLine: GomokuPoint[]
  isDraw: boolean
  log: Array<{ id: string, text: string }>
  speeches: GameSpeechEntry[]
  actionSeq: number
  recentActions: GomokuPublicAction[]
}

const stoneSchema = z.enum(['black', 'white'])

const playerSchema = z.object({
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
    level: z.string(),
  }).optional(),
  stone: stoneSchema.optional(),
})

const pointSchema = z.object({
  x: z.number(),
  y: z.number(),
})

const speechSchema = z.object({
  id: z.string(),
  playerId: z.string(),
  playerName: z.string(),
  text: z.string(),
  spokenAt: z.string(),
})

const moveSchema = pointSchema.extend({
  stone: stoneSchema,
  playerId: z.string(),
  playerName: z.string(),
  placedAt: z.string(),
})

const roomSchema: z.ZodType<GomokuOnlineRoom> = z.object({
  id: z.string(),
  hostUserId: z.string(),
  phase: z.enum(['lobby', 'playing', 'finished']),
  players: z.array(playerSchema),
  boardSize: z.number(),
  moves: z.array(moveSchema),
  currentPlayerId: z.string().optional(),
  winnerId: z.string().optional(),
  winningLine: z.array(pointSchema),
  isDraw: z.boolean(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
  speeches: z.array(speechSchema),
  actionSeq: z.number(),
  recentActions: z.array(z.object({
    seq: z.number(),
    type: z.enum(['place', 'win', 'draw']),
    actorId: z.string(),
    actorName: z.string(),
    x: z.number().optional(),
    y: z.number().optional(),
    stone: stoneSchema.optional(),
    message: z.string(),
  })),
})

const roomResponseSchema = z.object({ room: roomSchema })

export async function createGomokuRoom() {
  return requestRoom('/api/gomoku/rooms', { method: 'POST' })
}

export async function joinGomokuRoom(roomId: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function getGomokuRoom(roomId: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}`)
}

export async function addGomokuAI(roomId: string, level: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/ai`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function updateGomokuAI(roomId: string, playerId: string, level: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/ai/${encodeURIComponent(playerId)}`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function sayGomoku(roomId: string, text: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/speech`, {
    body: JSON.stringify({ text }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function startGomokuRoom(roomId: string) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/start`, { method: 'POST' })
}

export async function placeGomokuStone(roomId: string, x: number, y: number) {
  return requestRoom(`/api/gomoku/rooms/${encodeURIComponent(roomId)}/place`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ x, y }),
  })
}

async function requestRoom(input: RequestInfo | URL, init?: RequestInit) {
  const response = await fetch(input, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `Request failed: ${response.status}`)
  }

  return roomResponseSchema.parse(await response.json()).room
}
