import type { ReactNode } from 'react'
import type { SocialAlignment, SocialGameSlug, SocialRole, SocialRoom } from './online'
import { ArrowLeft } from 'lucide-react'
import { Link } from 'react-router'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { alignmentClass, roleLabel } from './socialStyle'
import { GAME_COPY, ROLE_ALIGNMENT, ROLES_BY_GAME } from './socialTheme'

export function RoleList({ game }: { game: SocialGameSlug }) {
  return (
    <div className="flex flex-wrap gap-2">
      {ROLES_BY_GAME[game].map(role => <RoleBadge key={role} role={role} />)}
    </div>
  )
}

export function RoleBadge({ dead = false, role, size = 'normal' }: { dead?: boolean, role: SocialRole, size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <SocialBadge className={dead ? deadBadgeClass() : alignmentClass(ROLE_ALIGNMENT[role])} size={size}>
      {roleLabel(role, t)}
    </SocialBadge>
  )
}

export function AlignmentBadge({ alignment, size = 'normal' }: { alignment: SocialAlignment, size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <SocialBadge className={alignmentClass(alignment)} size={size}>
      {t(`social.alignments.${alignment}`)}
    </SocialBadge>
  )
}

export function StatusBadge({ state, size = 'normal' }: { state: 'alive' | 'out', size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <SocialBadge className={state === 'alive' ? 'bg-emerald-200 text-emerald-950 ring-1 ring-emerald-300/45' : deadBadgeClass()} size={size}>
      {state === 'alive' ? t('social.alive') : t('social.out')}
    </SocialBadge>
  )
}

export function HiddenRoleBadge({ size = 'normal' }: { size?: 'normal' | 'large' }) {
  const { t } = useI18n()
  return (
    <SocialBadge className="bg-white/12 text-[#fff8e8]/78 ring-1 ring-white/12" size={size}>
      {t('social.hiddenRole')}
    </SocialBadge>
  )
}

export function SocialBadge({ children, className, size = 'normal' }: { children: ReactNode, className?: string, size?: 'normal' | 'large' }) {
  return (
    <span className={cn('inline-flex shrink-0 items-center rounded-full font-black leading-none', size === 'large' ? 'min-h-8 px-3 text-sm' : 'min-h-6 px-2 text-xs', className)}>
      {children}
    </span>
  )
}

function deadBadgeClass() {
  return 'bg-[#242933] text-[#cbd5e1] ring-1 ring-[#ff9aa8]/42 shadow-[inset_0_-2px_0_rgba(255,154,168,0.28)]'
}

export function SocialShell({
  children,
  config,
  fixedViewport = false,
  game,
  phase,
}: {
  children: ReactNode
  config: typeof GAME_COPY[SocialGameSlug]
  fixedViewport?: boolean
  game: SocialGameSlug
  phase?: SocialRoom['phase']
}) {
  const { t } = useI18n()
  const shellTheme = socialShellTheme(config, game, phase)
  return (
    <main className={cn('relative text-[#fff8e8] transition-colors duration-700 ease-in-out', fixedViewport ? 'h-svh overflow-hidden' : 'min-h-svh overflow-y-auto', shellTheme.bg)}>
      <div className={cn('pointer-events-none absolute inset-0 transition-[background,opacity] duration-700 ease-in-out', shellTheme.overlay)} />
      <div className={cn('relative mx-auto grid w-[min(1240px,calc(100vw-24px))] grid-rows-[auto_minmax(0,1fr)] gap-3 py-3', fixedViewport ? 'h-svh min-h-0' : 'min-h-svh')}>
        <header className="flex items-end justify-between gap-3">
          <div>
            <p className="mb-1 text-xs font-black tracking-normal text-[#fff8e8]/72">{t(GAME_COPY[game].subtitleKey)}</p>
            <h1 className="text-[clamp(30px,8vw,84px)] font-black leading-[0.82] tracking-normal [text-shadow:0_8px_0_rgba(0,0,0,0.34)]">
              {t(GAME_COPY[game].titleKey)}
            </h1>
          </div>
          <Link className={cn('inline-flex min-h-10 shrink-0 items-center justify-center rounded-full border px-3 text-sm font-bold transition sm:px-4', config.button)} to="/">
            <ArrowLeft className="mr-2 inline size-4" />
            <span className="hidden sm:inline">{t('common.backToLobby')}</span>
            <span className="sm:hidden">{t('catalog.back')}</span>
          </Link>
        </header>
        {children}
      </div>
    </main>
  )
}

function socialShellTheme(config: typeof GAME_COPY[SocialGameSlug], game: SocialGameSlug, phase?: SocialRoom['phase']) {
  if (game !== 'werewolf') {
    return {
      bg: config.bg,
      overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(255,209,102,0.12),transparent_28%),radial-gradient(circle_at_88%_12%,rgba(115,171,191,0.16),transparent_26%)]',
    }
  }

  if (phase === 'night') {
    return {
      bg: 'bg-[#050912]',
      overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(96,165,250,0.22),transparent_30%),radial-gradient(circle_at_88%_12%,rgba(45,212,191,0.10),transparent_24%),linear-gradient(180deg,rgba(0,0,0,0.10),rgba(0,0,0,0.34))]',
    }
  }

  if (phase === 'day' || phase === 'vote' || phase === 'hunter') {
    return {
      bg: 'bg-[#31465a]',
      overlay: 'bg-[radial-gradient(circle_at_18%_0%,rgba(255,209,102,0.34),transparent_30%),radial-gradient(circle_at_84%_10%,rgba(125,211,252,0.24),transparent_28%),linear-gradient(180deg,rgba(255,248,232,0.06),rgba(8,16,24,0.18))]',
    }
  }

  return {
    bg: config.bg,
    overlay: 'bg-[radial-gradient(circle_at_20%_0%,rgba(255,209,102,0.14),transparent_28%),radial-gradient(circle_at_88%_12%,rgba(115,171,191,0.18),transparent_26%)]',
  }
}

export function Panel({ children, config }: { children: ReactNode, config: typeof GAME_COPY[SocialGameSlug] }) {
  return <aside className={cn('grid content-start gap-3 rounded-lg border p-4', config.panel)}>{children}</aside>
}

export function StatusPill({ children, className }: { children: ReactNode, className?: string }) {
  return (
    <span className={cn('inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-white/16 bg-white/10 px-3 text-xs font-black text-[#fff8e8]/86 sm:text-sm', className)}>
      {children}
    </span>
  )
}
