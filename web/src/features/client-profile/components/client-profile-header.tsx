"use client"

import Image from "next/image"
import { Briefcase, Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { formatCurrency } from "@/shared/lib/utils"

export interface ClientProfileHeaderStats {
  totalSpent: number
  reviewCount: number
  averageRating: number
  projectsCompleted: number
}

interface ClientProfileHeaderProps {
  companyName: string
  avatarUrl: string | null
  stats: ClientProfileHeaderStats
}

// ClientProfileHeader renders the read-only identity + metrics strip
// for both the private `/client-profile` page and the public
// `/clients/[id]` page. It is intentionally avatar-read-only — the
// avatar is shared with the provider profile and is edited there, so
// the editor lives on the existing provider-profile surface.
export function ClientProfileHeader(props: ClientProfileHeaderProps) {
  const { companyName, avatarUrl, stats } = props
  const t = useTranslations("clientProfile")

  const initials = getInitials(companyName)
  // Amount comes in as integer cents — formatCurrency expects whole
  // euros, so divide by 100. Keeping the cents/unit conversion here
  // (rather than in the hook) mirrors the pattern used by the
  // billing FeePreview component.
  const totalSpent = formatCurrency(stats.totalSpent / 100)

  return (
    <section
      className="bg-card border border-border rounded-2xl p-6 shadow-sm"
      aria-label={t("pageTitle")}
    >
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        <Avatar avatarUrl={avatarUrl} initials={initials} name={companyName} />
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-semibold text-foreground truncate">
            {companyName}
          </h1>
          <p className="mt-1 inline-flex items-center gap-2 rounded-md bg-purple-50 px-2 py-1 text-xs font-medium text-purple-700 dark:bg-purple-500/15 dark:text-purple-300">
            <Briefcase className="h-3.5 w-3.5" aria-hidden="true" />
            {t("roleLabel")}
          </p>
        </div>
      </div>

      <dl className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label={t("totalSpentLabel")} value={totalSpent} />
        <Stat
          label={t("projectsCompleted")}
          value={String(stats.projectsCompleted)}
        />
        <Stat
          label={t("reviewsReceived")}
          value={String(stats.reviewCount)}
        />
        <Stat
          label={t("averageRating")}
          value={
            stats.reviewCount === 0
              ? "—"
              : `${stats.averageRating.toFixed(1)} / 5`
          }
          icon={stats.reviewCount > 0 ? <Star className="h-4 w-4 text-amber-500" aria-hidden="true" /> : null}
        />
      </dl>
    </section>
  )
}

function getInitials(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return "?"
  const parts = trimmed.split(/\s+/).slice(0, 2)
  return parts.map((p) => p.charAt(0).toUpperCase()).join("")
}

interface AvatarProps {
  avatarUrl: string | null
  initials: string
  name: string
}

function Avatar({ avatarUrl, initials, name }: AvatarProps) {
  if (avatarUrl) {
    return (
      <Image
        src={avatarUrl}
        alt={name}
        width={80}
        height={80}
        className="h-20 w-20 shrink-0 rounded-full object-cover"
      />
    )
  }
  return (
    <div
      aria-hidden="true"
      className="flex h-20 w-20 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 text-xl font-semibold text-white"
    >
      {initials}
    </div>
  )
}

interface StatProps {
  label: string
  value: string
  icon?: React.ReactNode
}

function Stat({ label, value, icon }: StatProps) {
  return (
    <div className="rounded-xl bg-muted/40 px-4 py-3">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd className="mt-1 flex items-center gap-1.5 text-lg font-semibold text-foreground">
        {icon}
        <span className="font-mono tabular-nums">{value}</span>
      </dd>
    </div>
  )
}
