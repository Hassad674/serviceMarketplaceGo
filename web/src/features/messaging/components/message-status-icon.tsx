"use client"

import { Clock, Check, CheckCheck } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { MessageStatus } from "../types"

interface MessageStatusIconProps {
  status: MessageStatus
}

export function MessageStatusIcon({ status }: MessageStatusIconProps) {
  switch (status) {
    case "sending":
      return (
        <Clock
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label="Sending"
        />
      )
    case "sent":
      return (
        <Check
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label="Sent"
        />
      )
    case "delivered":
      return (
        <CheckCheck
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label="Delivered"
        />
      )
    case "read":
      return (
        <CheckCheck
          className={cn("h-3 w-3 text-blue-300")}
          strokeWidth={1.5}
          aria-label="Read"
        />
      )
    default:
      return null
  }
}
