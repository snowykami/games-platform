import { useQuery } from '@tanstack/react-query'
import { Route, Routes } from 'react-router'
import { useAuth } from '@/auth/AuthContext'
import { RequireAuth } from '@/auth/RequireAuth'
import { getGames } from '@/games/api'
import { localGames } from '@/games/registry'
import { GameCatalogPage } from '@/pages/GameCatalogPage'
import { GamePage } from '@/pages/GamePage'
import { LoginPage } from '@/pages/LoginPage'

export function AppRouter() {
  const { user } = useAuth()
  const gamesQuery = useQuery({
    queryKey: ['games', user?.id ?? 'anonymous'],
    queryFn: getGames,
    initialData: localGames,
  })

  return (
    <Routes>
      <Route
        element={<GameCatalogPage games={gamesQuery.data} isLoading={gamesQuery.isFetching} />}
        path="/"
      />
      <Route element={<LoginPage />} path="/login" />
      <Route
        element={(
          <RequireAuth>
            <GamePage />
          </RequireAuth>
        )}
        path="/games/:slug"
      />
    </Routes>
  )
}
