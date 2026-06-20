import { SocialDeductionRoomGate } from './SocialDeductionRoomGate'

export function UndercoverRoomGate({ roomId }: { roomId?: string }) {
  return <SocialDeductionRoomGate game="undercover" roomId={roomId} />
}
