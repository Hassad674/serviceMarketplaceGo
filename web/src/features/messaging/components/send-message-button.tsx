"use client"

import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { openChatWithOrg } from "@/shared/components/chat-widget/use-chat-widget"
import { Button } from "@/shared/components/ui/button"
import { trackLead } from "@/shared/lib/analytics-events"

/**
 * Persona of the public profile being contacted. Drives the GA4
 * `generate_lead` segmentation so the dashboard can compare lead
 * volume per profile category.
 */
export type SendMessagePersona = "freelance" | "agency" | "referrer"

interface SendMessageButtonProps {
  targetOrgId: string
  targetDisplayName?: string
  /** Persona of the contacted profile (default: "freelance"). */
  persona?: SendMessagePersona
}

export function SendMessageButton({ targetOrgId, targetDisplayName, persona = "freelance" }: SendMessageButtonProps) {
  const t = useTranslations("messaging")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")

  function handleClick() {
    trackLead({ profileId: targetOrgId, persona })
    const name = targetDisplayName || ""
    if (isDesktop) {
      openChatWithOrg(targetOrgId, name)
    } else {
      router.push(`/messages?to=${targetOrgId}&name=${encodeURIComponent(name)}`)
    }
  }

  return (
    <Button variant="ghost" size="auto"
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium",
        "bg-primary text-white shadow-sm",
        "transition-all duration-200 hover:bg-primary-deep hover:shadow-md active:scale-[0.98]",
      )}
    >
      <MessageSquare className="h-4 w-4" strokeWidth={1.5} />
      {t("startConversation")}
    </Button>
  )
}
