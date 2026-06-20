import type { XiangqiGameState, XiangqiPiece } from './types'
import { describe, expect, it } from 'vitest'
import { createXiangqiGame, getLegalMoves, isSideInCheck, movePiece } from './engine'

describe('xiangqi engine', () => {
  it('creates the standard 32-piece opening with red to move', () => {
    const state = createXiangqiGame()

    expect(state.pieces).toHaveLength(32)
    expect(state.turn).toBe('red')
    expect(state.status).toBe('playing')
    expect(state.pieces.find(piece => piece.id === 'red-general-4-9')?.position).toEqual({ x: 4, y: 9 })
    expect(state.pieces.find(piece => piece.id === 'black-general-4-0')?.position).toEqual({ x: 4, y: 0 })
  })

  it('blocks a horse when its leg is occupied', () => {
    const state = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 3, 0),
      piece('red', 'horse', 1, 9),
      piece('red', 'soldier', 1, 8),
    ])

    expect(getLegalMoves(state, 'red-horse-1-9')).not.toContainEqual({ x: 0, y: 7 })
    expect(getLegalMoves(state, 'red-horse-1-9')).toContainEqual({ x: 3, y: 8 })
  })

  it('requires exactly one screen for cannon captures', () => {
    const state = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 3, 0),
      piece('red', 'cannon', 1, 7),
      piece('red', 'soldier', 1, 5),
      piece('black', 'soldier', 1, 3),
    ])

    expect(getLegalMoves(state, 'red-cannon-1-7')).toContainEqual({ x: 1, y: 3 })
    expect(getLegalMoves(state, 'red-cannon-1-7')).not.toContainEqual({ x: 1, y: 4 })
  })

  it('lets soldiers move sideways only after crossing the river', () => {
    const beforeRiver = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 3, 0),
      piece('red', 'soldier', 4, 6),
    ])
    const afterRiver = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 3, 0),
      piece('red', 'soldier', 4, 4),
    ])

    expect(getLegalMoves(beforeRiver, 'red-soldier-4-6')).toEqual([{ x: 4, y: 5 }])
    expect(getLegalMoves(afterRiver, 'red-soldier-4-4')).toEqual(expect.arrayContaining([{ x: 4, y: 3 }, { x: 3, y: 4 }, { x: 5, y: 4 }]))
  })

  it('prevents moves that expose the two generals to each other', () => {
    const state = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 4, 0),
      piece('red', 'rook', 4, 5),
    ])

    expect(isSideInCheck(state.pieces.filter(item => item.type !== 'rook'), 'red')).toBe(true)
    expect(getLegalMoves(state, 'red-rook-4-5')).not.toContainEqual({ x: 3, y: 5 })
  })

  it('finishes the game on checkmate or capture of the general', () => {
    const state = stateWithPieces([
      piece('red', 'general', 4, 9),
      piece('black', 'general', 4, 0),
      piece('red', 'rook', 4, 1),
    ])

    const nextState = movePiece(state, 'red-rook-4-1', { x: 4, y: 0 })

    expect(nextState.status).toBe('finished')
    expect(nextState.winner).toBe('red')
  })
})

function stateWithPieces(pieces: XiangqiPiece[]): XiangqiGameState {
  return {
    checkSide: undefined,
    message: '',
    moveHistory: [],
    pieces,
    status: 'playing',
    turn: 'red',
    winner: undefined,
  }
}

function piece(side: XiangqiPiece['side'], type: XiangqiPiece['type'], x: number, y: number): XiangqiPiece {
  return {
    id: `${side}-${type}-${x}-${y}`,
    position: { x, y },
    side,
    type,
  }
}
