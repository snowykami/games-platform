import type { SocialGameSlug, SocialRole, WerewolfRoleCounts } from './online'
import type { GAME_COPY } from './socialTheme'
import { cn } from '@/shared/lib/utils'

export function alignmentClass(alignment: 'good' | 'evil' | 'neutral') {
  if (alignment === 'evil') {
    return 'bg-[#4a1424] text-[#ffd6df] ring-1 ring-[#ff7a9a]/35'
  }
  if (alignment === 'neutral') {
    return 'bg-[#e7d5ff] text-[#241833] ring-1 ring-[#f4c7ff]/45'
  }
  return 'bg-[#fff0b8] text-[#1f2114] ring-1 ring-[#ffd166]/45'
}

export function roleTotal(counts: WerewolfRoleCounts) {
  return (counts.villager ?? 0)
    + (counts.werewolf ?? 0)
    + (counts.seer ?? 0)
    + (counts.guard ?? 0)
    + (counts.witch ?? 0)
    + (counts.hunter ?? 0)
    + (counts.idiot ?? 0)
}

export function socialButton(config: typeof GAME_COPY[SocialGameSlug], primary = false) {
  return cn('inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border px-3 text-sm font-black transition disabled:cursor-not-allowed disabled:opacity-45', primary ? config.primary : config.button)
}

export function socialIconButton(config: typeof GAME_COPY[SocialGameSlug]) {
  return cn('inline-grid size-8 place-items-center rounded-lg border transition', config.button)
}

export function roleLabel(role: SocialRole, t: (key: string) => string) {
  return t(`social.roles.${role}`)
}
