export type UserRole = 'admin' | 'player'
export type IdentityKind = 'guest' | 'oidc'

export interface AuthUser {
  id: string
  kind: IdentityKind
  role: UserRole
  displayName: string
  banned: boolean
  createdAt: string
}

export interface OIDCProvider {
  key: string
  displayName: string
  enabled: boolean
}
