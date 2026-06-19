export type UnoColor = 'red' | 'yellow' | 'green' | 'blue'
export type WildColor = 'wild'
export type CardColor = UnoColor | WildColor
export type CardKind = 'number' | 'skip' | 'reverse' | 'draw-two' | 'wild' | 'wild-draw-four'

export interface UnoCard {
  id: string
  color: CardColor
  kind: CardKind
  value?: number
}

export interface UnoPlayer {
  id: string
  name: string
  hand: UnoCard[]
  isHuman: boolean
}

export interface UnoLogEntry {
  id: string
  text: string
}

export interface UnoGameState {
  players: UnoPlayer[]
  drawPile: UnoCard[]
  discardPile: UnoCard[]
  currentPlayerIndex: number
  direction: 1 | -1
  activeColor: UnoColor
  status: 'playing' | 'finished'
  winnerId?: string
  log: UnoLogEntry[]
}
