"use client"

import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useUser } from "@/shared/hooks/use-user"
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
  // `useUser()` shares the singleton ["session"] query (see
  // shared/hooks/use-user.ts). Mounting it here triggers no extra
  // /auth/me request — the public profile shell already mounts a
  // session consumer (PublicLayout, PostHogProvider…), so this is
  // a cache read.
  const { data: user } = useUser()

  function handleClick() {
    // Unauthenticated users can't open a conversation: send them to
    // /login with a `next` param so they bounce back here after
    // signing in. We do NOT call `trackLead` for unauth clicks —
    // the GA4 lead funnel must only count actionable intent (a
    // signed-in user opening a chat), otherwise conversion ratios
    // are skewed by the bounce-to-login traffic.
    if (!user) {
      const currentPath = typeof window !== "undefined" ? window.location.pathname : "/"
      router.push(`/login?next=${encodeURIComponent(currentPath)}`)
      return
    }

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
