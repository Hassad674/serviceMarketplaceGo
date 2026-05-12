import { Briefcase, Building2, CalendarClock, Coins, MapPin } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"
import type { ClientSnapshot } from "../types"

interface AnonymizedClientCardProps {
  snapshot: ClientSnapshot
  className?: string
  /**
   * When `revealed` is true the card stops masking the client. For the
   * apporteur owner the card becomes purely informational — the only
   * content is the display name. There is no profile link, no eyebrow
   * label, no masking explainer.
   * Defaults to false to preserve the masked behaviour for the
   * provider viewer.
   */
  revealed?: boolean
  /**
   * Human-readable client name resolved server-side (organization name
   * when the user owns an enterprise / agency, "First Last" otherwise).
   * Required when `revealed` is true; rendered with the Soleil v2
   * Fraunces display face for the minimalist card.
   */
  displayName?: string
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
//
// When the apporteur (owner) views the card, the masked snapshot is
// replaced by a minimalist card showing just the display name — they
// already know who they introduced.
export function AnonymizedClientCard({
  snapshot,
  className,
  revealed = false,
  displayName,
}: AnonymizedClientCardProps) {
  const t = useTranslations("referralIdentity")
  if (revealed) {
    return (
      <RevealedClientCard
        title={t("clientTitle")}
        displayName={displayName}
        className={className}
      />
    )
  }
  return <MaskedClientCard snapshot={snapshot} className={className} t={t} />
}

interface RevealedClientCardProps {
  title: string
  displayName?: string
  className?: string
}

// RevealedClientCard is the minimalist apporteur-only variant: just
// the display name in the Soleil v2 Fraunces face, role label above.
// No button, no badge, no explainer.
function RevealedClientCard({
  title,
  displayName,
  className,
}: RevealedClientCardProps) {
  return (
    <article
      data-testid="anonymized-client-revealed"
      className={cn(
        "rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]",
        className,
      )}
    >
      <div className="flex items-center gap-3">
        <div className="grid h-12 w-12 place-items-center rounded-full bg-blue-50 text-blue-500">
          <Building2 className="h-6 w-6" aria-hidden="true" />
        </div>
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">
            {title}
          </p>
          <p
            className="truncate font-serif text-2xl font-medium text-foreground"
            data-testid="revealed-identity-name"
          >
            {displayName || "—"}
          </p>
        </div>
      </div>
    </article>
  )
}

interface MaskedClientCardProps {
  snapshot: ClientSnapshot
  className?: string
  t: ReturnType<typeof useTranslations>
}

function MaskedClientCard({
  snapshot,
  className,
  t,
}: MaskedClientCardProps) {
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
        "rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]",
        className,
      )}
    >
      <header className="mb-4 flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid h-12 w-12 place-items-center rounded-full bg-blue-50 text-blue-500">
            <Building2 className="h-6 w-6" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-foreground">
              {t("clientTitle")}
            </h2>
            <p className="text-xs text-muted-foreground">
              {t("maskedSubtitle")}
            </p>
          </div>
        </div>
      </header>

      {!hasAnyField ? (
        <p className="text-sm text-muted-foreground">{t("maskedNoDetail")}</p>
      ) : (
        <dl className="space-y-3 text-sm">
          {snapshot.industry && (
            <Row icon={<Briefcase className="h-4 w-4" />} label={t("rowSector")}>
              {snapshot.industry}
            </Row>
          )}
          {snapshot.size_bucket && (
            <Row icon={<Building2 className="h-4 w-4" />} label={t("rowSize")}>
              {SIZE_LABELS[snapshot.size_bucket] ?? snapshot.size_bucket}
            </Row>
          )}
          {snapshot.region && (
            <Row icon={<MapPin className="h-4 w-4" />} label={t("rowRegion")}>
              {snapshot.region}
            </Row>
          )}
          {snapshot.budget_estimate_min_cents !== null &&
            snapshot.budget_estimate_min_cents !== undefined && (
              <Row icon={<Coins className="h-4 w-4" />} label={t("rowBudget")}>
                {formatBudget(
                  snapshot.budget_estimate_min_cents,
                  snapshot.budget_estimate_max_cents,
                  snapshot.budget_currency,
                )}
              </Row>
            )}
          {snapshot.timeline && (
            <Row icon={<CalendarClock className="h-4 w-4" />} label={t("rowTiming")}>
              {snapshot.timeline}
            </Row>
          )}
          {snapshot.need_summary && (
            <div className="rounded-lg bg-muted p-3 text-sm text-foreground">
              <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t("rowNeed")}
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
        <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
        <dd className="text-sm text-foreground">{children}</dd>
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
