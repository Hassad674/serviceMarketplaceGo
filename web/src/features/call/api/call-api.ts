import { apiClient } from "@/shared/lib/api-client"

export type InitiateCallResponse = {
  call_id: string
  room_name: string
  token: string
}

export type AcceptCallResponse = {
  token: string
  room_name: string
}

export function initiateCall(conversationId: string, recipientId: string, type: "audio" | "video" = "audio"): Promise<InitiateCallResponse> {
  return apiClient<InitiateCallResponse>("/api/v1/calls/initiate", {
    method: "POST",
    body: { conversation_id: conversationId, recipient_id: recipientId, type },
  })
}

export function acceptCall(callId: string): Promise<AcceptCallResponse> {
  return apiClient<AcceptCallResponse>(`/api/v1/calls/${callId}/accept`, {
    method: "POST",
  })
}

export function declineCall(callId: string): Promise<void> {
  return apiClient<void>(`/api/v1/calls/${callId}/decline`, {
    method: "POST",
  })
}

export function endCall(callId: string, duration: number): Promise<void> {
  return apiClient<void>(`/api/v1/calls/${callId}/end`, {
    method: "POST",
    body: { duration },
  })
}
