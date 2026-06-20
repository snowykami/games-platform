import type { GameSpeechEntry } from '@/games/speech'
import { z } from 'zod'

export type SocialGameSlug = 'werewolf' | 'avalon' | 'undercover'
export type SocialPhase = 'lobby' | 'night' | 'day' | 'vote' | 'team' | 'team_vote' | 'quest' | 'assassination' | 'describe' | 'undercover_vote' | 'finished'
export type SocialAlignment = 'good' | 'evil' | 'neutral'
export type SocialRole = 'villager' | 'werewolf' | 'seer' | 'guard' | 'merlin' | 'assassin' | 'minion' | 'loyal' | 'civilian' | 'undercover' | 'blank'

export interface WerewolfRoleCounts {
  villager: number
  werewolf: number
  seer: number
  guard: number
}

export interface WerewolfRoleConfig {
  mode: 'preset' | 'custom'
  presetId?: string
  name: string
  description?: string
  counts: WerewolfRoleCounts
}

export interface WerewolfRolePreset {
  id: string
  name: string
  description: string
  players: number
  counts: WerewolfRoleCounts
}

export interface SocialPlayer {
  id: string
  userId: string
  name: string
  seat: number
  roomRole: 'host' | 'player'
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
  alive: boolean
  role?: SocialRole
  alignment?: SocialAlignment
  visibleToYou: boolean
}

export interface SocialRoom {
  id: string
  game: SocialGameSlug
  hostUserId: string
  phase: SocialPhase
  players: SocialPlayer[]
  youPlayerId?: string
  minPlayers: number
  maxPlayers: number
  werewolf: {
    day: number
    roleConfig: WerewolfRoleConfig
    rolePresets: WerewolfRolePreset[]
    seerChecks?: Record<string, SocialAlignment>
    votes: Record<string, string>
    lastNight?: string
  }
  avalon: {
    round: number
    leaderId?: string
    team: string[]
    teamVotes: Record<string, boolean>
    questResults: Array<{ round: number, teamSize: number, failCards: number }>
    rejectedTeams: number
    requiredTeam: number
    requiredFails: number
    successes: number
    fails: number
  }
  undercover: {
    round: number
    presetId: string
    presets?: Array<{
      id: string
      name: string
      description: string
      pairs?: Array<{
        id: string
        civilianWord?: string
        undercoverWord?: string
        category?: string
      }>
    }>
    wordPair?: {
      id: string
      civilianWord?: string
      undercoverWord?: string
      category?: string
    }
    includeBlank: boolean
    currentSpeakerId?: string
    described: Record<string, boolean>
    votes: Record<string, string>
    lastEliminatedId?: string
  }
  winner?: SocialAlignment
  winnerMessage?: string
  log: Array<{ id: string, text: string }>
  speeches: GameSpeechEntry[]
  actionSeq: number
  recentActions: Array<{
    seq: number
    type: string
    actorId?: string
    actorName?: string
    targetId?: string
    message: string
  }>
}

const roleSchema = z.enum(['villager', 'werewolf', 'seer', 'guard', 'merlin', 'assassin', 'minion', 'loyal', 'civilian', 'undercover', 'blank'])
const alignmentSchema = z.enum(['good', 'evil', 'neutral'])
const undercoverWordPairSchema = z.object({
  id: z.string(),
  civilianWord: z.string().optional(),
  undercoverWord: z.string().optional(),
  category: z.string().optional(),
})
const werewolfRoleCountsSchema = z.object({
  villager: z.number(),
  werewolf: z.number(),
  seer: z.number(),
  guard: z.number(),
})
const werewolfRoleConfigSchema: z.ZodType<WerewolfRoleConfig> = z.object({
  mode: z.enum(['preset', 'custom']),
  presetId: z.string().optional(),
  name: z.string(),
  description: z.string().optional(),
  counts: werewolfRoleCountsSchema,
})
const werewolfRolePresetSchema: z.ZodType<WerewolfRolePreset> = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  players: z.number(),
  counts: werewolfRoleCountsSchema,
})

const playerSchema = z.object({
  id: z.string(),
  userId: z.string(),
  name: z.string(),
  seat: z.number(),
  roomRole: z.enum(['host', 'player']),
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
  alive: z.boolean(),
  role: roleSchema.optional(),
  alignment: alignmentSchema.optional(),
  visibleToYou: z.boolean(),
})

const speechSchema = z.object({
  id: z.string(),
  playerId: z.string(),
  playerName: z.string(),
  text: z.string(),
  spokenAt: z.string(),
})

const roomSchema: z.ZodType<SocialRoom> = z.object({
  id: z.string(),
  game: z.enum(['werewolf', 'avalon', 'undercover']),
  hostUserId: z.string(),
  phase: z.enum(['lobby', 'night', 'day', 'vote', 'team', 'team_vote', 'quest', 'assassination', 'describe', 'undercover_vote', 'finished']),
  players: z.array(playerSchema),
  youPlayerId: z.string().optional(),
  minPlayers: z.number(),
  maxPlayers: z.number(),
  werewolf: z.object({
    day: z.number(),
    roleConfig: werewolfRoleConfigSchema,
    rolePresets: z.array(werewolfRolePresetSchema).default([]),
    seerChecks: z.record(z.string(), alignmentSchema).optional(),
    votes: z.record(z.string(), z.string()),
    lastNight: z.string().optional(),
  }),
  avalon: z.object({
    round: z.number(),
    leaderId: z.string().optional(),
    team: z.array(z.string()),
    teamVotes: z.record(z.string(), z.boolean()),
    questResults: z.array(z.object({ round: z.number(), teamSize: z.number(), failCards: z.number() })),
    rejectedTeams: z.number(),
    requiredTeam: z.number(),
    requiredFails: z.number(),
    successes: z.number(),
    fails: z.number(),
  }),
  undercover: z.object({
    round: z.number(),
    presetId: z.string(),
    presets: z.array(z.object({
      id: z.string(),
      name: z.string(),
      description: z.string(),
      pairs: z.array(undercoverWordPairSchema).optional(),
    })).optional(),
    wordPair: undercoverWordPairSchema.optional(),
    includeBlank: z.boolean(),
    currentSpeakerId: z.string().optional(),
    described: z.record(z.string(), z.boolean()),
    votes: z.record(z.string(), z.string()),
    lastEliminatedId: z.string().optional(),
  }),
  winner: alignmentSchema.optional(),
  winnerMessage: z.string().optional(),
  log: z.array(z.object({ id: z.string(), text: z.string() })),
  speeches: z.array(speechSchema),
  actionSeq: z.number(),
  recentActions: z.array(z.object({
    seq: z.number(),
    type: z.string(),
    actorId: z.string().optional(),
    actorName: z.string().optional(),
    targetId: z.string().optional(),
    message: z.string(),
  })),
})

const roomResponseSchema = z.object({ room: roomSchema })

export async function createSocialRoom(game: SocialGameSlug) {
  return requestRoom(game, '/rooms', { method: 'POST' })
}

export async function joinSocialRoom(game: SocialGameSlug, roomId: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/join`, { method: 'POST' })
}

export async function addSocialAI(game: SocialGameSlug, roomId: string, level: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/ai`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function updateSocialAI(game: SocialGameSlug, roomId: string, playerId: string, level: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/ai/${encodeURIComponent(playerId)}`, {
    body: JSON.stringify({ level }),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  })
}

export async function removeSocialPlayer(game: SocialGameSlug, roomId: string, playerId: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/players/${encodeURIComponent(playerId)}`, { method: 'DELETE' })
}

export async function saySocial(game: SocialGameSlug, roomId: string, text: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/speech`, {
    body: JSON.stringify({ text }),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
}

export async function startSocialRoom(game: SocialGameSlug, roomId: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/start`, { method: 'POST' })
}

export async function updateUndercoverConfig(roomId: string, presetId: string, includeBlank: boolean) {
  return requestRoom('undercover', `/rooms/${encodeURIComponent(roomId)}/undercover-config`, jsonPatch({ includeBlank, presetId }))
}

export async function describeUndercover(roomId: string, text: string) {
  return requestRoom('undercover', `/rooms/${encodeURIComponent(roomId)}/describe`, jsonPost({ text }))
}

export async function voteUndercover(roomId: string, targetId: string) {
  return requestRoom('undercover', `/rooms/${encodeURIComponent(roomId)}/undercover-vote`, jsonPost({ targetId }))
}

export async function updateWerewolfRoles(roomId: string, config: WerewolfRoleConfig) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/werewolf-roles`, jsonPost({ config }))
}

export async function werewolfNightAction(roomId: string, targetId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/night-action`, jsonPost({ targetId }))
}

export async function advanceWerewolfDay(roomId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/advance-day`, { method: 'POST' })
}

export async function werewolfVote(roomId: string, targetId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/werewolf-vote`, jsonPost({ targetId }))
}

export async function proposeAvalonTeam(roomId: string, team: string[]) {
  return requestRoom('avalon', `/rooms/${encodeURIComponent(roomId)}/team`, jsonPost({ team }))
}

export async function voteAvalonTeam(roomId: string, approve: boolean) {
  return requestRoom('avalon', `/rooms/${encodeURIComponent(roomId)}/team-vote`, jsonPost({ approve }))
}

export async function playAvalonQuest(roomId: string, card: 'success' | 'fail') {
  return requestRoom('avalon', `/rooms/${encodeURIComponent(roomId)}/quest`, jsonPost({ card }))
}

export async function assassinateAvalon(roomId: string, targetId: string) {
  return requestRoom('avalon', `/rooms/${encodeURIComponent(roomId)}/assassinate`, jsonPost({ targetId }))
}

function jsonPost(body: unknown): RequestInit {
  return {
    body: JSON.stringify(body),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  }
}

function jsonPatch(body: unknown): RequestInit {
  return {
    body: JSON.stringify(body),
    headers: { 'Content-Type': 'application/json' },
    method: 'PATCH',
  }
}

async function requestRoom(game: SocialGameSlug, path: string, init?: RequestInit) {
  const response = await fetch(`/api/${game}${path}`, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `Request failed: ${response.status}`)
  }
  return roomResponseSchema.parse(await response.json()).room
}
