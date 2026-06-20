import type { FanPattern, MahjongMeld, MahjongRuleset, MahjongTile, WinResult } from './types'

export const CHINESE_OFFICIAL_RULESET: MahjongRuleset = {
  id: 'chinese-official',
  name: '国标麻将',
  minFan: 8,
  description: '首版实现国标 8 番起胡与常用番型子集，规则层可继续扩展。',
}

interface WinContext {
  concealedTiles: MahjongTile[]
  melds: MahjongMeld[]
  winningTile?: MahjongTile
  selfDraw: boolean
  seatWind: string
  roundWind: string
  ruleset: MahjongRuleset
}

interface Decomposition {
  pair: string
  melds: MeldCode[]
  isSevenPairs: boolean
}

interface MeldCode {
  kind: 'chow' | 'pung' | 'kong'
  codes: string[]
  exposed: boolean
}

export function evaluateWin(context: WinContext): WinResult {
  const sortedCodes = sortCodes(context.concealedTiles.map(tile => tile.code))
  const decompositions = getWinDecompositions(sortedCodes, context.melds)

  if (!decompositions.length) {
    return {
      canWin: false,
      fan: 0,
      patterns: [],
      reason: '牌型还没有组成 4 副面子加 1 对将，或七对。',
    }
  }

  const bestResult = decompositions
    .map(decomposition => scoreDecomposition(decomposition, context))
    .sort((left, right) => right.fan - left.fan)[0]

  if (bestResult.fan < context.ruleset.minFan) {
    return {
      ...bestResult,
      canWin: false,
      reason: `国标麻将至少 ${context.ruleset.minFan} 番起胡，当前 ${bestResult.fan} 番。`,
    }
  }

  return {
    ...bestResult,
    canWin: true,
  }
}

export function formatTile(tile: MahjongTile) {
  if (tile.kind === 'characters') {
    return `${tile.rank}万`
  }
  if (tile.kind === 'dots') {
    return `${tile.rank}筒`
  }
  if (tile.kind === 'bamboo') {
    return `${tile.rank}条`
  }
  if (tile.kind === 'wind') {
    const labels: Record<string, string> = {
      east: '东',
      north: '北',
      south: '南',
      west: '西',
    }

    return labels[tile.wind ?? 'east']
  }

  const labels: Record<string, string> = {
    green: '发',
    red: '中',
    white: '白',
  }

  return labels[tile.dragon ?? 'white']
}

export function sortTiles(tiles: MahjongTile[]) {
  return [...tiles].sort((left, right) => tileOrder(left) - tileOrder(right))
}

function scoreDecomposition(decomposition: Decomposition, context: WinContext): WinResult {
  const allMelds = [
    ...decomposition.melds,
    ...context.melds.map(meld => ({
      codes: meld.tiles.map(tile => tile.code),
      exposed: meld.exposed,
      kind: meld.kind,
    })),
  ]
  const allCodes = [
    ...context.concealedTiles.map(tile => tile.code),
    ...context.melds.flatMap(meld => meld.tiles.map(tile => tile.code)),
  ]
  const patterns: FanPattern[] = []

  if (decomposition.isSevenPairs) {
    patterns.push({ fan: 24, name: '七对' })
  }

  if (isPureOneSuit(allCodes)) {
    patterns.push({ fan: 24, name: '清一色' })
  }
  else if (isHalfFlush(allCodes)) {
    patterns.push({ fan: 6, name: '混一色' })
  }

  if (!decomposition.isSevenPairs && allMelds.every(meld => meld.kind === 'pung' || meld.kind === 'kong')) {
    patterns.push({ fan: 6, name: '碰碰和' })
  }

  if (!decomposition.isSevenPairs && allMelds.every(meld => meld.kind === 'chow') && !isHonorCode(decomposition.pair)) {
    patterns.push({ fan: 2, name: '平和' })
  }

  const dragonPungs = allMelds.filter(meld => (meld.kind === 'pung' || meld.kind === 'kong') && isDragonCode(meld.codes[0])).length
  for (let index = 0; index < dragonPungs; index += 1) {
    patterns.push({ fan: 2, name: '箭刻' })
  }

  if (allMelds.some(meld => (meld.kind === 'pung' || meld.kind === 'kong') && meld.codes[0] === context.seatWind)) {
    patterns.push({ fan: 2, name: '门风刻' })
  }

  if (allMelds.some(meld => (meld.kind === 'pung' || meld.kind === 'kong') && meld.codes[0] === context.roundWind)) {
    patterns.push({ fan: 2, name: '圈风刻' })
  }

  if (context.selfDraw) {
    patterns.push({ fan: 1, name: '自摸' })
  }

  if (!allCodes.some(isHonorCode)) {
    patterns.push({ fan: 1, name: '无字' })
  }

  if (!context.selfDraw && context.melds.every(meld => !meld.exposed)) {
    patterns.push({ fan: 2, name: '门前清' })
  }

  const fan = patterns.reduce((sum, pattern) => sum + pattern.fan, 0)

  return {
    canWin: false,
    fan,
    patterns,
  }
}

function getWinDecompositions(codes: string[], melds: MahjongMeld[]): Decomposition[] {
  const neededConcealedMelds = 4 - melds.length
  const decompositions: Decomposition[] = []

  if (melds.length === 0 && isSevenPairs(codes)) {
    decompositions.push({
      isSevenPairs: true,
      melds: [],
      pair: codes[0],
    })
  }

  if ((codes.length - 2) / 3 !== neededConcealedMelds) {
    return decompositions
  }

  const counts = toCountMap(codes)

  for (const [pairCode, count] of counts) {
    if (count < 2) {
      continue
    }

    const nextCounts = new Map(counts)
    removeCode(nextCounts, pairCode, 2)
    const meldResults = findMelds(nextCounts, neededConcealedMelds)

    meldResults.forEach((result) => {
      decompositions.push({
        isSevenPairs: false,
        melds: result,
        pair: pairCode,
      })
    })
  }

  return decompositions
}

function findMelds(counts: Map<string, number>, remaining: number): MeldCode[][] {
  if (remaining === 0) {
    return totalCount(counts) === 0 ? [[]] : []
  }

  const firstCode = firstRemainingCode(counts)

  if (!firstCode) {
    return []
  }

  const results: MeldCode[][] = []
  const count = counts.get(firstCode) ?? 0

  if (count >= 3) {
    const nextCounts = new Map(counts)
    removeCode(nextCounts, firstCode, 3)
    findMelds(nextCounts, remaining - 1).forEach((result) => {
      results.push([{ codes: [firstCode, firstCode, firstCode], exposed: false, kind: 'pung' }, ...result])
    })
  }

  const chowCodes = getChowCodes(firstCode)
  if (chowCodes && chowCodes.every(code => (counts.get(code) ?? 0) > 0)) {
    const nextCounts = new Map(counts)
    chowCodes.forEach(code => removeCode(nextCounts, code, 1))
    findMelds(nextCounts, remaining - 1).forEach((result) => {
      results.push([{ codes: chowCodes, exposed: false, kind: 'chow' }, ...result])
    })
  }

  return results
}

function isSevenPairs(codes: string[]) {
  if (codes.length !== 14) {
    return false
  }

  const pairCount = [...toCountMap(codes).values()].reduce((sum, count) => {
    if (count === 2) {
      return sum + 1
    }
    if (count === 4) {
      return sum + 2
    }
    return sum
  }, 0)

  return pairCount === 7
}

function getChowCodes(code: string) {
  const suit = code[0]
  const rank = Number(code[1])

  if (!['m', 'p', 's'].includes(suit) || !Number.isInteger(rank) || rank > 7) {
    return undefined
  }

  return [`${suit}${rank}`, `${suit}${rank + 1}`, `${suit}${rank + 2}`]
}

function isPureOneSuit(codes: string[]) {
  const suits = new Set(codes.filter(code => !isHonorCode(code)).map(code => code[0]))

  return suits.size === 1 && !codes.some(isHonorCode)
}

function isHalfFlush(codes: string[]) {
  const suits = new Set(codes.filter(code => !isHonorCode(code)).map(code => code[0]))

  return suits.size === 1 && codes.some(isHonorCode)
}

function isHonorCode(code: string) {
  return code.startsWith('z')
}

function isDragonCode(code: string) {
  return code === 'z5' || code === 'z6' || code === 'z7'
}

function toCountMap(codes: string[]) {
  const counts = new Map<string, number>()

  codes.forEach((code) => {
    counts.set(code, (counts.get(code) ?? 0) + 1)
  })

  return counts
}

function sortCodes(codes: string[]) {
  return [...codes].sort((left, right) => codeOrder(left) - codeOrder(right))
}

function firstRemainingCode(counts: Map<string, number>) {
  return [...counts.entries()]
    .filter(([, count]) => count > 0)
    .sort(([left], [right]) => codeOrder(left) - codeOrder(right))[0]?.[0]
}

function removeCode(counts: Map<string, number>, code: string, amount: number) {
  counts.set(code, (counts.get(code) ?? 0) - amount)
}

function totalCount(counts: Map<string, number>) {
  return [...counts.values()].reduce((sum, count) => sum + Math.max(count, 0), 0)
}

function tileOrder(tile: MahjongTile) {
  return codeOrder(tile.code)
}

function codeOrder(code: string) {
  const suitOrder: Record<string, number> = {
    m: 0,
    p: 20,
    s: 40,
    z: 60,
  }

  return (suitOrder[code[0]] ?? 80) + Number(code[1])
}
