import type { CardKind, UnoCard, UnoColor, UnoGameState, UnoLogEntry, UnoPlayer } from './types'

const COLORS: UnoColor[] = ['red', 'yellow', 'green', 'blue']
const ACTION_KINDS: CardKind[] = ['skip', 'reverse', 'draw-two']
let logId = 0

export function createUnoGame(): UnoGameState {
  const deck = shuffle(createDeck())
  const players: UnoPlayer[] = [
    { id: 'you', name: '你', hand: deck.splice(0, 7), isHuman: true },
    { id: 'bot-1', name: '北风', hand: deck.splice(0, 7), isHuman: false },
    { id: 'bot-2', name: '南星', hand: deck.splice(0, 7), isHuman: false },
  ]

  const firstNumberIndex = deck.findIndex(card => card.kind === 'number')
  const firstCard = deck.splice(firstNumberIndex, 1)[0]

  return {
    players,
    drawPile: deck,
    discardPile: [firstCard],
    currentPlayerIndex: 0,
    direction: 1,
    activeColor: firstCard.color as UnoColor,
    status: 'playing',
    log: [createLog(`开局弃牌是 ${formatCard(firstCard)}。`)],
  }
}

export function getTopCard(state: UnoGameState) {
  return state.discardPile.at(-1)!
}

export function getCurrentPlayer(state: UnoGameState) {
  return state.players[state.currentPlayerIndex]
}

export function isCardPlayable(card: UnoCard, state: UnoGameState) {
  const topCard = getTopCard(state)

  if (card.color === 'wild') {
    return true
  }

  if (card.color === state.activeColor) {
    return true
  }

  if (card.kind === 'number') {
    return card.value === topCard.value
  }

  return card.kind === topCard.kind
}

export function getPlayableCards(state: UnoGameState, player: UnoPlayer) {
  return player.hand.filter(card => isCardPlayable(card, state))
}

export function playCard(
  state: UnoGameState,
  playerId: string,
  cardId: string,
  chosenColor?: UnoColor,
): UnoGameState {
  if (state.status !== 'playing') {
    return state
  }

  const player = getCurrentPlayer(state)

  if (player.id !== playerId) {
    return addLog(state, '还没轮到这位玩家。')
  }

  const card = player.hand.find(item => item.id === cardId)

  if (!card || !isCardPlayable(card, state)) {
    return addLog(state, '这张牌现在不能打出。')
  }

  const nextPlayers = state.players.map(item =>
    item.id === playerId
      ? { ...item, hand: item.hand.filter(handCard => handCard.id !== cardId) }
      : item,
  )
  const activeColor = card.color === 'wild' ? chosenColor ?? chooseBestColor(player.hand) : card.color
  const afterPlay: UnoGameState = {
    ...state,
    players: nextPlayers,
    discardPile: [...state.discardPile, card],
    activeColor,
    log: [...state.log, createLog(`${player.name} 打出了 ${formatCard(card)}。`)],
  }

  const updatedPlayer = nextPlayers[state.currentPlayerIndex]

  if (updatedPlayer.hand.length === 0) {
    return {
      ...afterPlay,
      status: 'finished',
      winnerId: updatedPlayer.id,
      log: [...afterPlay.log, createLog(`${updatedPlayer.name} 获胜。`)],
    }
  }

  return applyCardEffect(afterPlay, card)
}

export function drawCard(state: UnoGameState, playerId: string): UnoGameState {
  if (state.status !== 'playing') {
    return state
  }

  const player = getCurrentPlayer(state)

  if (player.id !== playerId) {
    return addLog(state, '还没轮到这位玩家。')
  }

  const { cards, state: afterDraw } = drawCards(state, state.currentPlayerIndex, 1)

  return advanceTurn({
    ...afterDraw,
    log: [...afterDraw.log, createLog(`${player.name} 摸了 ${cards.length} 张牌并结束回合。`)],
  })
}

export function playBotTurn(state: UnoGameState): UnoGameState {
  const player = getCurrentPlayer(state)
  const playableCards = getPlayableCards(state, player)

  if (playableCards.length > 0) {
    const card = playableCards[0]
    return playCard(state, player.id, card.id, chooseBestColor(player.hand))
  }

  return drawCard(state, player.id)
}

export function formatCard(card: UnoCard) {
  if (card.kind === 'number') {
    return `${formatColor(card.color)} ${card.value}`
  }

  return `${formatColor(card.color)} ${formatKind(card.kind)}`
}

export function formatColor(color: CardColorLabel) {
  const labels: Record<CardColorLabel, string> = {
    red: '红色',
    yellow: '黄色',
    green: '绿色',
    blue: '蓝色',
    wild: '万能',
  }

  return labels[color]
}

function createDeck() {
  const deck: UnoCard[] = []

  for (const color of COLORS) {
    deck.push(createCard(color, 'number', 0))

    for (let value = 1; value <= 9; value += 1) {
      deck.push(createCard(color, 'number', value), createCard(color, 'number', value))
    }

    for (const kind of ACTION_KINDS) {
      deck.push(createCard(color, kind), createCard(color, kind))
    }
  }

  for (let index = 0; index < 4; index += 1) {
    deck.push(createCard('wild', 'wild'), createCard('wild', 'wild-draw-four'))
  }

  return deck
}

function createCard(color: UnoCard['color'], kind: CardKind, value?: number): UnoCard {
  const valuePart = value === undefined ? '' : `-${value}`

  return {
    id: `${color}-${kind}${valuePart}-${crypto.randomUUID()}`,
    color,
    kind,
    value,
  }
}

function shuffle(cards: UnoCard[]) {
  return [...cards].sort(() => Math.random() - 0.5)
}

function applyCardEffect(state: UnoGameState, card: UnoCard): UnoGameState {
  if (card.kind === 'reverse') {
    const reversedState: UnoGameState = {
      ...state,
      direction: state.direction === 1 ? -1 : 1,
      log: [...state.log, createLog('回合方向反转。')],
    }

    return advanceTurn(reversedState)
  }

  if (card.kind === 'skip') {
    const skippedPlayer = state.players[nextIndex(state)]
    return advanceTurn(advanceTurn({
      ...state,
      log: [...state.log, createLog(`${skippedPlayer.name} 被跳过。`)],
    }))
  }

  if (card.kind === 'draw-two') {
    return drawAndSkip(state, 2)
  }

  if (card.kind === 'wild-draw-four') {
    return drawAndSkip(state, 4)
  }

  return advanceTurn(state)
}

function drawAndSkip(state: UnoGameState, count: number): UnoGameState {
  const targetIndex = nextIndex(state)
  const target = state.players[targetIndex]
  const { cards, state: afterDraw } = drawCards(state, targetIndex, count)

  return advanceTurn(advanceTurn({
    ...afterDraw,
    log: [...afterDraw.log, createLog(`${target.name} 摸了 ${cards.length} 张牌并被跳过。`)],
  }))
}

function drawCards(state: UnoGameState, playerIndex: number, count: number) {
  const drawPile = [...state.drawPile]
  const discardPile = [...state.discardPile]
  const cards: UnoCard[] = []

  for (let index = 0; index < count; index += 1) {
    if (drawPile.length === 0 && discardPile.length > 1) {
      const topCard = discardPile.pop()!
      drawPile.push(...shuffle(discardPile))
      discardPile.length = 0
      discardPile.push(topCard)
    }

    const card = drawPile.pop()

    if (card) {
      cards.push(card)
    }
  }

  const players = state.players.map((player, index) =>
    index === playerIndex ? { ...player, hand: [...player.hand, ...cards] } : player,
  )

  return {
    cards,
    state: {
      ...state,
      players,
      drawPile,
      discardPile,
    },
  }
}

function advanceTurn(state: UnoGameState): UnoGameState {
  return {
    ...state,
    currentPlayerIndex: nextIndex(state),
  }
}

function nextIndex(state: UnoGameState) {
  const next = state.currentPlayerIndex + state.direction

  if (next < 0) {
    return state.players.length - 1
  }

  if (next >= state.players.length) {
    return 0
  }

  return next
}

function chooseBestColor(hand: UnoCard[]): UnoColor {
  const counts = COLORS.map(color => ({
    color,
    count: hand.filter(card => card.color === color).length,
  }))

  return counts.sort((left, right) => right.count - left.count)[0].color
}

function formatKind(kind: CardKind) {
  const labels: Record<CardKind, string> = {
    'number': '数字',
    'skip': '跳过',
    'reverse': '反转',
    'draw-two': '+2',
    'wild': '变色',
    'wild-draw-four': '+4',
  }

  return labels[kind]
}

function addLog(state: UnoGameState, text: string): UnoGameState {
  return {
    ...state,
    log: [...state.log, createLog(text)],
  }
}

function createLog(text: string): UnoLogEntry {
  logId += 1

  return {
    id: `log-${logId}`,
    text,
  }
}

type CardColorLabel = UnoCard['color']
