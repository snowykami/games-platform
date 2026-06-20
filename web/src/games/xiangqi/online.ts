import type { GameSpeechEntry } from '@/games/speech'
import { z } from 'zod'

export type XiangqiPhase = 'lobby' | 'playing' | 'finished'
export type XiangqiSide = 'red' | 'black'
export type XiangqiPieceType = 'general' | 'advisor' | 'elephant' | 'horse' | 'rook' | 'cannon' | 'soldier'

export interface XiangqiOnlinePiece {
  id: string
  side: XiangqiSide
  type: XiangqiPieceType
  x: number
  y: number
}

export interface XiangqiOnlinePlayer {
  id: string
  userId: string
  name: string
  role: 'host' | 'player'
  kind: 'guest' | 'oidc' | 'ai'
  isAI: boolean
  connected: boolean
  side?: XiangqiSide
  ai?: {
    name: string
    personality: string
    level: string
  }
}

export interface XiangqiOnlineMove {
  id: string
  pieceId: string
  pieceType: XiangqiPieceType
  side: XiangqiSide
  from: { x: number, y: number }
  to: { x: number, y: number }
  captured?: XiangqiOnlinePiece
  check: boolean
  checkmate: boolean
  playerId: string
  playerName: string
  playedAt: string
}

export interface XiangqiOnlineRoom {
  id: string
  hostUserId: string
  phase: XiangqiPhase
  players: XiangqiOnlinePlayer[]
  pieces: XiangqiOnlinePiece[]
  moves: XiangqiOnlineMove[]
  currentPlayerId?: string
  winnerId?: string
  checkSide?: XiangqiSide
  log: Array<{ id: string, text: string }>
  speeches: GameSpeechEntry[]
  actionSeq: number
  recentActions: Array<{
    seq: number
    type: 'move' | 'capture' | 'check' | 'checkmate'
    actorId: string
    actorName: string
    move?: XiangqiOnlineMove
    message: string
  }>
}

const sideSchema = z.enum(['red', 'black'])
const pieceTypeSchema = z.enum(['general', 'advisor', 'elephant', 'horse', 'rook', 'cannon', 'soldier'])
const positionSchema = z.object({ x: z.number(), y: z.number() })
const pieceSchema = z.object({
  id: z.string(),
  side: sideSchema,
  type: pieceTypeSchema,
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
const moveSchema: z.ZodType<XiangqiOnlineMove> = z.object({
  id: z.string(),
  pieceId: z.string(),
  pieceType: pieceTypeSchema,
  side: sideSchema,
  from: positionSchema,
  to: positionSchema,
  captured: pieceSchema.optional(),
  check: z.boolean(),
  checkmate: z.boolean(),
  playerId: z.string(),
  playerName: z.string(),
  playedAt: z.string(),
})
const playerSchema = z.object({
  id: z.string(),
  userId: z.string(),
  name: z.string(),
  role: z.enum(['host', 'player']),
  kind: z.enum(['guest', 'oidc', 'ai']),
  isAI: z.boolean(),
  connected: z.boolean(),
  side: sideSchema.optional(),
  ai: z.object({
    name: z.string(),
    personality: z.string(),
    level: z.string(),
  }).optional(),
})
const roomSchema: z.ZodType<XiangqiOnlineRoom> = z.object({
  id: z.string(),
  hostUserId: z.string(),
  phase: z.enum(['lobby', 'playing', 'finished']),
  players: z.array(playerSchema),
  pieces: z.array(pieceSchema),
  moves: z.array(moveSchema),
  currentPlayerId: z.string().optional(),
  winnerId: z.string().optional(),
  checkSide: sideSchema.optional(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
  speeches: z.array(speechSchema),
  actionSeq: z.number(),
  recentActions: z.array(z.object({
    seq: z.number(),
    type: z.enum(['move', 'capture', 'check', 'checkmate']),
    actorId: z.string(),
    actorName: z.string(),
    move: moveSchema.optional(),
    message: z.string(),
  })),
})
const roomResponseSchema = z.object({ room: roomSchema })

export async function createXiangqiRoom() {
  return requestRoom('/api/xiangqi/rooms', { method: 'POST' })
}

export async function joinXiangqiRoom(roomId: string) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function addXiangqiAI(roomId: string, level: string) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/ai`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level }),
  })
}

export async function updateXiangqiAI(roomId: string, playerId: string, level: string) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/ai/${encodeURIComponent(playerId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level }),
  })
}

export async function sayXiangqi(roomId: string, text: string) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/speech`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  })
}

export async function startXiangqiRoom(roomId: string) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/start`, { method: 'POST' })
}

export async function moveXiangqiPiece(roomId: string, pieceId: string, x: number, y: number) {
  return requestRoom(`/api/xiangqi/rooms/${encodeURIComponent(roomId)}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pieceId, x, y }),
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
