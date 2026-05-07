import { API_BASE_URL, apiClient } from "@/shared/lib/api-client"

export type InitiateCallResponse = {
  call_id: string
  room_name: string
  token: string
}

export type AcceptCallResponse = {
  token: string
  room_name: string
}

/**
 * Reconciliation read for the caller's currently active call. Mirrors
 * backend `MyActiveCallResponse`. `null` is the dominant case (no
 * call) and is NOT an error — the front-end uses this on mount to
 * detect orphan Redis state from a brutal browser close, network
 * loss, or hangup race condition.
 */
export type MyActiveCall = {
  call_id: string
  conversation_id: string
  room_name: string
  type: "audio" | "video"
  status: string
  started_at?: string | null
  other_participant_id: string
}

type MyActiveCallEnvelope = {
  data: MyActiveCall | null
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

export async function getMyActiveCall(signal?: AbortSignal): Promise<MyActiveCall | null> {
  const envelope = await apiClient<MyActiveCallEnvelope>("/api/v1/calls/me/active", {
    method: "GET",
    signal,
  })
  return envelope?.data ?? null
}

/**
 * Synchronously notify the backend that this tab is closing while a
 * call is active. Uses `navigator.sendBeacon` because it is the only
 * mechanism guaranteed to fire during `pagehide` (`fetch` is racey,
 * `XMLHttpRequest` synchronous is deprecated in unload handlers).
 *
 * The beacon is fire-and-forget — the function returns whether the
 * browser accepted it for delivery. `false` typically means the
 * payload exceeded the queueing budget (~64 KiB) or the API is
 * unavailable; in that case the backend will eventually reap the
 * Redis state via TTL (30 minutes).
 *
 * Body shape mirrors `POST /calls/{id}/end` so the existing handler
 * accepts it without a transport-specific branch.
 */
export function endCallBeacon(callId: string, duration: number): boolean {
  if (typeof navigator === "undefined" || typeof navigator.sendBeacon !== "function") {
    return false
  }
  const url = `${API_BASE_URL}/api/v1/calls/${callId}/end`
  // text/plain keeps the request CORS-safelisted: no preflight is
  // issued, which is mandatory because the browser is already in the
  // pagehide phase and cannot await an OPTIONS round-trip. The JSON
  // body is still parsed by the existing handler — Go's
  // json.Decoder reads from the body regardless of Content-Type.
  const blob = new Blob([JSON.stringify({ duration })], { type: "text/plain" })
  return navigator.sendBeacon(url, blob)
}
