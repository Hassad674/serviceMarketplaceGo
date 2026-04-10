"use client"

import { Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import type { PublicProfileSummary, SearchType } from "../api/search-api"

const ROLE_BADGE_STYLES: Record<string, string> = {
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400",
  provider: "bg-rose-50 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400",
  referrer: "bg-amber-50 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400",
}

const ROLE_LABELS: Record<string, string> = {
  agency: "Agency",
  provider: "Freelancer",
  referrer: "Referrer",
}

function getProfileHref(profile: PublicProfileSummary, type: SearchType): string {
  switch (type) {
    case "agency":
      return `/agencies/${profile.user_id}`
    case "referrer":
      return `/referrers/${profile.user_id}`
    case "freelancer":
    default:
      return `/freelancers/${profile.user_id}`
  }
}

function getDisplayName(profile: PublicProfileSummary, type: SearchType): string {
  if (type === "agency") {
    return profile.display_name || `${profile.first_name} ${profile.last_name}`
  }
  return `${profile.first_name} ${profile.last_name}`.trim() || profile.display_name
}

function getInitials(profile: PublicProfileSummary): string {
  if (profile.first_name && profile.last_name) {
    return `${profile.first_name.charAt(0)}${profile.last_name.charAt(0)}`
  }
  if (profile.display_name) {
    return profile.display_name.charAt(0).toUpperCase()
  }
  return "?"
}

interface ProviderCardProps {
  profile: PublicProfileSummary
  type: SearchType
}

export function ProviderCard({ profile, type }: ProviderCardProps) {
  const t = useTranslations("search")
  const displayRole = type === "referrer" ? "referrer" : profile.role
  const badgeStyle = ROLE_BADGE_STYLES[displayRole] ?? ROLE_BADGE_STYLES.provider
  const badgeLabel = ROLE_LABELS[displayRole] ?? displayRole

  return (
    <Link
      href={getProfileHref(profile, type)}
      className={cn(
        "group block rounded-xl border border-gray-100 dark:border-gray-800",
        "bg-white dark:bg-gray-900 p-4 shadow-sm",
        "transition-all duration-200 hover:shadow-md hover:-translate-y-0.5",
      )}
    >
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className="shrink-0">
          {profile.photo_url ? (
            <img
              src={profile.photo_url}
              alt={getDisplayName(profile, type)}
              width={48}
              height={48}
              className="h-12 w-12 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
              {getInitials(profile)}
            </div>
          )}
        </div>

        {/* Info */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-white group-hover:text-rose-600 dark:group-hover:text-rose-400 transition-colors">
              {getDisplayName(profile, type)}
            </h3>
            <span
              className={cn(
                "inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                badgeStyle,
              )}
            >
              {badgeLabel}
            </span>
          </div>
          <p className="mt-0.5 truncate text-sm text-gray-500 dark:text-gray-400">
            {profile.title || t("noTitle")}
          </p>
          {profile.review_count > 0 && (
            <div className="mt-1.5 flex items-center gap-1 text-xs">
              <Star
                className="h-3 w-3 fill-amber-400 text-amber-400"
                strokeWidth={1.5}
                aria-hidden="true"
              />
              <span className="font-semibold text-gray-900 dark:text-white">
                {profile.average_rating.toFixed(1)}
              </span>
              <span className="text-gray-500 dark:text-gray-400">
                ({profile.review_count})
              </span>
            </div>
          )}
        </div>
      </div>
    </Link>
  )
}
