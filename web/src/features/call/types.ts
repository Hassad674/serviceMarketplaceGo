export type CallState =
  | "idle"
  | "ringing_outgoing"
  | "ringing_incoming"
  | "active"
  | "ended"

export type CallType = "audio" | "video"

export type ActiveCall = {
  callId: string
  conversationId: string
  roomName: string
  token: string
  callType: CallType
  startedAt: number | null
}

export type IncomingCall = {
  callId: string
  conversationId: string
  initiatorId: string
  initiatorName: string
  callType: CallType
}

export type CallEventPayload = {
  event: string
  call_id: string
  conversation_id: string
  initiator_id: string
  recipient_id: string
  call_type: CallType
}
