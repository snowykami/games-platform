import type { SocialGameSlug, SocialPlayer, SocialRole, SocialRoom } from './online'
import type { GAME_COPY } from './socialTheme'
import { useI18n } from '@/i18n/context'
import { cn } from '@/shared/lib/utils'
import { PlayerRefLabel } from './socialPlayers'
import { AlignmentBadge, HiddenRoleBadge, RoleBadge, StatusBadge } from './socialUi'

export function SelfIntel({
  className,
  config,
  game,
  room,
  you,
}: {
  className?: string
  config: typeof GAME_COPY[SocialGameSlug]
  game: SocialGameSlug
  room: SocialRoom
  you?: SocialPlayer
}) {
  const { t } = useI18n()
  const visiblePlayers = room.players.filter(player => player.visibleToYou && player.id !== you?.id && player.role)
  const percivalMarks = room.players.filter(player => room.avalon.percivalMarks?.includes(player.id))
  const seerChecks = Object.entries(room.werewolf.seerChecks ?? {})
  const yourWord = undercoverWord(room, you)
  const shouldCenterIntel = game === 'undercover'
  return (
    <section className={cn('rounded-lg border p-4', config.panel, shouldCenterIntel && 'text-center', className)}>
      <h2 className={cn('text-xl font-black', config.accent)}>{t('social.yourIntel')}</h2>
      <div className={cn('mt-3 flex flex-wrap gap-2', shouldCenterIntel && 'justify-center')}>
        {you?.role ? <RoleBadge dead={you.alive === false} role={you.role} size="large" /> : <HiddenRoleBadge size="large" />}
        {you?.alive === false && (
          <StatusBadge size="large" state="out" />
        )}
      </div>
      <p className="mt-3 text-sm leading-6 text-[#fff8e8]/72">
        {rolePlayHint(game, you)}
      </p>
      {game === 'undercover' && (
        <div className="mt-3 rounded-lg bg-[#fff8e8] p-3 text-[#211728]">
          <p className="text-xs font-black">{t('undercover.yourWord')}</p>
          <strong className="text-2xl">{yourWord || t('undercover.blankWord')}</strong>
          {room.phase === 'finished' && room.undercover.wordPair && (
            <p className="mt-2 text-xs font-black">
              {t('undercover.finalWords', {
                civilian: room.undercover.wordPair.civilianWord ?? '-',
                undercover: room.undercover.wordPair.undercoverWord ?? '-',
              })}
            </p>
          )}
        </div>
      )}
      {visiblePlayers.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {visiblePlayers.map(player => (
            <span key={player.id} className="inline-flex items-center gap-1.5 rounded-full bg-black/28 px-3 py-1 text-xs font-black">
              <PlayerRefLabel player={player} room={room} />
              <RoleBadge role={player.role!} />
            </span>
          ))}
        </div>
      )}
      {game === 'avalon' && percivalMarks.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {percivalMarks.map(player => (
            <span key={player.id} className="inline-flex items-center gap-1.5 rounded-full bg-black/28 px-3 py-1 text-xs font-black">
              <PlayerRefLabel player={player} room={room} />
              <span className="text-[#fff8e8]/72">梅林 / 莫甘娜</span>
            </span>
          ))}
        </div>
      )}
      {game === 'werewolf' && seerChecks.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {seerChecks.map(([playerId, alignment]) => {
            const checkedPlayer = room.players.find(player => player.id === playerId)
            return (
              <span key={playerId} className="inline-flex items-center gap-1.5 rounded-full bg-black/28 px-3 py-1 text-xs font-black">
                {checkedPlayer ? <PlayerRefLabel player={checkedPlayer} room={room} /> : t('common.player')}
                <AlignmentBadge alignment={alignment} />
              </span>
            )
          })}
        </div>
      )}
    </section>
  )
}

function undercoverWord(room: SocialRoom, you?: SocialPlayer) {
  if (room.undercover.yourWord !== undefined) {
    return room.undercover.yourWord
  }
  if (!you || !room.undercover.wordPair || room.phase !== 'finished') {
    return ''
  }
  if (you.role === 'undercover') {
    return room.undercover.wordPair.undercoverWord ?? ''
  }
  if (you.role === 'civilian') {
    return room.undercover.wordPair.civilianWord ?? ''
  }
  return ''
}

function rolePlayHint(game: SocialGameSlug, player?: SocialPlayer) {
  if (!player?.role) {
    if (game === 'undercover') {
      return '你只知道自己的词。描述时不要直接说出底词，同时观察谁的描述和大家不太一样。'
    }
    return '先观察局势，等身份揭晓后按你的阵营目标行动。'
  }

  if (game === 'werewolf') {
    return werewolfRoleHint(player.role)
  }
  if (game === 'avalon') {
    return avalonRoleHint(player.role)
  }
  return undercoverRoleHint(player.role)
}

function werewolfRoleHint(role: SocialRole) {
  switch (role) {
    case 'werewolf':
      return '你是狼人。夜里和狼队一起找关键好人下手，白天要伪装成好人、带偏投票。'
    case 'seer':
      return '你是预言家。每晚查验一名玩家阵营，白天用发言把信息传出去，同时别太早暴露自己。'
    case 'witch':
      return '你是女巫。你有一瓶解药和一瓶毒药，各只能用一次；救人、毒人都要看局势。'
    case 'guard':
      return '你是守卫。每晚保护一名玩家，尽量预判狼刀目标，避免连续死守同一个思路。'
    case 'hunter':
      return '你是猎人。正常白天找狼；如果死亡触发开枪机会，尽量带走你最怀疑的玩家。'
    case 'idiot':
      return '你是白痴。首次被放逐会翻牌免死，但之后不能投票；发言时可以更大胆地找狼。'
    default:
      return '你是村民。没有夜间技能，白天主要靠发言、投票和观察行为找出狼人。'
  }
}

function avalonRoleHint(role: SocialRole) {
  switch (role) {
    case 'merlin':
      return '你是梅林。你知道邪恶阵营的大致身份，要引导正义获胜，但不能让刺客看出你是梅林。'
    case 'percival':
      return '你是派西维尔。你会看到两名像梅林的人，其中一个是真梅林、一个是莫甘娜；保护真梅林，同时别被假线索骗走。'
    case 'assassin':
      return '你是刺客。和邪恶阵营一起破坏任务；如果正义完成三次任务，你还可以刺杀梅林翻盘。'
    case 'morgana':
      return '你是莫甘娜。你会混进派西维尔的视野里，尽量让他误以为你才是梅林。'
    case 'mordred':
      return '你是莫德雷德。你属于邪恶阵营，但梅林看不见你；利用这个盲区混进关键队伍。'
    case 'oberon':
      return '你是奥伯伦。你属于邪恶阵营，但其他邪恶队友看不见你，你也看不见他们；要独立判断局势。'
    case 'minion':
      return '你是爪牙。帮邪恶阵营混入任务队伍、制造怀疑，必要时掩护刺客判断梅林。'
    case 'lancelot':
      return '你是兰斯洛特。你的阵营可能随局势变化，要根据当前阵营目标行动，并留意任务破坏留下的线索。'
    case 'lady_of_lake':
      return '你是湖中仙女。通常在中后期获得查验阵营的能力，查验后线索会传递给被查验的人。'
    default:
      return '你是忠臣。你不知道谁好谁坏，要通过组队、投票和任务结果判断阵营。'
  }
}

function undercoverRoleHint(role: SocialRole) {
  switch (role) {
    case 'undercover':
      return '你是卧底。你的词和多数人不一样，描述时要贴近大家但别暴露差异。'
    case 'blank':
      return '你是白板。你没有词，要从别人描述里推断主题，再用模糊但可信的线索混进去。'
    default:
      return '你是平民。描述自己的词但不要直接说出来，同时观察谁的描述和大家不太一样。'
  }
}
