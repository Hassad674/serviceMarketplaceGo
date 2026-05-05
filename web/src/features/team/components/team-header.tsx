"use client"

import { Users2 } from "lucide-react"
import { useTranslations } from "next-intl"
import type { CurrentOrganization } from "@/shared/hooks/use-user"

// Editorial Soleil header for /team. Eyebrow corail mono caps + Fraunces
// title with italic-corail accent + tabac subtitle. Ports the W-22
// hero anatomy from `design/assets/sources/phase1/soleil.jsx` (editorial
// hero pattern shared with Notifications W-24 / Account W-XX).
//
// Member + pending counters are rendered as Soleil pills (rounded-full,
// ivoire-soft bg, mono caption). The numeric counters use Geist Mono
// (font-mono in Tailwind).

type TeamHeaderProps = {
  organization: CurrentOrganization
  memberCount: number
  pendingInvitationCount: number
}

export function TeamHeader({
  organization,
  memberCount,
  pendingInvitationCount,
}: TeamHeaderProps) {
  const t = useTranslations("team")

  const typeLabel =
    organization.type === "agency" ? t("agencyType") : t("enterpriseType")

  return (
    <header className="px-1">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-[var(--primary)]">
        {t("w22.eyebrow")}
      </p>
      <h1 className="mt-2 font-serif text-[32px] font-medium leading-[1.05] tracking-[-0.025em] text-[var(--foreground)] sm:text-[36px]">
        {t("w22.titleLead")}{" "}
        <span className="italic text-[var(--primary)]">
          {t("w22.titleAccent")}
        </span>
        .
      </h1>
      <p className="mt-2 max-w-2xl font-serif text-[14px] italic text-[var(--muted-foreground)]">
        {t("w22.subtitle", { orgType: typeLabel })}
      </p>

      <div className="mt-4 flex flex-wrap gap-2">
        <StatPill
          icon={<Users2 className="h-3.5 w-3.5" strokeWidth={1.8} />}
          label={t("membersCount", { count: memberCount })}
        />
        {pendingInvitationCount > 0 && (
          <StatPill
            label={t("pendingInvitationsCount", { count: pendingInvitationCount })}
            highlight
          />
        )}
      </div>
    </header>
  )
}

type StatPillProps = {
  icon?: React.ReactNode
  label: string
  highlight?: boolean
}

function StatPill({ icon, label, highlight }: StatPillProps) {
  const base =
    "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-[12px] font-medium"
  const tone = highlight
    ? "bg-[var(--primary-soft)] text-[var(--primary-deep)]"
    : "bg-[var(--background)] border border-[var(--border)] text-[var(--muted-foreground)]"
  return (
    <span className={`${base} ${tone}`}>
      {icon}
      {label}
    </span>
  )
}
