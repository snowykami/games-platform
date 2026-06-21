import type { AuthUser } from './types'
import { createContext, use } from 'react'

export interface AuthContextValue {
  isLoading: boolean
  loginGuest: () => Promise<AuthUser>
  logout: () => Promise<void>
  refresh: () => Promise<void>
  user?: AuthUser
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function useAuth() {
  const context = use(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used inside AuthProvider')
  }
  return context
}
