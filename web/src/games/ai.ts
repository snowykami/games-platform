import { z } from 'zod'
import { fetchWithAuthRedirect } from '@/auth/fetch'
import { messages } from '@/i18n/messages'

export type AILevel = 'beginner' | 'normal' | 'master' | 'ai'

export const AI_LEVELS: AILevel[] = ['beginner', 'normal', 'master', 'ai']

export function normalizeAILevel(level: string | undefined): AILevel {
  return AI_LEVELS.includes(level as AILevel) ? level as AILevel : 'normal'
}

export function getAILevelLabel(level: AILevel, locale: keyof typeof messages) {
  return messages[locale].ai[level].name
}

export function getAILevelDescription(level: AILevel, locale: keyof typeof messages) {
  return messages[locale].ai[level].description
}

export interface AICapabilities {
  levels: AILevel[]
  llmEnabled: boolean
  profiles: Array<{
    name: string
    personality: string
    speechStyle: string
  }>
}

export interface LegalAction {
  id: string
  label: string
  description?: string
}

const capabilitiesSchema = z.object({
  levels: z.array(z.enum(['beginner', 'normal', 'master', 'ai'])),
  llmEnabled: z.boolean(),
  profiles: z.array(z.object({
    name: z.string(),
    personality: z.string(),
    speechStyle: z.string(),
  })).default([]),
})

const decisionSchema = z.object({
  decision: z.object({
    actionId: z.string(),
    reason: z.string().optional(),
    speech: z.string().optional(),
    source: z.string(),
  }),
})

export async function getAICapabilities(): Promise<AICapabilities> {
  const response = await fetch('/api/ai/capabilities')
  if (!response.ok) {
    return { levels: ['beginner', 'normal', 'master'], llmEnabled: false, profiles: [] }
  }
  return capabilitiesSchema.parse(await response.json())
}

export async function decideWithAI(input: {
  actions: LegalAction[]
  game: string
  level: AILevel
  personality?: string
  playerName?: string
  speechStyle?: string
  sessionId: string
  state: unknown
}) {
  const response = await fetchWithAuthRedirect('/api/ai/decide', {
    body: JSON.stringify(input),
    headers: { 'Content-Type': 'application/json' },
    method: 'POST',
  })
  if (!response.ok) {
    throw new Error(`AI decision failed: ${response.status}`)
  }
  return decisionSchema.parse(await response.json()).decision
}
