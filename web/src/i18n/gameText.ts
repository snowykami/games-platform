import type { Locale } from './messages'
import type { GameDefinition } from '@/games/registry'
import { messages } from './messages'

export function localizeGame(game: GameDefinition, locale: Locale): GameDefinition {
  const text = messages[locale].games[game.slug as keyof typeof messages.en.games] ?? messages.en.games[game.slug as keyof typeof messages.en.games]
  if (!text) {
    return game
  }
  return {
    ...game,
    description: text.description,
    tags: [...text.tags],
    title: text.title,
  }
}

export function localizeGames(games: GameDefinition[], locale: Locale) {
  return games.map(game => localizeGame(game, locale))
}
