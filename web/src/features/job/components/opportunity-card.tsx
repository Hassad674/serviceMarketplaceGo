"use client"

import { useState } from "react"
import { Briefcase, Calendar, Users, MoreVertical } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { ReportDialog } from "@/shared/components/reporting/report-dialog"
import { Portrait } from "@/shared/components/ui/portrait"
import type { JobResponse } from "../types"
import { Button } from "@/shared/components/ui/button"

interface OpportunityCardProps {
  job: JobResponse
  hasApplied?: boolean
}

// W-12 · Opportunity card — Soleil v2 anatomy:
//   - ivoire/white surface, rounded-2xl, border + shadow-card
//   - Portrait (deterministic palette) + serif company headline area
//   - Fraunces title (line-clamp-2)
//   - tabac excerpt + skill pills (corail-soft)
//   - Footer row: Geist Mono budget pill + applicant type + posted date
//
// Reference: design/assets/sources/phase1/soleil-lotC.jsx lines 197-237.
// The Portrait id is derived deterministically from the job id so the
// avatar is stable across renders.
function portraitId(seed: string): number {
  let hash = 0
  for (let i = 0; i < seed.length; i += 1) hash = (hash * 31 + seed.charCodeAt(i)) | 0
  return Math.abs(hash) % 6
}

export function OpportunityCard({ job, hasApplied = false }: OpportunityCardProps) {
  const t = useTranslations("opportunity")
  const tReport = useTranslations("reporting")
  const [showReport, setShowReport] = useState(false)

  const budgetLabel =
    job.budget_type === "long_term"
      ? `${job.min_budget.toLocaleString("fr-FR")}€ — ${job.max_budget.toLocaleString("fr-FR")}€ / ${job.payment_frequency === "weekly" ? t("weekly") : t("monthly")}`
      : `${job.min_budget.toLocaleString("fr-FR")}€ — ${job.max_budget.toLocaleString("fr-FR")}€`

  const applicantLabel =
    job.applicant_type === "all"
      ? t("allTypes")
      : job.applicant_type === "freelancers"
        ? t("freelancersOnly")
        : t("agenciesOnly")

  const kindLabel = job.budget_type === "long_term" ? t("longTerm") : t("oneShot")
  const seedId = portraitId(job.id)

  return (
    <>
      <Link href={`/opportunities/${job.id}`} className="block">
        <article
          className={cn(
            "group relative h-full overflow-hidden rounded-2xl border border-border bg-card p-6",
            "transition-all duration-200 ease-out",
            "hover:-translate-y-0.5 hover:border-border-strong",
          )}
          style={{ boxShadow: "var(--shadow-card)" }}
        >
          {/* Header row: portrait + kind/applicant + report menu */}
          <div className="mb-4 flex items-start justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <Portrait id={seedId} size={36} rounded="md" />
              <div className="min-w-0">
                <p className="truncate text-[13px] font-semibold text-foreground">
                  {applicantLabel}
                </p>
                <p className="font-mono text-[11px] text-muted-foreground">
                  {kindLabel}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1">
              {hasApplied && (
                <span className="rounded-full bg-success-soft px-2.5 py-0.5 text-[11px] font-semibold uppercase tracking-[0.04em] text-success">
                  {t("alreadyApplied")}
                </span>
              )}
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  setShowReport(true)
                }}
                aria-label={tReport("report")}
                className="flex h-7 w-7 items-center justify-center rounded-full text-subtle-foreground hover:bg-background hover:text-muted-foreground"
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Title — Fraunces */}
          <h3 className="mb-2 line-clamp-2 font-serif text-[22px] font-medium leading-[1.2] tracking-[-0.015em] text-foreground">
            {job.title}
          </h3>

          {/* Excerpt — tabac */}
          {job.description && (
            <p className="mb-4 line-clamp-2 break-words text-[14px] leading-relaxed text-muted-foreground overflow-wrap-anywhere">
              {job.description}
            </p>
          )}

          {/* Skill pills (Soleil ivoire neutral) */}
          {job.skills.length > 0 && (
            <div className="mb-4 flex flex-wrap gap-1.5">
              {job.skills.slice(0, 4).map((skill) => (
                <span
                  key={skill}
                  className="rounded-full bg-background px-2.5 py-1 text-[11.5px] font-medium text-muted-foreground"
                >
                  {skill}
                </span>
              ))}
              {job.skills.length > 4 && (
                <span className="self-center text-[11.5px] text-subtle-foreground">
                  +{job.skills.length - 4}
                </span>
              )}
            </div>
          )}

          {/* Footer: budget pill (mono) + applicant + date */}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border pt-4 text-[12.5px] text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <Briefcase className="h-3.5 w-3.5" strokeWidth={1.6} />
              <strong className="font-serif text-[13.5px] font-medium text-foreground">
                {budgetLabel}
              </strong>
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Users className="h-3.5 w-3.5" strokeWidth={1.6} />
              {applicantLabel}
            </span>
            <span className="ml-auto inline-flex items-center gap-1.5 font-mono text-[11.5px]">
              <Calendar className="h-3.5 w-3.5" strokeWidth={1.6} />
              {new Date(job.created_at).toLocaleDateString("fr-FR", {
                day: "numeric",
                month: "short",
              })}
            </span>
          </div>
        </article>
      </Link>
      <ReportDialog
        open={showReport}
        onClose={() => setShowReport(false)}
        targetType="job"
        targetId={job.id}
      />
    </>
  )
}
