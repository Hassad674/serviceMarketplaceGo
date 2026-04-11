"use client"

import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { openChatWithOrg } from "@/shared/components/chat-widget/use-chat-widget"

interface SendMessageButtonProps {
  targetOrgId: string
  targetDisplayName?: string
}

export function SendMessageButton({ targetOrgId, targetDisplayName }: SendMessageButtonProps) {
  const t = useTranslations("messaging")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")

  function handleClick() {
    const name = targetDisplayName || ""
    if (isDesktop) {
      openChatWithOrg(targetOrgId, name)
    } else {
      router.push(`/messages?to=${targetOrgId}&name=${encodeURIComponent(name)}`)
    }
  }

  return (
    <button
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium",
        "bg-rose-500 text-white shadow-sm",
        "transition-all duration-200 hover:bg-rose-600 hover:shadow-md active:scale-[0.98]",
      )}
    >
      <MessageSquare className="h-4 w-4" strokeWidth={1.5} />
      {t("startConversation")}
    </button>
  )
}
