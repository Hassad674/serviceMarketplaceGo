import { Briefcase, Building2, CalendarClock, Coins, MapPin } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import type { ClientSnapshot } from "../types"

interface AnonymizedClientCardProps {
  snapshot: ClientSnapshot
  className?: string
}

const SIZE_LABELS: Record<string, string> = {
  tpe: "TPE (< 10 salariés)",
  pme: "PME (10-250 salariés)",
  eti: "ETI (250-5000 salariés)",
  ge: "Grande entreprise (> 5000)",
}

// AnonymizedClientCard renders the safe-to-reveal client attributes for the
// provider's modal-as-page side. Company name, logo, and contact are
// intentionally absent — the apporteur surfaces sector / size / region /
// budget so the provider can decide whether the deal is worth their time.
export function AnonymizedClientCard({
  snapshot,
  className,
}: AnonymizedClientCardProps) {
  const hasAnyField =
    snapshot.industry ||
    snapshot.size_bucket ||
    snapshot.region ||
    snapshot.need_summary ||
    snapshot.timeline ||
    (snapshot.budget_estimate_min_cents !== null &&
      snapshot.budget_estimate_min_cents !== undefined)

  return (
    <article
      className={cn(
        "rounded-2xl border border-slate-200 bg-white p-6 shadow-sm",
        className,
      )}
    >
      <header className="mb-4 flex items-center gap-3">
        <div className="grid h-12 w-12 place-items-center rounded-full bg-blue-50 text-blue-500">
          <Building2 className="h-6 w-6" aria-hidden="true" />
        </div>
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Client proposé
          </h2>
          <p className="text-xs text-slate-500">
            Identité révélée à l&rsquo;acceptation
          </p>
        </div>
      </header>

      {!hasAnyField ? (
        <p className="text-sm text-slate-500">
          L&rsquo;apporteur a choisi de ne révéler aucun détail avant
          l&rsquo;acceptation.
        </p>
      ) : (
        <dl className="space-y-3 text-sm">
          {snapshot.industry && (
            <Row icon={<Briefcase className="h-4 w-4" />} label="Secteur">
              {snapshot.industry}
            </Row>
          )}
          {snapshot.size_bucket && (
            <Row icon={<Building2 className="h-4 w-4" />} label="Taille">
              {SIZE_LABELS[snapshot.size_bucket] ?? snapshot.size_bucket}
            </Row>
          )}
          {snapshot.region && (
            <Row icon={<MapPin className="h-4 w-4" />} label="Région">
              {snapshot.region}
            </Row>
          )}
          {snapshot.budget_estimate_min_cents !== null &&
            snapshot.budget_estimate_min_cents !== undefined && (
              <Row icon={<Coins className="h-4 w-4" />} label="Budget estimé">
                {formatBudget(
                  snapshot.budget_estimate_min_cents,
                  snapshot.budget_estimate_max_cents,
                  snapshot.budget_currency,
                )}
              </Row>
            )}
          {snapshot.timeline && (
            <Row icon={<CalendarClock className="h-4 w-4" />} label="Timing">
              {snapshot.timeline}
            </Row>
          )}
          {snapshot.need_summary && (
            <div className="rounded-lg bg-slate-50 p-3 text-sm text-slate-700">
              <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                Besoin
              </p>
              {snapshot.need_summary}
            </div>
          )}
        </dl>
      )}
    </article>
  )
}

interface RowProps {
  icon: React.ReactNode
  label: string
  children: React.ReactNode
}

function Row({ icon, label, children }: RowProps) {
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5 text-blue-500">{icon}</div>
      <div className="flex-1">
        <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
        <dd className="text-sm text-slate-900">{children}</dd>
      </div>
    </div>
  )
}

function formatBudget(
  minCents: number | null | undefined,
  maxCents: number | null | undefined,
  currency: string | undefined,
): string {
  if (minCents === null || minCents === undefined) return ""
  const cur = currency?.toUpperCase() ?? "EUR"
  const min = (minCents / 100).toLocaleString("fr-FR")
  const max =
    maxCents !== null && maxCents !== undefined
      ? (maxCents / 100).toLocaleString("fr-FR")
      : null
  if (max && max !== min) {
    return `${min} – ${max} ${cur}`
  }
  return `${min} ${cur}`
}
