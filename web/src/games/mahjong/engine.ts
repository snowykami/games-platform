import type { ClaimOption, Dragon, MahjongGameState, MahjongMeld, MahjongPlayer, MahjongRuleset, MahjongTile, Wind, WinResult } from './types'
import { CHINESE_OFFICIAL_RULESET, evaluateWin, formatTile, sortTiles } from './scoring'

const PLAYER_NAMES = ['你', '南山', '西楼', '北川']
const WINDS: Wind[] = ['east', 'south', 'west', 'north']
const SUITS = [
  { code: 'm', kind: 'characters' as const },
  { code: 'p', kind: 'dots' as const },
  { code: 's', kind: 'bamboo' as const },
]
const WIND_CODES: Record<Wind, string> = {
  east: 'z1',
  north: 'z4',
  south: 'z2',
  west: 'z3',
}
const DRAGON_CODES: Record<Dragon, string> = {
  green: 'z6',
  red: 'z5',
  white: 'z7',
}

let logId = 0

export function createMahjongGame(ruleset: MahjongRuleset = CHINESE_OFFICIAL_RULESET): MahjongGameState {
  const wall = shuffle(createWall())
  const players: MahjongPlayer[] = PLAYER_NAMES.map((name, index) => ({
    discards: [],
    hand: sortTiles(wall.splice(0, index === 0 ? 14 : 13)),
    id: index === 0 ? 'you' : `bot-${index}`,
    isHuman: index === 0,
    melds: [],
    name,
    wind: WINDS[index],
  }))

  return {
    claimOptions: [],
    currentPlayerIndex: 0,
    dealerIndex: 0,
    deadWall: wall.splice(-14),
    hasDrawn: true,
    log: [createLog('东风起局，庄家先打。国标麻将 8 番起胡。')],
    phase: 'playing',
    players,
    roundWind: 'east',
    ruleset,
    wall,
  }
}

export function getCurrentPlayer(state: MahjongGameState) {
  return state.players[state.currentPlayerIndex]
}

export function getHumanPlayer(state: MahjongGameState) {
  return state.players.find(player => player.isHuman)!
}

export function drawTile(state: MahjongGameState, playerId: string): MahjongGameState {
  if (state.phase !== 'playing' || state.hasDrawn) {
    return state
  }

  const player = getCurrentPlayer(state)
  if (player.id !== playerId) {
    return addLog(state, '还没轮到这位玩家摸牌。')
  }

  if (state.wall.length === 0) {
    return {
      ...state,
      log: [...state.log, createLog('流局：牌墙已经摸完。')],
      phase: 'finished',
    }
  }

  const wall = [...state.wall]
  const tile = wall.shift()!
  const players = state.players.map(item =>
    item.id === playerId ? { ...item, hand: sortTiles([...item.hand, tile]) } : item,
  )

  return {
    ...state,
    hasDrawn: true,
    lastDiscard: undefined,
    log: [...state.log, createLog(`${player.name} 摸牌。`)],
    players,
    wall,
  }
}

export function discardTile(state: MahjongGameState, playerId: string, tileId: string): MahjongGameState {
  if (state.phase !== 'playing' || !state.hasDrawn) {
    return state
  }

  const player = getCurrentPlayer(state)
  if (player.id !== playerId) {
    return addLog(state, '还没轮到这位玩家打牌。')
  }

  const tile = player.hand.find(item => item.id === tileId)
  if (!tile) {
    return addLog(state, '这张牌不在当前玩家手里。')
  }

  const players = state.players.map((item) => {
    if (item.id !== playerId) {
      return item
    }

    return {
      ...item,
      discards: [...item.discards, tile],
      hand: item.hand.filter(handTile => handTile.id !== tileId),
    }
  })

  const afterDiscard: MahjongGameState = {
    ...state,
    hasDrawn: false,
    lastDiscard: { playerId, tile },
    log: [...state.log, createLog(`${player.name} 打出 ${formatTile(tile)}。`)],
    players,
  }

  return openClaimWindow(afterDiscard)
}

export function claimOption(state: MahjongGameState, claimId: string): MahjongGameState {
  if (state.phase !== 'claiming' || !state.lastDiscard) {
    return state
  }

  const claim = state.claimOptions.find(option => option.id === claimId)
  if (!claim) {
    return addLog(state, '这个声明已经不可用。')
  }

  const claimer = state.players.find(player => player.id === claim.playerId)!
  const discarder = state.players.find(player => player.id === state.lastDiscard?.playerId)!

  if (claim.kind === 'hu' && claim.winResult?.canWin) {
    return {
      ...state,
      claimOptions: [],
      log: [...state.log, createLog(`${claimer.name} 荣和 ${formatTile(claim.tile)}，${claim.winResult.fan} 番。`)],
      phase: 'finished',
      winResult: claim.winResult,
      winnerId: claimer.id,
    }
  }

  const meld: MahjongMeld = {
    exposed: true,
    fromPlayerId: discarder.id,
    id: `meld-${claim.kind}-${claim.tile.code}-${createId()}`,
    kind: claim.kind === 'chi' ? 'chow' : 'pung',
    tiles: sortTiles([...claim.tilesFromHand, claim.tile]),
  }
  const players = state.players.map((player) => {
    if (player.id === discarder.id) {
      return {
        ...player,
        discards: player.discards.filter(tile => tile.id !== claim.tile.id),
      }
    }
    if (player.id === claimer.id) {
      const usedIds = new Set(claim.tilesFromHand.map(tile => tile.id))

      return {
        ...player,
        hand: player.hand.filter(tile => !usedIds.has(tile.id)),
        melds: [...player.melds, meld],
      }
    }
    return player
  })

  return {
    ...state,
    claimOptions: [],
    currentPlayerIndex: state.players.findIndex(player => player.id === claimer.id),
    hasDrawn: true,
    lastDiscard: undefined,
    log: [...state.log, createLog(`${claimer.name} ${claim.kind === 'chi' ? '吃' : '碰'}了 ${formatTile(claim.tile)}。`)],
    phase: 'playing',
    players,
  }
}

export function skipClaims(state: MahjongGameState, playerId: string): MahjongGameState {
  if (state.phase !== 'claiming' || !state.lastDiscard) {
    return state
  }

  const remainingClaims = state.claimOptions.filter(option => option.playerId !== playerId)
  if (remainingClaims.length > 0) {
    return {
      ...state,
      claimOptions: remainingClaims,
      log: [...state.log, createLog('暂不声明。')],
    }
  }

  return advanceAfterClaimWindow({
    ...state,
    claimOptions: [],
    log: [...state.log, createLog('无人吃碰胡，继续摸牌。')],
    phase: 'playing',
  })
}

export function getSelfDrawWinResult(state: MahjongGameState, playerId: string): WinResult {
  const player = state.players.find(item => item.id === playerId)

  if (!player || state.phase !== 'playing' || !state.hasDrawn) {
    return emptyWinResult()
  }

  return evaluateWin({
    concealedTiles: player.hand,
    melds: player.melds,
    roundWind: WIND_CODES[state.roundWind],
    ruleset: state.ruleset,
    seatWind: WIND_CODES[player.wind],
    selfDraw: true,
  })
}

export function declareSelfDraw(state: MahjongGameState, playerId: string): MahjongGameState {
  const player = state.players.find(item => item.id === playerId)
  const winResult = getSelfDrawWinResult(state, playerId)

  if (!player || !winResult.canWin) {
    return addLog(state, winResult.reason ?? '现在还不能胡牌。')
  }

  return {
    ...state,
    log: [...state.log, createLog(`${player.name} 自摸，${winResult.fan} 番。`)],
    phase: 'finished',
    winResult,
    winnerId: playerId,
  }
}

export function playBotTurn(state: MahjongGameState): MahjongGameState {
  if (state.phase !== 'playing') {
    return state
  }

  const player = getCurrentPlayer(state)
  if (player.isHuman) {
    return state
  }

  if (!state.hasDrawn) {
    const afterDraw = drawTile(state, player.id)
    const winResult = getSelfDrawWinResult(afterDraw, player.id)

    if (winResult.canWin) {
      return declareSelfDraw(afterDraw, player.id)
    }

    return afterDraw
  }

  const tile = chooseBotDiscard(player)
  return discardTile(state, player.id, tile.id)
}

export function playBotDiscard(state: MahjongGameState, tileId: string): MahjongGameState {
  const player = getCurrentPlayer(state)
  if (state.phase !== 'playing' || player.isHuman || !state.hasDrawn) {
    return state
  }
  return discardTile(state, player.id, tileId)
}

export function formatWind(wind: Wind) {
  const labels: Record<Wind, string> = {
    east: '东',
    north: '北',
    south: '南',
    west: '西',
  }

  return labels[wind]
}

function openClaimWindow(state: MahjongGameState): MahjongGameState {
  if (!state.lastDiscard) {
    return state
  }

  const options = createClaimOptions(state)
  const botHu = options.find(option => option.kind === 'hu' && state.players.find(player => player.id === option.playerId)?.isHuman === false)

  if (botHu) {
    return claimOption({ ...state, claimOptions: [botHu], phase: 'claiming' }, botHu.id)
  }

  const humanClaims = options.filter(option => state.players.find(player => player.id === option.playerId)?.isHuman)
  if (humanClaims.length > 0) {
    return {
      ...state,
      claimOptions: humanClaims,
      log: [...state.log, createLog('你可以声明吃、碰或胡。')],
      phase: 'claiming',
    }
  }

  return advanceAfterClaimWindow(state)
}

function advanceAfterClaimWindow(state: MahjongGameState): MahjongGameState {
  if (!state.lastDiscard) {
    return state
  }

  return {
    ...state,
    currentPlayerIndex: nextPlayerIndex(state, state.lastDiscard.playerId),
    hasDrawn: false,
    lastDiscard: undefined,
    phase: 'playing',
  }
}

function createClaimOptions(state: MahjongGameState): ClaimOption[] {
  const discard = state.lastDiscard
  if (!discard) {
    return []
  }

  const discarderIndex = state.players.findIndex(player => player.id === discard.playerId)
  const nextIndex = (discarderIndex + 1) % state.players.length

  return state.players.flatMap((player, index) => {
    if (player.id === discard.playerId) {
      return []
    }

    const options: ClaimOption[] = []
    const winResult = evaluateWin({
      concealedTiles: sortTiles([...player.hand, discard.tile]),
      melds: player.melds,
      roundWind: WIND_CODES[state.roundWind],
      ruleset: state.ruleset,
      seatWind: WIND_CODES[player.wind],
      selfDraw: false,
    })

    if (winResult.canWin) {
      options.push({
        id: `hu-${player.id}-${discard.tile.id}`,
        kind: 'hu',
        playerId: player.id,
        tile: discard.tile,
        tilesFromHand: [],
        winResult,
      })
    }

    const sameTiles = player.hand.filter(tile => tile.code === discard.tile.code)
    if (sameTiles.length >= 2) {
      options.push({
        id: `peng-${player.id}-${discard.tile.id}`,
        kind: 'peng',
        playerId: player.id,
        tile: discard.tile,
        tilesFromHand: sameTiles.slice(0, 2),
      })
    }

    if (index === nextIndex) {
      options.push(...createChiOptions(player, discard.tile))
    }

    return options
  })
}

function createChiOptions(player: MahjongPlayer, tile: MahjongTile): ClaimOption[] {
  if (!tile.rank || !['characters', 'dots', 'bamboo'].includes(tile.kind)) {
    return []
  }

  const suitPrefix = tile.code[0]
  const windows = [
    [tile.rank - 2, tile.rank - 1],
    [tile.rank - 1, tile.rank + 1],
    [tile.rank + 1, tile.rank + 2],
  ].filter(ranks => ranks.every(rank => rank >= 1 && rank <= 9))

  return windows.flatMap((ranks) => {
    const handTiles = ranks.map(rank => player.hand.find(handTile => handTile.code === `${suitPrefix}${rank}`))

    if (handTiles.some(handTile => !handTile)) {
      return []
    }

    const tilesFromHand = handTiles as MahjongTile[]

    return [{
      id: `chi-${player.id}-${tile.id}-${tilesFromHand.map(handTile => handTile.id).join('-')}`,
      kind: 'chi' as const,
      playerId: player.id,
      tile,
      tilesFromHand,
    }]
  })
}

function nextPlayerIndex(state: MahjongGameState, playerId: string) {
  const index = state.players.findIndex(player => player.id === playerId)

  return (index + 1) % state.players.length
}

function chooseBotDiscard(player: MahjongPlayer) {
  const counts = new Map<string, number>()
  player.hand.forEach(tile => counts.set(tile.code, (counts.get(tile.code) ?? 0) + 1))

  return [...player.hand]
    .sort((left, right) => discardScore(right, counts) - discardScore(left, counts))[0]
}

function discardScore(tile: MahjongTile, counts: Map<string, number>) {
  const sameCount = counts.get(tile.code) ?? 0

  if (tile.kind === 'wind' || tile.kind === 'dragon') {
    return sameCount === 1 ? 8 : 1
  }

  const neighborCount = [-2, -1, 1, 2].filter(offset => counts.has(`${tile.code[0]}${(tile.rank ?? 0) + offset}`)).length

  return 6 - sameCount * 2 - neighborCount
}

function createWall() {
  const wall: MahjongTile[] = []

  SUITS.forEach(({ code, kind }) => {
    for (let rank = 1; rank <= 9; rank += 1) {
      for (let copy = 0; copy < 4; copy += 1) {
        wall.push({
          code: `${code}${rank}`,
          id: `${code}${rank}-${copy}-${createId()}`,
          kind,
          rank,
        })
      }
    }
  })

  WINDS.forEach((wind) => {
    for (let copy = 0; copy < 4; copy += 1) {
      wall.push({
        code: WIND_CODES[wind],
        id: `${WIND_CODES[wind]}-${copy}-${createId()}`,
        kind: 'wind',
        wind,
      })
    }
  })

  Object.entries(DRAGON_CODES).forEach(([dragon, code]) => {
    for (let copy = 0; copy < 4; copy += 1) {
      wall.push({
        code,
        dragon: dragon as Dragon,
        id: `${code}-${copy}-${createId()}`,
        kind: 'dragon',
      })
    }
  })

  return wall
}

function shuffle<T>(items: T[]) {
  return [...items].sort(() => Math.random() - 0.5)
}

function addLog(state: MahjongGameState, text: string): MahjongGameState {
  return {
    ...state,
    log: [...state.log, createLog(text)],
  }
}

function createLog(text: string) {
  logId += 1

  return {
    id: `mahjong-log-${logId}`,
    text,
  }
}

function emptyWinResult(): WinResult {
  return {
    canWin: false,
    fan: 0,
    patterns: [],
  }
}

function createId() {
  return globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2)
}
