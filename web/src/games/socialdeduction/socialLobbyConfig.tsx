import type { SocialGameSlug, SocialRole, SocialRoom, WerewolfRoleCounts } from './online'
import type { GAME_COPY } from './socialTheme'
import type { useSocialRoom } from './useSocialRoom'
import { useState } from 'react'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { usePendingAction } from '../usePendingAction'
import { roleLabel, roleTotal, socialButton } from './socialStyle'
import { RoleBadge, SocialBadge } from './socialUi'

export function WerewolfRoleSetup({
  actions,
  config,
  isHost,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  room: SocialRoom
}) {
  const { t } = useI18n()
  const [draft, setDraft] = useState<WerewolfRoleCounts>(room.werewolf.roleConfig.counts)
  const [message, setMessage] = useState('')
  const pending = usePendingAction()
  const rolePresets = room.werewolf.rolePresets ?? []
  const total = roleTotal(draft)
  const canSave = isHost
    && total === room.players.length
    && draft.werewolf > 0
    && draft.werewolf < room.players.length
    && draft.seer <= 1
    && draft.guard <= 1
    && draft.witch <= 1
    && draft.hunter <= 1
    && draft.idiot <= 1

  async function selectPreset(presetId: string) {
    const preset = rolePresets.find(candidate => candidate.id === presetId)
    if (!preset) {
      return
    }
    setMessage(t('werewolf.updatingRoles'))
    await pending.run(`preset:${preset.id}`, () => actions.updateWerewolfRoles({
      counts: preset.counts,
      description: preset.description,
      mode: 'preset',
      name: preset.name,
      presetId: preset.id,
    }).then(() => setMessage(t('werewolf.rolesUpdated'))), { releaseOnSettle: false })
  }

  async function saveCustom() {
    if (!canSave) {
      setMessage(t('werewolf.roleCountMismatch', { count: room.players.length, total }))
      return
    }
    setMessage(t('werewolf.updatingRoles'))
    await pending.run('custom', () => actions.updateWerewolfRoles({
      counts: draft,
      description: t('werewolf.customRolesDescription'),
      mode: 'custom',
      name: t('werewolf.customRoles'),
    }).then(() => setMessage(t('werewolf.rolesUpdated'))), { releaseOnSettle: false })
  }

  return (
    <section className="grid gap-3 rounded-lg border border-white/14 bg-black/22 p-3">
      <div>
        <h3 className="text-sm font-black">{t('werewolf.roleSetup')}</h3>
        <p className="mt-1 text-xs font-bold leading-5 text-[#fff8e8]/62">
          {t('werewolf.roleTotal', { count: room.players.length, total: roleTotal(room.werewolf.roleConfig.counts) })}
        </p>
      </div>

      <label className="grid gap-1 text-xs font-black">
        {t('werewolf.rolePreset')}
        <select
          className="min-h-10 rounded-lg border border-white/20 bg-black/30 px-3 text-sm font-black text-[#fff8e8] outline-none"
          disabled={!isHost || rolePresets.length === 0 || pending.isPending(`preset:${room.werewolf.roleConfig.presetId ?? ''}`)}
          value={room.werewolf.roleConfig.mode === 'preset' ? room.werewolf.roleConfig.presetId ?? '' : 'custom'}
          onChange={event => event.target.value === 'custom' ? undefined : void selectPreset(event.target.value)}
        >
          {rolePresets.length === 0 && <option value="">{t('werewolf.needMorePlayers')}</option>}
          {rolePresets.map(preset => <option key={preset.id} className="text-[#171411]" value={preset.id}>{preset.name}</option>)}
          <option className="text-[#171411]" value="custom">{t('werewolf.customRoles')}</option>
        </select>
      </label>

      <div className="grid gap-2">
        <RoleCountLine count={draft.werewolf} disabled={!isHost} role="werewolf" onChange={count => setDraft({ ...draft, werewolf: count })} />
        <RoleCountLine count={draft.villager} disabled={!isHost} role="villager" onChange={count => setDraft({ ...draft, villager: count })} />
        <RoleCountLine count={draft.seer} disabled={!isHost} max={1} role="seer" onChange={count => setDraft({ ...draft, seer: count })} />
        <RoleCountLine count={draft.witch} disabled={!isHost} max={1} role="witch" onChange={count => setDraft({ ...draft, witch: count })} />
        <RoleCountLine count={draft.hunter} disabled={!isHost} max={1} role="hunter" onChange={count => setDraft({ ...draft, hunter: count })} />
        <RoleCountLine count={draft.idiot} disabled={!isHost} max={1} role="idiot" onChange={count => setDraft({ ...draft, idiot: count })} />
        <RoleCountLine count={draft.guard} disabled={!isHost} max={1} role="guard" onChange={count => setDraft({ ...draft, guard: count })} />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button className={socialButton(config, true)} disabled={!canSave || pending.isPending('custom')} type="button" onClick={() => void saveCustom()}>
          {pending.isPending('custom') ? t('common.syncing') : t('werewolf.applyCustomRoles')}
        </button>
        <SocialBadge className={total === room.players.length ? 'bg-emerald-200 text-emerald-950 ring-1 ring-emerald-300/45' : 'bg-[#4a1424] text-[#ffd6df] ring-1 ring-[#ff7a9a]/35'}>
          {total}
          {' / '}
          {room.players.length}
        </SocialBadge>
      </div>
      <p className="min-h-5 text-xs font-bold text-[#fff8e8]/65">{message || room.werewolf.roleConfig.description}</p>
    </section>
  )
}

export function UndercoverLobbyConfig({
  actions,
  config,
  isHost,
  room,
}: {
  actions: ReturnType<typeof useSocialRoom>['actions']
  config: typeof GAME_COPY[SocialGameSlug]
  isHost: boolean
  room: SocialRoom
}) {
  const { t } = useI18n()
  const pending = usePendingAction()
  const presets = room.undercover.presets ?? []
  const selectedPreset = presets.find(preset => preset.id === room.undercover.presetId)

  return (
    <section className="grid gap-2 rounded-lg border border-white/14 bg-black/24 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <strong className={config.accent}>{t('undercover.presetTitle')}</strong>
          <p className="text-xs font-bold text-[#fff8e8]/62">{selectedPreset?.description ?? t('undercover.presetHint')}</p>
        </div>
        <label className="flex items-center gap-2 text-xs font-black text-[#fff8e8]/78">
          <input
            checked={room.undercover.includeBlank}
            className="size-4 accent-[#f4c7ff]"
            disabled={!isHost || pending.isPending('blank')}
            type="checkbox"
            onChange={event => void pending.run('blank', () => actions.undercoverConfig(room.undercover.presetId, event.target.checked), { releaseOnSettle: false })}
          />
          {t('undercover.includeBlank')}
        </label>
      </div>
      <div className="grid gap-2 sm:grid-cols-4">
        {presets.map(preset => (
          <button
            key={preset.id}
            className={cn(socialButton(config), preset.id === room.undercover.presetId && 'ring-2 ring-[#f4c7ff]')}
            disabled={!isHost || pending.isPending(`preset:${preset.id}`)}
            type="button"
            onClick={() => void pending.run(`preset:${preset.id}`, () => actions.undercoverConfig(preset.id, room.undercover.includeBlank), { releaseOnSettle: false })}
          >
            {pending.isPending(`preset:${preset.id}`) ? t('common.syncing') : preset.name}
          </button>
        ))}
      </div>
    </section>
  )
}

function RoleCountLine({ count, disabled, max = 12, onChange, role }: { count: number, disabled: boolean, max?: number, onChange: (count: number) => void, role: SocialRole }) {
  const { t } = useI18n()

  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 rounded-lg bg-black/24 px-2 py-2">
      <RoleBadge role={role} />
      <div className="grid grid-cols-[28px_34px_28px] items-center gap-1">
        <button className="grid size-7 place-items-center rounded-md bg-white/10 text-sm font-black disabled:opacity-40" disabled={disabled || count <= 0} type="button" onClick={() => onChange(Math.max(0, count - 1))}>-</button>
        <span className="text-center text-sm font-black" title={roleLabel(role, t)}>{count}</span>
        <button className="grid size-7 place-items-center rounded-md bg-white/10 text-sm font-black disabled:opacity-40" disabled={disabled || count >= max} type="button" onClick={() => onChange(count + 1)}>+</button>
      </div>
    </div>
  )
}
