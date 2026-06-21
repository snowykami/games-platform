import type { GameSpeechEntry } from '@/games/speech'
import { z } from 'zod'
import { fetchWithAuthRedirect } from '@/auth/fetch'

export type SocialGameSlug = 'werewolf' | 'avalon' | 'undercover'
export type SocialPhase = 'lobby' | 'night' | 'day' | 'vote' | 'hunter' | 'team' | 'team_vote' | 'quest' | 'assassination' | 'describe' | 'undercover_vote' | 'finished'
export type SocialAlignment = 'good' | 'evil' | 'neutral'
export type SocialRole = 'villager' | 'werewolf' | 'seer' | 'guard' | 'witch' | 'hunter' | 'idiot' | 'merlin' | 'assassin' | 'minion' | 'loyal' | 'civilian' | 'undercover' | 'blank'

export interface WerewolfRoleCounts {
  villager: number
  werewolf: number
  seer: number
  guard: number
  witch: number
  hunter: number
  idiot: number
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
  note?: string
  visibleToYou: boolean
}

export interface SocialRoom {
  id: string
  game: SocialGameSlug
  hostPlayerId?: string
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
    votes: Record<string, { targetId: string, confirmed: boolean }>
    lastNight?: string
    witchVictimId?: string
    witchAntidoteUsed?: boolean
    witchPoisonUsed?: boolean
    hunterPendingId?: string
    revealedIdiots?: Record<string, boolean>
  }
  avalon: {
    round: number
    leaderId?: string
    team: string[]
    teamVotes: Record<string, boolean>
    teamVoteCount: number
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
    votes: Record<string, { targetId: string, confirmed: boolean }>
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

const roleSchema = z.enum(['villager', 'werewolf', 'seer', 'guard', 'witch', 'hunter', 'idiot', 'merlin', 'assassin', 'minion', 'loyal', 'civilian', 'undercover', 'blank'])
const alignmentSchema = z.enum(['good', 'evil', 'neutral'])
const undercoverWordPairSchema = z.object({
  id: z.string(),
  civilianWord: z.string().optional(),
  undercoverWord: z.string().optional(),
  category: z.string().optional(),
})
const werewolfRoleCountsSchema = z.object({
  villager: z.number().default(0),
  werewolf: z.number().default(0),
  seer: z.number().default(0),
  guard: z.number().default(0),
  witch: z.number().default(0),
  hunter: z.number().default(0),
  idiot: z.number().default(0),
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

const defaultWerewolfRoleCounts: WerewolfRoleCounts = { guard: 0, hunter: 0, idiot: 0, seer: 0, villager: 0, werewolf: 0, witch: 0 }
const defaultWerewolfRoleConfig: WerewolfRoleConfig = {
  counts: defaultWerewolfRoleCounts,
  description: '',
  mode: 'custom',
  name: '',
}

const playerSchema = z.object({
  id: z.string(),
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
  note: z.string().optional(),
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
  hostPlayerId: z.string().optional(),
  phase: z.enum(['lobby', 'night', 'day', 'vote', 'hunter', 'team', 'team_vote', 'quest', 'assassination', 'describe', 'undercover_vote', 'finished']),
  players: z.array(playerSchema),
  youPlayerId: z.string().optional(),
  minPlayers: z.number(),
  maxPlayers: z.number(),
  werewolf: z.object({
    day: z.number().default(0),
    roleConfig: werewolfRoleConfigSchema.catch(defaultWerewolfRoleConfig),
    rolePresets: z.array(werewolfRolePresetSchema).default([]),
    seerChecks: z.record(z.string(), alignmentSchema).optional(),
    votes: z.record(z.string(), z.object({
      targetId: z.string(),
      confirmed: z.boolean().default(false),
    })).default({}),
    lastNight: z.string().optional(),
    witchVictimId: z.string().optional(),
    witchAntidoteUsed: z.boolean().optional(),
    witchPoisonUsed: z.boolean().optional(),
    hunterPendingId: z.string().optional(),
    revealedIdiots: z.record(z.string(), z.boolean()).optional(),
  }).default({ day: 0, roleConfig: defaultWerewolfRoleConfig, rolePresets: [], votes: {} }),
  avalon: z.object({
    round: z.number().default(0),
    leaderId: z.string().optional(),
    team: z.array(z.string()).default([]),
    teamVotes: z.record(z.string(), z.boolean()).default({}),
    teamVoteCount: z.number().default(0),
    questResults: z.array(z.object({ round: z.number(), teamSize: z.number(), failCards: z.number() })).default([]),
    rejectedTeams: z.number().default(0),
    requiredTeam: z.number().default(0),
    requiredFails: z.number().default(0),
    successes: z.number().default(0),
    fails: z.number().default(0),
  }).default({ fails: 0, questResults: [], rejectedTeams: 0, requiredFails: 0, requiredTeam: 0, round: 0, successes: 0, team: [], teamVoteCount: 0, teamVotes: {} }),
  undercover: z.object({
    round: z.number().default(0),
    presetId: z.string().default(''),
    presets: z.array(z.object({
      id: z.string(),
      name: z.string(),
      description: z.string(),
      pairs: z.array(undercoverWordPairSchema).optional(),
    })).optional(),
    wordPair: undercoverWordPairSchema.optional(),
    includeBlank: z.boolean().default(false),
    currentSpeakerId: z.string().optional(),
    described: z.record(z.string(), z.boolean()).default({}),
    votes: z.record(z.string(), z.object({
      targetId: z.string(),
      confirmed: z.boolean(),
    })).default({}),
    lastEliminatedId: z.string().optional(),
  }).default({ described: {}, includeBlank: false, presetId: '', round: 0, votes: {} }),
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

export function parseSocialRoom(value: unknown): SocialRoom {
  return roomSchema.parse(value)
}

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

export async function renameSocialPlayer(game: SocialGameSlug, roomId: string, name: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/name`, jsonPatch({ name }))
}

export async function updateSocialPlayerNote(game: SocialGameSlug, roomId: string, playerId: string, note: string) {
  return requestRoom(game, `/rooms/${encodeURIComponent(roomId)}/notes/${encodeURIComponent(playerId)}`, jsonPatch({ note }))
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

export async function voteUndercover(roomId: string, targetId: string, confirmed: boolean) {
  return requestRoom('undercover', `/rooms/${encodeURIComponent(roomId)}/undercover-vote`, jsonPost({ confirmed, targetId }))
}

export async function updateWerewolfRoles(roomId: string, config: WerewolfRoleConfig) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/werewolf-roles`, jsonPost({ config }))
}

export async function werewolfNightAction(roomId: string, actionId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/night-action`, jsonPost({ actionId }))
}

export async function werewolfHunterShot(roomId: string, targetId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/hunter-shot`, jsonPost({ targetId }))
}

export async function advanceWerewolfDay(roomId: string) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/advance-day`, { method: 'POST' })
}

export async function werewolfVote(roomId: string, targetId: string, confirmed: boolean) {
  return requestRoom('werewolf', `/rooms/${encodeURIComponent(roomId)}/werewolf-vote`, jsonPost({ confirmed, targetId }))
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
  const response = await fetchWithAuthRedirect(`/api/${game}${path}`, init)
  if (!response.ok) {
    const error = await response.json().catch(() => undefined)
    throw new Error(error?.error ?? `Request failed: ${response.status}`)
  }
  return roomResponseSchema.parse(await response.json()).room
}
