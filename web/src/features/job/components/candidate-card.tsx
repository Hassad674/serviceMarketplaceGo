"use client"

import { Send, User } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { openChatWithUser } from "@/shared/components/chat-widget/use-chat-widget"
import type { ApplicationWithProfile } from "../types"

interface CandidateCardProps {
  item: ApplicationWithProfile
  jobId: string
}

const ROLE_COLORS: Record<string, string> = {
  provider: "bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-400",
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
}

export function CandidateCard({ item, jobId }: CandidateCardProps) {
  const t = useTranslations("opportunity")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const { application, profile } = item

  const initials = (profile.first_name?.[0] ?? "") + (profile.last_name?.[0] ?? "")
  const displayName = profile.display_name || `${profile.first_name} ${profile.last_name}`

  function handleSendMessage() {
    if (isDesktop) {
      // Desktop: open chat widget with this user (lazy conversation)
      openChatWithUser(application.applicant_id, displayName)
    } else {
      // Mobile: navigate to messaging page with recipient param
      router.push(`/messages?to=${application.applicant_id}&name=${encodeURIComponent(displayName)}`)
    }
  }

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-rose-100 text-sm font-semibold text-rose-700 dark:bg-rose-500/20 dark:text-rose-400">
          {initials || "?"}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <p className="text-sm font-semibold text-slate-900 dark:text-white truncate">
              {displayName}
            </p>
            <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", ROLE_COLORS[profile.role] ?? "bg-slate-100 text-slate-600")}>
              {profile.role}
            </span>
          </div>
          {profile.title && (
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">{profile.title}</p>
          )}
          {application.message && (
            <p className="text-sm text-slate-600 dark:text-slate-300 line-clamp-3">{application.message}</p>
          )}
          <p className="text-xs text-slate-400 mt-2">{new Date(application.created_at).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" })}</p>
        </div>

        {/* Actions */}
        <div className="shrink-0 flex flex-col gap-2">
          <Link
            href={`/freelancers/${application.applicant_id}`}
            className={cn(
              "flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-all",
              "border border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700",
            )}
          >
            <User className="h-3.5 w-3.5" />
            {t("viewProfile")}
          </Link>
          <button
            type="button"
            onClick={handleSendMessage}
            className={cn(
              "flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-all",
              "bg-rose-50 text-rose-700 hover:bg-rose-100 dark:bg-rose-500/10 dark:text-rose-400 dark:hover:bg-rose-500/20",
            )}
          >
            <Send className="h-3.5 w-3.5" />
            {t("sendMessage")}
          </button>
        </div>
      </div>
    </div>
  )
}
