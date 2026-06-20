import { SocialDeductionRoomGate } from './SocialDeductionRoomGate'

export function WerewolfRoomGate({ roomId }: { roomId?: string }) {
  return <SocialDeductionRoomGate game="werewolf" roomId={roomId} />
}
