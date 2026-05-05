"use client"

import Image from "next/image"
import { Send, User, Video as VideoIcon } from "lucide-react"
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

// W-08 candidate card — Soleil v2.
// Anatomy: ivoire background, rounded-2xl, hover translation. Avatar
// stays a 40×40 next/image when photo_url is set (the card test pins
// width/height/classes); otherwise falls back to a corail-soft
// initials disc. Org type uses Soleil pills (corail-soft / sapin-soft
// / sable-soft). Action row keeps "View profile" as ghost outline +
// "Send message" as corail pill.

function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return "?"
  if (parts.length === 1) return parts[0][0].toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function formatDateFr(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "long",
    year: "numeric",
  })
}

const ORG_PILL_CLASSES: Record<string, string> = {
  provider_personal: "bg-primary-soft text-primary-deep",
  agency: "bg-success-soft text-success",
  enterprise: "bg-amber-soft text-foreground",
}

function orgLabelKey(orgType: string): string {
  switch (orgType) {
    case "agency":
      return "jobDetail_w08_orgAgency"
    case "enterprise":
      return "jobDetail_w08_orgEnterprise"
    default:
      return "jobDetail_w08_orgFreelance"
  }
}

export function CandidateCard({ item, isSelected, onClick }: CandidateCardProps) {
  const t = useTranslations("opportunity")
  const tJob = useTranslations("job")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const { application, profile } = item

  const displayName = profile.name
  const initials = initialsFromName(displayName)
  const pillClass =
    ORG_PILL_CLASSES[profile.org_type] ?? "bg-border text-muted-foreground"

  function handleSendMessage(e: React.MouseEvent) {
    e.stopPropagation()
    if (isDesktop) {
      openChatWithOrg(application.applicant_id, displayName)
    } else {
      router.push(
        `/messages?to=${application.applicant_id}&name=${encodeURIComponent(displayName)}`,
      )
    }
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") onClick?.()
      }}
      className={cn(
        "rounded-2xl border bg-card p-5 transition-all duration-200 ease-out",
        "hover:-translate-y-0.5 hover:border-border-strong",
        isSelected
          ? "border-primary/40 ring-2 ring-primary/15"
          : "border-border",
        onClick && "cursor-pointer",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="flex items-center gap-3.5">
        {/* Avatar — next/image when photo_url is set (40×40, rounded). */}
        {profile.photo_url ? (
          <Image
            src={profile.photo_url}
            alt={displayName}
            width={40}
            height={40}
            className="h-10 w-10 shrink-0 rounded-full object-cover"
          />
        ) : (
          <div
            className={cn(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-full",
              "bg-primary-soft text-[13px] font-semibold text-primary-deep",
            )}
          >
            {initials}
          </div>
        )}

        {/* Identity column */}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <p className="truncate text-[14.5px] font-semibold text-foreground">
              {displayName}
            </p>
            <span
              className={cn(
                "inline-flex items-center rounded-full px-2 py-0.5",
                "text-[10.5px] font-bold uppercase tracking-[0.05em]",
                pillClass,
              )}
            >
              {tJob(orgLabelKey(profile.org_type))}
            </span>
            {application.video_url && (
              <span
                className={cn(
                  "inline-flex items-center gap-1 rounded-full px-2 py-0.5",
                  "bg-background text-[10.5px] font-medium text-muted-foreground",
                )}
              >
                <VideoIcon className="h-2.5 w-2.5" strokeWidth={1.8} />
                {tJob("jobDetail_w08_videoBadge")}
              </span>
            )}
          </div>
          {profile.title && (
            <p className="mt-1 truncate text-[12.5px] text-muted-foreground">
              {profile.title}
            </p>
          )}
        </div>

        {/* Action buttons */}
        <div className="hidden shrink-0 items-center gap-1.5 sm:flex">
          <Link
            href={`/freelancers/${application.applicant_id}`}
            onClick={(e) => e.stopPropagation()}
            className={cn(
              "inline-flex items-center gap-1 rounded-full px-3 py-1.5",
              "border border-border-strong bg-card text-[11.5px] font-medium text-foreground",
              "transition-colors duration-150 hover:bg-primary-soft hover:text-primary-deep",
            )}
          >
            <User className="h-3 w-3" strokeWidth={1.8} />
            {t("viewProfile")}
          </Link>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleSendMessage}
            className={cn(
              "inline-flex items-center gap-1 rounded-full px-3 py-1.5",
              "bg-primary text-[11.5px] font-semibold text-white",
              "transition-colors duration-150 hover:bg-primary-deep",
            )}
          >
            <Send className="h-3 w-3" strokeWidth={1.8} />
            {t("sendMessage")}
          </Button>
        </div>
      </div>

      {/* Application excerpt + date */}
      {application.message && (
        <p className="ml-[54px] mt-3 line-clamp-2 text-[13.5px] leading-relaxed text-muted-foreground">
          {application.message}
        </p>
      )}
      <p className="ml-[54px] mt-2 font-mono text-[10.5px] uppercase tracking-[0.06em] text-subtle-foreground">
        {tJob("jobDetail_w08_appliedRelative", {
          when: formatDateFr(application.created_at),
        })}
      </p>
    </div>
  )
}
