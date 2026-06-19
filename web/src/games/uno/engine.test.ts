import type { UnoGameState } from './types'
import { describe, expect, it } from 'vitest'
import { createUnoGame, drawCard, getCurrentPlayer, getTopCard, isCardPlayable, playCard } from './engine'

describe('uno engine', () => {
  it('creates a playable game with three players and seven cards each', () => {
    const state = createUnoGame()

    expect(state.players).toHaveLength(3)
    expect(state.players.every(player => player.hand.length === 7)).toBe(true)
    expect(state.discardPile).toHaveLength(1)
    expect(state.status).toBe('playing')
  })

  it('lets the current player draw one card and advances the turn', () => {
    const state = createUnoGame()
    const player = getCurrentPlayer(state)
    const nextState = drawCard(state, player.id)

    expect(nextState.players[0].hand).toHaveLength(8)
    expect(nextState.currentPlayerIndex).toBe(1)
  })

  it('plays a matching color card', () => {
    const state = createMatchingCardState()
    const card = state.players[0].hand[0]
    const nextState = playCard(state, 'you', card.id)

    expect(getTopCard(nextState).id).toBe(card.id)
    expect(nextState.players[0].hand).toHaveLength(1)
    expect(nextState.status).toBe('playing')
  })

  it('rejects an unplayable card', () => {
    const state = createMatchingCardState()
    const blockedCard = state.players[0].hand[1]

    expect(isCardPlayable(blockedCard, state)).toBe(false)

    const nextState = playCard(state, 'you', blockedCard.id)

    expect(getTopCard(nextState).id).toBe('red-5')
    expect(nextState.players[0].hand).toHaveLength(2)
  })
})

function createMatchingCardState(): UnoGameState {
  return {
    activeColor: 'red',
    currentPlayerIndex: 0,
    direction: 1,
    discardPile: [{ id: 'red-5', color: 'red', kind: 'number', value: 5 }],
    drawPile: [],
    log: [],
    players: [
      {
        hand: [
          { id: 'red-7', color: 'red', kind: 'number', value: 7 },
          { id: 'blue-9', color: 'blue', kind: 'number', value: 9 },
        ],
        id: 'you',
        isHuman: true,
        name: '你',
      },
      {
        hand: [],
        id: 'bot',
        isHuman: false,
        name: '北风',
      },
    ],
    status: 'playing',
  }
}
