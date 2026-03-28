"use client"

import { createContext, useContext } from "react"

type StartCallFn = (
  conversationId: string,
  recipientId: string,
  recipientName?: string,
  callType?: "audio" | "video",
) => Promise<void>

interface CallContextValue {
  startCall: StartCallFn
}

export const CallContext = createContext<CallContextValue | null>(null)

export function useCallContext(): CallContextValue | null {
  return useContext(CallContext)
}
