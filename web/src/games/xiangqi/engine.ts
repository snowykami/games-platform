import type { XiangqiGameState, XiangqiMove, XiangqiPiece, XiangqiPieceType, XiangqiPosition, XiangqiSide } from './types'

const BOARD_WIDTH = 9
const BOARD_HEIGHT = 10
const RED_PALACE_Y = [7, 8, 9]
const BLACK_PALACE_Y = [0, 1, 2]
const PALACE_X = [3, 4, 5]

export function createXiangqiGame(): XiangqiGameState {
  return {
    checkSide: undefined,
    message: '红方先行。',
    moveHistory: [],
    pieces: createInitialPieces(),
    status: 'playing',
    turn: 'red',
    winner: undefined,
  }
}

export function getPieceAt(pieces: XiangqiPiece[], position: XiangqiPosition) {
  return pieces.find(piece => samePosition(piece.position, position))
}

export function getLegalMoves(state: XiangqiGameState, pieceId: string) {
  const piece = state.pieces.find(item => item.id === pieceId)

  if (!piece || state.status !== 'playing') {
    return []
  }

  return getPseudoMoves(state.pieces, piece).filter(position => isLegalMove(state.pieces, piece, position))
}

export function getAllLegalMoves(state: XiangqiGameState, side: XiangqiSide) {
  if (state.status !== 'playing') {
    return []
  }

  return state.pieces
    .filter(piece => piece.side === side)
    .flatMap(piece => getLegalMoves({ ...state, turn: side }, piece.id).map(to => ({ piece, to })))
}

export function chooseAIMove(state: XiangqiGameState, side: XiangqiSide) {
  const moves = getAllLegalMoves(state, side)

  if (moves.length === 0) {
    return undefined
  }

  return [...moves].sort((first, second) => scoreAIMove(state, second.piece, second.to) - scoreAIMove(state, first.piece, first.to))[0]
}

export function movePiece(state: XiangqiGameState, pieceId: string, to: XiangqiPosition): XiangqiGameState {
  if (state.status !== 'playing') {
    return state
  }

  const piece = state.pieces.find(item => item.id === pieceId)

  if (!piece || piece.side !== state.turn) {
    return {
      ...state,
      message: '请选择当前回合的棋子。',
    }
  }

  const legalMoves = getLegalMoves(state, pieceId)

  if (!legalMoves.some(position => samePosition(position, to))) {
    return {
      ...state,
      message: '这一步不符合象棋走法。',
    }
  }

  const captured = getPieceAt(state.pieces, to)
  const nextPieces = applyMove(state.pieces, piece, to)
  const nextTurn = oppositeSide(piece.side)
  const nextSideInCheck = isSideInCheck(nextPieces, nextTurn)
  const nextSideCanMove = hasAnyLegalMove(nextPieces, nextTurn)
  const checkmate = !nextSideCanMove
  const move: XiangqiMove = {
    captured,
    check: nextSideInCheck,
    checkmate,
    from: piece.position,
    id: crypto.randomUUID(),
    piece,
    to,
  }

  if (captured?.type === 'general' || checkmate) {
    return {
      ...state,
      checkSide: nextSideInCheck ? nextTurn : undefined,
      message: `${formatSide(piece.side)}获胜。`,
      moveHistory: [...state.moveHistory, move],
      pieces: nextPieces,
      status: 'finished',
      winner: piece.side,
    }
  }

  return {
    ...state,
    checkSide: nextSideInCheck ? nextTurn : undefined,
    message: formatMoveMessage(move, nextTurn),
    moveHistory: [...state.moveHistory, move],
    pieces: nextPieces,
    turn: nextTurn,
  }
}

export function isSideInCheck(pieces: XiangqiPiece[], side: XiangqiSide) {
  const general = pieces.find(piece => piece.side === side && piece.type === 'general')

  if (!general) {
    return true
  }

  return pieces
    .filter(piece => piece.side !== side)
    .some(piece => getPseudoMoves(pieces, piece).some(position => samePosition(position, general.position)))
}

export function formatPiece(piece: XiangqiPiece) {
  const labels: Record<XiangqiSide, Record<XiangqiPieceType, string>> = {
    black: {
      advisor: '士',
      cannon: '砲',
      elephant: '象',
      general: '將',
      horse: '馬',
      rook: '車',
      soldier: '卒',
    },
    red: {
      advisor: '仕',
      cannon: '炮',
      elephant: '相',
      general: '帥',
      horse: '傌',
      rook: '俥',
      soldier: '兵',
    },
  }

  return labels[piece.side][piece.type]
}

export function formatSide(side: XiangqiSide) {
  return side === 'red' ? '红方' : '黑方'
}

export function oppositeSide(side: XiangqiSide): XiangqiSide {
  return side === 'red' ? 'black' : 'red'
}

function createInitialPieces(): XiangqiPiece[] {
  const pieces: XiangqiPiece[] = []

  addBackRank(pieces, 'black', 0)
  addBackRank(pieces, 'red', 9)
  addPieces(pieces, 'black', 'cannon', 2, [1, 7])
  addPieces(pieces, 'red', 'cannon', 7, [1, 7])
  addPieces(pieces, 'black', 'soldier', 3, [0, 2, 4, 6, 8])
  addPieces(pieces, 'red', 'soldier', 6, [0, 2, 4, 6, 8])

  return pieces
}

function addBackRank(pieces: XiangqiPiece[], side: XiangqiSide, y: number) {
  const order: XiangqiPieceType[] = ['rook', 'horse', 'elephant', 'advisor', 'general', 'advisor', 'elephant', 'horse', 'rook']

  order.forEach((type, x) => {
    pieces.push(createPiece(side, type, x, y))
  })
}

function addPieces(pieces: XiangqiPiece[], side: XiangqiSide, type: XiangqiPieceType, y: number, files: number[]) {
  files.forEach((x) => {
    pieces.push(createPiece(side, type, x, y))
  })
}

function createPiece(side: XiangqiSide, type: XiangqiPieceType, x: number, y: number): XiangqiPiece {
  return {
    id: `${side}-${type}-${x}-${y}`,
    position: { x, y },
    side,
    type,
  }
}

function getPseudoMoves(pieces: XiangqiPiece[], piece: XiangqiPiece) {
  if (piece.type === 'general') {
    return getGeneralMoves(pieces, piece)
  }
  if (piece.type === 'advisor') {
    return getAdvisorMoves(piece)
  }
  if (piece.type === 'elephant') {
    return getElephantMoves(pieces, piece)
  }
  if (piece.type === 'horse') {
    return getHorseMoves(pieces, piece)
  }
  if (piece.type === 'rook') {
    return getSlidingMoves(pieces, piece, false)
  }
  if (piece.type === 'cannon') {
    return getSlidingMoves(pieces, piece, true)
  }

  return getSoldierMoves(piece)
}

function getGeneralMoves(pieces: XiangqiPiece[], piece: XiangqiPiece) {
  const moves = [
    { x: piece.position.x + 1, y: piece.position.y },
    { x: piece.position.x - 1, y: piece.position.y },
    { x: piece.position.x, y: piece.position.y + 1 },
    { x: piece.position.x, y: piece.position.y - 1 },
  ].filter(position => isInsidePalace(position, piece.side))
  const opposingGeneral = pieces.find(item => item.side !== piece.side && item.type === 'general')

  if (
    opposingGeneral
    && opposingGeneral.position.x === piece.position.x
    && countPiecesBetween(pieces, piece.position, opposingGeneral.position) === 0
  ) {
    moves.push(opposingGeneral.position)
  }

  return moves
}

function getAdvisorMoves(piece: XiangqiPiece) {
  return [
    { x: piece.position.x + 1, y: piece.position.y + 1 },
    { x: piece.position.x + 1, y: piece.position.y - 1 },
    { x: piece.position.x - 1, y: piece.position.y + 1 },
    { x: piece.position.x - 1, y: piece.position.y - 1 },
  ].filter(position => isInsidePalace(position, piece.side))
}

function getElephantMoves(pieces: XiangqiPiece[], piece: XiangqiPiece) {
  return [
    { x: piece.position.x + 2, y: piece.position.y + 2 },
    { x: piece.position.x + 2, y: piece.position.y - 2 },
    { x: piece.position.x - 2, y: piece.position.y + 2 },
    { x: piece.position.x - 2, y: piece.position.y - 2 },
  ].filter((position) => {
    const eye = {
      x: piece.position.x + (position.x - piece.position.x) / 2,
      y: piece.position.y + (position.y - piece.position.y) / 2,
    }

    return isInsideBoard(position)
      && isOnOwnRiverSide(position, piece.side)
      && !getPieceAt(pieces, eye)
  })
}

function getHorseMoves(pieces: XiangqiPiece[], piece: XiangqiPiece) {
  const offsets = [
    { x: 1, y: 2 },
    { x: 2, y: 1 },
    { x: -1, y: 2 },
    { x: -2, y: 1 },
    { x: 1, y: -2 },
    { x: 2, y: -1 },
    { x: -1, y: -2 },
    { x: -2, y: -1 },
  ]

  return offsets
    .map(offset => ({
      leg: Math.abs(offset.x) === 2
        ? { x: piece.position.x + offset.x / 2, y: piece.position.y }
        : { x: piece.position.x, y: piece.position.y + offset.y / 2 },
      position: { x: piece.position.x + offset.x, y: piece.position.y + offset.y },
    }))
    .filter(item => isInsideBoard(item.position) && !getPieceAt(pieces, item.leg))
    .map(item => item.position)
}

function getSlidingMoves(pieces: XiangqiPiece[], piece: XiangqiPiece, isCannon: boolean) {
  const directions = [
    { x: 1, y: 0 },
    { x: -1, y: 0 },
    { x: 0, y: 1 },
    { x: 0, y: -1 },
  ]
  const moves: XiangqiPosition[] = []

  directions.forEach((direction) => {
    let position = { x: piece.position.x + direction.x, y: piece.position.y + direction.y }
    let screenFound = false

    while (isInsideBoard(position)) {
      const blockingPiece = getPieceAt(pieces, position)

      if (!isCannon) {
        moves.push(position)
        if (blockingPiece) {
          break
        }
      }
      else if (!screenFound) {
        if (blockingPiece) {
          screenFound = true
        }
        else {
          moves.push(position)
        }
      }
      else if (blockingPiece) {
        moves.push(position)
        break
      }

      position = { x: position.x + direction.x, y: position.y + direction.y }
    }
  })

  return moves
}

function getSoldierMoves(piece: XiangqiPiece) {
  const forward = piece.side === 'red' ? -1 : 1
  const moves = [{ x: piece.position.x, y: piece.position.y + forward }]

  if (hasCrossedRiver(piece)) {
    moves.push(
      { x: piece.position.x + 1, y: piece.position.y },
      { x: piece.position.x - 1, y: piece.position.y },
    )
  }

  return moves.filter(isInsideBoard)
}

function isLegalMove(pieces: XiangqiPiece[], piece: XiangqiPiece, position: XiangqiPosition) {
  const target = getPieceAt(pieces, position)

  if (target?.side === piece.side) {
    return false
  }

  return !isSideInCheck(applyMove(pieces, piece, position), piece.side)
}

function applyMove(pieces: XiangqiPiece[], piece: XiangqiPiece, to: XiangqiPosition) {
  return pieces
    .filter(item => item.id !== piece.id && !samePosition(item.position, to))
    .concat({ ...piece, position: to })
}

function hasAnyLegalMove(pieces: XiangqiPiece[], side: XiangqiSide) {
  const state: XiangqiGameState = {
    checkSide: isSideInCheck(pieces, side) ? side : undefined,
    message: '',
    moveHistory: [],
    pieces,
    status: 'playing',
    turn: side,
    winner: undefined,
  }

  return pieces
    .filter(piece => piece.side === side)
    .some(piece => getLegalMoves(state, piece.id).length > 0)
}

function formatMoveMessage(move: XiangqiMove, nextTurn: XiangqiSide) {
  const captureText = move.captured ? `，吃掉${formatSide(move.captured.side)}${formatPiece(move.captured)}` : ''
  const checkText = move.check ? '，将军' : ''

  return `${formatSide(move.piece.side)}${formatPiece(move.piece)}走到 ${formatPosition(move.to)}${captureText}${checkText}。${formatSide(nextTurn)}行棋。`
}

function scoreAIMove(state: XiangqiGameState, piece: XiangqiPiece, to: XiangqiPosition) {
  const captured = getPieceAt(state.pieces, to)
  const nextPieces = applyMove(state.pieces, piece, to)
  const opponent = oppositeSide(piece.side)
  let score = Math.random() * 0.1

  if (captured) {
    score += pieceValue(captured) * 10
  }
  if (captured?.type === 'general') {
    score += 10000
  }
  if (isSideInCheck(nextPieces, opponent)) {
    score += 35
  }
  if (!hasAnyLegalMove(nextPieces, opponent)) {
    score += 5000
  }
  if (isSideInCheck(nextPieces, piece.side)) {
    score -= 10000
  }

  score += (4 - Math.abs(to.x - 4)) * 0.4
  score += piece.side === 'red' ? (9 - to.y) * 0.12 : to.y * 0.12

  return score
}

function pieceValue(piece: XiangqiPiece) {
  const values: Record<XiangqiPieceType, number> = {
    advisor: 2,
    cannon: 4.5,
    elephant: 2,
    general: 1000,
    horse: 4,
    rook: 9,
    soldier: 1.5,
  }

  return values[piece.type]
}

function formatPosition(position: XiangqiPosition) {
  return `${position.x + 1}路${position.y + 1}线`
}

function countPiecesBetween(pieces: XiangqiPiece[], from: XiangqiPosition, to: XiangqiPosition) {
  if (from.x !== to.x && from.y !== to.y) {
    return 0
  }

  const stepX = Math.sign(to.x - from.x)
  const stepY = Math.sign(to.y - from.y)
  let position = { x: from.x + stepX, y: from.y + stepY }
  let count = 0

  while (!samePosition(position, to)) {
    if (getPieceAt(pieces, position)) {
      count += 1
    }
    position = { x: position.x + stepX, y: position.y + stepY }
  }

  return count
}

function isInsideBoard(position: XiangqiPosition) {
  return position.x >= 0 && position.x < BOARD_WIDTH && position.y >= 0 && position.y < BOARD_HEIGHT
}

function isInsidePalace(position: XiangqiPosition, side: XiangqiSide) {
  const palaceRanks = side === 'red' ? RED_PALACE_Y : BLACK_PALACE_Y

  return PALACE_X.includes(position.x) && palaceRanks.includes(position.y)
}

function isOnOwnRiverSide(position: XiangqiPosition, side: XiangqiSide) {
  return side === 'red' ? position.y >= 5 : position.y <= 4
}

function hasCrossedRiver(piece: XiangqiPiece) {
  return piece.side === 'red' ? piece.position.y <= 4 : piece.position.y >= 5
}

function samePosition(first: XiangqiPosition, second: XiangqiPosition) {
  return first.x === second.x && first.y === second.y
}
