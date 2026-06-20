export type MahjongSuit = 'characters' | 'dots' | 'bamboo'
export type HonorKind = 'wind' | 'dragon'
export type Wind = 'east' | 'south' | 'west' | 'north'
export type Dragon = 'red' | 'green' | 'white'

export type TileKind = MahjongSuit | HonorKind

export interface MahjongTile {
  id: string
  code: string
  kind: TileKind
  rank?: number
  wind?: Wind
  dragon?: Dragon
}

export type MeldKind = 'chow' | 'pung' | 'kong'

export interface MahjongMeld {
  id: string
  kind: MeldKind
  tiles: MahjongTile[]
  fromPlayerId?: string
  exposed: boolean
}

export interface MahjongPlayer {
  id: string
  name: string
  hand: MahjongTile[]
  melds: MahjongMeld[]
  discards: MahjongTile[]
  isHuman: boolean
  wind: Wind
}

export interface MahjongLogEntry {
  id: string
  text: string
}

export interface FanPattern {
  name: string
  fan: number
}

export interface WinResult {
  canWin: boolean
  fan: number
  patterns: FanPattern[]
  reason?: string
}

export interface ClaimOption {
  id: string
  playerId: string
  kind: 'hu' | 'peng' | 'chi'
  tile: MahjongTile
  tilesFromHand: MahjongTile[]
  winResult?: WinResult
}

export interface MahjongRuleset {
  id: string
  name: string
  minFan: number
  description: string
}

export interface MahjongGameState {
  players: MahjongPlayer[]
  wall: MahjongTile[]
  deadWall: MahjongTile[]
  currentPlayerIndex: number
  dealerIndex: number
  roundWind: Wind
  phase: 'playing' | 'claiming' | 'finished'
  hasDrawn: boolean
  lastDiscard?: {
    tile: MahjongTile
    playerId: string
  }
  claimOptions: ClaimOption[]
  ruleset: MahjongRuleset
  winnerId?: string
  winResult?: WinResult
  log: MahjongLogEntry[]
}
