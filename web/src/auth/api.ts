import type { AuthUser, OIDCProvider } from './types'
import { z } from 'zod'

const userSchema = z.object({
  id: z.string(),
  kind: z.enum(['guest', 'oidc']),
  role: z.enum(['admin', 'player']),
  displayName: z.string(),
  banned: z.boolean(),
  createdAt: z.string(),
})

const meSchema = z.object({
  user: userSchema.nullish(),
})

const providersSchema = z.object({
  providers: z.array(z.object({
    key: z.string(),
    displayName: z.string(),
    enabled: z.boolean(),
  })),
})

export async function getMe(): Promise<AuthUser | undefined> {
  const response = await fetch('/api/auth/me')
  if (!response.ok) {
    return undefined
  }

  return meSchema.parse(await response.json()).user ?? undefined
}

export async function loginAsGuest(guestUuid: string): Promise<AuthUser> {
  const response = await fetch('/api/auth/guest', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ guestUuid }),
  })
  if (!response.ok) {
    throw new Error('游客登录失败')
  }

  const data = meSchema.parse(await response.json())
  if (!data.user) {
    throw new Error('游客登录没有返回用户')
  }

  return data.user
}

export async function getOIDCProviders(): Promise<OIDCProvider[]> {
  const response = await fetch('/api/auth/oidc/providers')
  if (!response.ok) {
    return []
  }

  return providersSchema.parse(await response.json()).providers
}
