import type { MahjongGameState, MahjongPlayer, MahjongTile, Wind } from './types'
import { describe, expect, it } from 'vitest'
import { createMahjongGame, discardTile, drawTile } from './engine'
import { CHINESE_OFFICIAL_RULESET, evaluateWin } from './scoring'

describe('mahjong engine', () => {
  it('creates a four-player chinese official mahjong game', () => {
    const state = createMahjongGame()

    expect(state.players).toHaveLength(4)
    expect(state.players[0].hand).toHaveLength(14)
    expect(state.players.slice(1).every(player => player.hand.length === 13)).toBe(true)
    expect(state.deadWall).toHaveLength(14)
    expect(state.ruleset.id).toBe('chinese-official')
    expect(state.hasDrawn).toBe(true)
  })

  it('lets the current player discard and advances to the next draw', () => {
    const state = createSimpleState()
    const discardedTile = state.players[0].hand[0]
    const nextState = discardTile(state, 'you', discardedTile.id)

    expect(nextState.players[0].hand).toHaveLength(13)
    expect(nextState.players[0].discards[0].code).toBe(discardedTile.code)
    expect(nextState.currentPlayerIndex).toBe(1)
    expect(nextState.hasDrawn).toBe(false)

    const afterDraw = drawTile(nextState, 'bot-1')
    expect(afterDraw.players[1].hand).toHaveLength(14)
    expect(afterDraw.hasDrawn).toBe(true)
  })

  it('enforces the chinese official eight-fan minimum', () => {
    const sevenPairs = evaluateWin({
      concealedTiles: [
        tile('m1', 0),
        tile('m1', 1),
        tile('m2', 0),
        tile('m2', 1),
        tile('m3', 0),
        tile('m3', 1),
        tile('m4', 0),
        tile('m4', 1),
        tile('m5', 0),
        tile('m5', 1),
        tile('m6', 0),
        tile('m6', 1),
        tile('m7', 0),
        tile('m7', 1),
      ],
      melds: [],
      roundWind: 'z1',
      ruleset: CHINESE_OFFICIAL_RULESET,
      seatWind: 'z1',
      selfDraw: true,
    })

    const lowFan = evaluateWin({
      concealedTiles: [
        tile('m1', 0),
        tile('m2', 0),
        tile('m3', 0),
        tile('p1', 0),
        tile('p2', 0),
        tile('p3', 0),
        tile('s1', 0),
        tile('s2', 0),
        tile('s3', 0),
        tile('m4', 0),
        tile('m5', 0),
        tile('m6', 0),
        tile('p9', 0),
        tile('p9', 1),
      ],
      melds: [],
      roundWind: 'z1',
      ruleset: CHINESE_OFFICIAL_RULESET,
      seatWind: 'z1',
      selfDraw: false,
    })

    expect(sevenPairs.canWin).toBe(true)
    expect(sevenPairs.fan).toBeGreaterThanOrEqual(8)
    expect(lowFan.canWin).toBe(false)
    expect(lowFan.reason).toContain('至少 8 番')
  })
})

function createSimpleState(): MahjongGameState {
  return {
    claimOptions: [],
    currentPlayerIndex: 0,
    dealerIndex: 0,
    deadWall: [],
    hasDrawn: true,
    lastDiscard: undefined,
    log: [],
    phase: 'playing',
    players: [
      player('you', '你', 'east', true, [
        'z7',
        'm1',
        'm2',
        'm3',
        'p1',
        'p2',
        'p3',
        's1',
        's2',
        's3',
        'm4',
        'm5',
        'm6',
        'p9',
      ]),
      player('bot-1', '南山', 'south', false, ['m7', 'm8', 'm9', 'p4', 'p5', 'p6', 's4', 's5', 's6', 'z1', 'z2', 'z3', 'z4']),
      player('bot-2', '西楼', 'west', false, ['m1', 'p1', 's1', 'm2', 'p2', 's2', 'm3', 'p3', 's3', 'm4', 'p4', 's4', 'z5']),
      player('bot-3', '北川', 'north', false, ['m5', 'p5', 's5', 'm6', 'p6', 's6', 'm7', 'p7', 's7', 'm8', 'p8', 's8', 'z6']),
    ],
    roundWind: 'east',
    ruleset: CHINESE_OFFICIAL_RULESET,
    wall: [tile('z1', 9)],
  }
}

function player(id: string, name: string, wind: Wind, isHuman: boolean, codes: string[]): MahjongPlayer {
  return {
    discards: [],
    hand: codes.map((code, index) => tile(code, index)),
    id,
    isHuman,
    melds: [],
    name,
    wind,
  }
}

function tile(code: string, copy: number): MahjongTile {
  const suitMap = {
    m: 'characters',
    p: 'dots',
    s: 'bamboo',
  } as const

  if (code[0] !== 'z') {
    return {
      code,
      id: `${code}-${copy}`,
      kind: suitMap[code[0] as 'm' | 'p' | 's'],
      rank: Number(code[1]),
    }
  }

  const windMap = {
    z1: 'east',
    z2: 'south',
    z3: 'west',
    z4: 'north',
  } as const
  const dragonMap = {
    z5: 'red',
    z6: 'green',
    z7: 'white',
  } as const

  if (code in windMap) {
    return {
      code,
      id: `${code}-${copy}`,
      kind: 'wind',
      wind: windMap[code as keyof typeof windMap],
    }
  }

  return {
    code,
    dragon: dragonMap[code as keyof typeof dragonMap],
    id: `${code}-${copy}`,
    kind: 'dragon',
  }
}
