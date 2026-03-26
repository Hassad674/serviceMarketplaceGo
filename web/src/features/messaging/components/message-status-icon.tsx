"use client"

import { Clock, Check, CheckCheck } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { MessageStatus } from "../types"

interface MessageStatusIconProps {
  status: MessageStatus
}

export function MessageStatusIcon({ status }: MessageStatusIconProps) {
  const t = useTranslations("messaging")

  switch (status) {
    case "sending":
      return (
        <Clock
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label={t("statusSending")}
        />
      )
    case "sent":
      return (
        <Check
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label={t("statusSent")}
        />
      )
    case "delivered":
      return (
        <CheckCheck
          className="h-3 w-3 text-rose-200"
          strokeWidth={1.5}
          aria-label={t("statusDelivered")}
        />
      )
    case "read":
      return (
        <CheckCheck
          className={cn("h-3 w-3 text-blue-300")}
          strokeWidth={1.5}
          aria-label={t("statusRead")}
        />
      )
    default:
      return null
  }
}
