import type { SocialGameSlug, SocialRole } from './online'

export const GAME_COPY = {
  werewolf: {
    titleKey: 'werewolf.title',
    subtitleKey: 'werewolf.subtitle',
    lobbyTitleKey: 'werewolf.lobbyTitle',
    lobbyDescriptionKey: 'werewolf.lobbyDescription',
    enterTitleKey: 'werewolf.enterTitle',
    rulesTitleKey: 'werewolf.rulesTitle',
    rulesKey: 'werewolf.rules',
    roomPrefix: 'WWF42',
    bg: 'bg-[#081018]',
    cover: '/game-covers/werewolf.webp',
    accent: 'text-[#ffd166]',
    panel: 'border-[#c8d7ff]/18 bg-[#101827]/76',
    button: 'border-[#c8d7ff]/20 bg-[#111b2a]/72 text-[#f7f1de] hover:bg-[#f7f1de] hover:text-[#111827]',
    primary: 'bg-[#ffd166] text-[#111827] hover:bg-[#ffe29a]',
  },
  avalon: {
    titleKey: 'avalon.title',
    subtitleKey: 'avalon.subtitle',
    lobbyTitleKey: 'avalon.lobbyTitle',
    lobbyDescriptionKey: 'avalon.lobbyDescription',
    enterTitleKey: 'avalon.enterTitle',
    rulesTitleKey: 'avalon.rulesTitle',
    rulesKey: 'avalon.rules',
    roomPrefix: 'AVL42',
    bg: 'bg-[#10140f]',
    cover: '/game-covers/avalon.webp',
    accent: 'text-[#d9b35f]',
    panel: 'border-[#f2dfad]/18 bg-[#162016]/78',
    button: 'border-[#f2dfad]/22 bg-[#1a2619]/72 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#172114]',
    primary: 'bg-[#d9b35f] text-[#172114] hover:bg-[#f2cf77]',
  },
  undercover: {
    titleKey: 'undercover.title',
    subtitleKey: 'undercover.subtitle',
    lobbyTitleKey: 'undercover.lobbyTitle',
    lobbyDescriptionKey: 'undercover.lobbyDescription',
    enterTitleKey: 'undercover.enterTitle',
    rulesTitleKey: 'undercover.rulesTitle',
    rulesKey: 'undercover.rules',
    roomPrefix: 'UND42',
    bg: 'bg-[#130f18]',
    cover: '/game-covers/undercover.svg',
    accent: 'text-[#f4c7ff]',
    panel: 'border-[#f4c7ff]/18 bg-[#211728]/78',
    button: 'border-[#f4c7ff]/22 bg-[#2a1b33]/72 text-[#fff8e8] hover:bg-[#fff8e8] hover:text-[#211728]',
    primary: 'bg-[#f4c7ff] text-[#211728] hover:bg-[#ffe2ff]',
  },
} satisfies Record<SocialGameSlug, Record<string, string>>

export const ROLES_BY_GAME: Record<SocialGameSlug, SocialRole[]> = {
  werewolf: ['villager', 'werewolf', 'seer', 'witch', 'hunter', 'idiot', 'guard'],
  avalon: ['merlin', 'assassin', 'minion', 'loyal'],
  undercover: ['civilian', 'undercover', 'blank'],
}

export const ROLE_ALIGNMENT: Record<SocialRole, 'good' | 'evil' | 'neutral'> = {
  assassin: 'evil',
  guard: 'good',
  hunter: 'good',
  idiot: 'good',
  loyal: 'good',
  merlin: 'good',
  minion: 'evil',
  seer: 'good',
  villager: 'good',
  werewolf: 'evil',
  witch: 'good',
  civilian: 'good',
  undercover: 'evil',
  blank: 'neutral',
}

export interface PlayerAccent {
  border: string
  ink: string
  soft: string
  solid: string
  text: string
}

export const PLAYER_COLOR_PALETTE = [
  { border: 'rgba(96,165,250,0.62)', ink: '#1d4ed8', soft: 'rgba(96,165,250,0.16)', solid: '#60a5fa', text: '#bfdbfe' },
  { border: 'rgba(251,113,133,0.62)', ink: '#be123c', soft: 'rgba(251,113,133,0.16)', solid: '#fb7185', text: '#fecdd3' },
  { border: 'rgba(52,211,153,0.62)', ink: '#047857', soft: 'rgba(52,211,153,0.16)', solid: '#34d399', text: '#a7f3d0' },
  { border: 'rgba(251,191,36,0.64)', ink: '#a16207', soft: 'rgba(251,191,36,0.18)', solid: '#fbbf24', text: '#fde68a' },
  { border: 'rgba(196,181,253,0.64)', ink: '#6d28d9', soft: 'rgba(196,181,253,0.17)', solid: '#c4b5fd', text: '#ddd6fe' },
  { border: 'rgba(45,212,191,0.62)', ink: '#0f766e', soft: 'rgba(45,212,191,0.16)', solid: '#2dd4bf', text: '#99f6e4' },
  { border: 'rgba(244,114,182,0.62)', ink: '#be185d', soft: 'rgba(244,114,182,0.16)', solid: '#f472b6', text: '#fbcfe8' },
  { border: 'rgba(129,140,248,0.62)', ink: '#4338ca', soft: 'rgba(129,140,248,0.16)', solid: '#818cf8', text: '#c7d2fe' },
  { border: 'rgba(251,146,60,0.62)', ink: '#c2410c', soft: 'rgba(251,146,60,0.16)', solid: '#fb923c', text: '#fed7aa' },
  { border: 'rgba(125,211,252,0.62)', ink: '#0369a1', soft: 'rgba(125,211,252,0.16)', solid: '#7dd3fc', text: '#bae6fd' },
  { border: 'rgba(163,230,53,0.62)', ink: '#4d7c0f', soft: 'rgba(163,230,53,0.16)', solid: '#a3e635', text: '#d9f99d' },
  { border: 'rgba(250,204,21,0.62)', ink: '#854d0e', soft: 'rgba(250,204,21,0.16)', solid: '#facc15', text: '#fef08a' },
] satisfies PlayerAccent[]
