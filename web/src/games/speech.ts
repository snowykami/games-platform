export interface GameSpeechEntry {
  id: string
  playerId: string
  playerName: string
  text: string
  spokenAt: string
}

export function latestSpeechForPlayer(speeches: GameSpeechEntry[] | undefined, playerId: string) {
  return speeches?.filter(speech => speech.playerId === playerId).at(-1)
}
