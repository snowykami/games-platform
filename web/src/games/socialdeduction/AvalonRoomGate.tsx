import { SocialDeductionRoomGate } from './SocialDeductionRoomGate'

export function AvalonRoomGate({ roomId }: { roomId?: string }) {
  return <SocialDeductionRoomGate game="avalon" roomId={roomId} />
}
