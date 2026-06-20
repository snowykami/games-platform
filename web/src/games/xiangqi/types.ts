export type XiangqiSide = 'red' | 'black'
export type XiangqiPieceType = 'general' | 'advisor' | 'elephant' | 'horse' | 'rook' | 'cannon' | 'soldier'
export type XiangqiStatus = 'playing' | 'finished'

export interface XiangqiPosition {
  x: number
  y: number
}

export interface XiangqiPiece {
  id: string
  side: XiangqiSide
  type: XiangqiPieceType
  position: XiangqiPosition
}

export interface XiangqiMove {
  id: string
  piece: XiangqiPiece
  from: XiangqiPosition
  to: XiangqiPosition
  captured?: XiangqiPiece
  check: boolean
  checkmate: boolean
}

export interface XiangqiGameState {
  pieces: XiangqiPiece[]
  turn: XiangqiSide
  status: XiangqiStatus
  moveHistory: XiangqiMove[]
  message: string
  winner?: XiangqiSide
  checkSide?: XiangqiSide
}
