"use client"

import Image from "next/image"
import { Send, User } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { openChatWithOrg } from "@/shared/components/chat-widget/use-chat-widget"
import type { ApplicationWithProfile } from "../types"
import { Button } from "@/shared/components/ui/button"

interface CandidateCardProps {
  item: ApplicationWithProfile
  isSelected?: boolean
  onClick?: () => void
}

const ORG_TYPE_COLORS: Record<string, string> = {
  provider_personal: "bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-400",
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
  enterprise: "bg-purple-50 text-purple-700 dark:bg-purple-500/10 dark:text-purple-400",
}

function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return "?"
  if (parts.length === 1) return parts[0][0].toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export function CandidateCard({ item, isSelected, onClick }: CandidateCardProps) {
  const t = useTranslations("opportunity")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const { application, profile } = item

  const displayName = profile.name
  const initials = initialsFromName(displayName)

  function handleSendMessage(e: React.MouseEvent) {
    e.stopPropagation()
    if (isDesktop) {
      openChatWithOrg(application.applicant_id, displayName)
    } else {
      router.push(`/messages?to=${application.applicant_id}&name=${encodeURIComponent(displayName)}`)
    }
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") onClick?.() }}
      className={cn(
        "rounded-2xl border bg-white p-4 shadow-sm transition-all duration-200 ease-out",
        "dark:bg-slate-800/80",
        "hover:shadow-md hover:-translate-y-0.5",
        isSelected
          ? "border-rose-200 ring-2 ring-rose-500/10 dark:border-rose-500/30 dark:ring-rose-500/10"
          : "border-slate-100 hover:border-slate-200 dark:border-slate-700 dark:hover:border-slate-600",
        onClick && "cursor-pointer",
      )}
    >
      {/* Top row: avatar + info + action buttons */}
      <div className="flex items-center gap-3">
        {/* 40×40 candidate avatar (MinIO/R2 declared in next.config.ts). */}
        {profile.photo_url ? (
          <Image
            src={profile.photo_url}
            alt={displayName}
            width={40}
            height={40}
            className="h-10 w-10 shrink-0 rounded-full object-cover"
          />
        ) : (
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-rose-100 text-sm font-semibold text-rose-700 dark:bg-rose-500/20 dark:text-rose-400">
            {initials}
          </div>
        )}

        {/* Name + org type */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm font-semibold text-slate-900 dark:text-white truncate">
              {displayName}
            </p>
            <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0", ORG_TYPE_COLORS[profile.org_type] ?? "bg-slate-100 text-slate-600")}>
              {profile.org_type}
            </span>
            {application.video_url && (
              <span className="flex items-center gap-0.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-500 dark:bg-slate-700 dark:text-slate-400 shrink-0">
                <svg className="h-2.5 w-2.5" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true">
                  <path d="M4.5 2.5v7l5-3.5-5-3.5z" />
                </svg>
                Video
              </span>
            )}
          </div>
          {profile.title && (
            <p className="text-xs text-slate-500 dark:text-slate-400">{profile.title}</p>
          )}
        </div>

        {/* Action buttons — top right */}
        <div className="shrink-0 flex items-center gap-1.5">
          <Link
            href={`/freelancers/${application.applicant_id}`}
            onClick={(e) => e.stopPropagation()}
            className={cn(
              "flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-all",
              "border border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700",
            )}
          >
            <User className="h-3 w-3" />
            {t("viewProfile")}
          </Link>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleSendMessage}
            className={cn(
              "flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-all",
              "bg-rose-50 text-rose-600 hover:bg-rose-100 dark:bg-rose-500/10 dark:text-rose-400 dark:hover:bg-rose-500/20",
            )}
          >
            <Send className="h-3 w-3" />
            {t("sendMessage")}
          </Button>
        </div>
      </div>

      {/* Message preview + date */}
      {application.message && (
        <p className="text-sm text-slate-600 dark:text-slate-300 line-clamp-2 mt-2.5 ml-[52px]">{application.message}</p>
      )}
      <p className="text-xs text-slate-400 mt-1.5 ml-[52px]">
        {new Date(application.created_at).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" })}
      </p>
    </div>
  )
}
