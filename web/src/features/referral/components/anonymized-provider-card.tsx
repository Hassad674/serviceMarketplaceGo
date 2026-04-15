import { Award, Globe2, MapPin, Sparkles, Star, Wallet } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import type { ProviderSnapshot } from "../types"

interface AnonymizedProviderCardProps {
  snapshot: ProviderSnapshot
  className?: string
}

// AnonymizedProviderCard renders the provider's safe-to-reveal attributes
// for the client viewing a pre-active referral. Identity, name, photo and
// exact city are intentionally absent — the apporteur introduced them but
// the client does not get to skip the social step (Modèle A).
//
// Empty fields are simply not rendered so the card adapts gracefully when
// the apporteur revealed only a subset via the toggles.
export function AnonymizedProviderCard({
  snapshot,
  className,
}: AnonymizedProviderCardProps) {
  const hasAnyField =
    (snapshot.expertise_domains?.length ?? 0) > 0 ||
    snapshot.years_experience !== null && snapshot.years_experience !== undefined ||
    snapshot.average_rating !== null && snapshot.average_rating !== undefined ||
    snapshot.region ||
    (snapshot.languages?.length ?? 0) > 0 ||
    snapshot.availability_state ||
    snapshot.pricing_min_cents !== null && snapshot.pricing_min_cents !== undefined

  return (
    <article
      className={cn(
        "rounded-2xl border border-slate-200 bg-white p-6 shadow-sm",
        className,
      )}
    >
      <header className="mb-4 flex items-center gap-3">
        <div className="grid h-12 w-12 place-items-center rounded-full bg-rose-50 text-rose-500">
          <Sparkles className="h-6 w-6" aria-hidden="true" />
        </div>
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Prestataire recommandé
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
          {snapshot.expertise_domains && snapshot.expertise_domains.length > 0 && (
            <Row icon={<Award className="h-4 w-4" aria-hidden="true" />} label="Expertise">
              {snapshot.expertise_domains.join(", ")}
            </Row>
          )}
          {snapshot.years_experience !== null &&
            snapshot.years_experience !== undefined && (
              <Row icon={<Sparkles className="h-4 w-4" aria-hidden="true" />} label="Expérience">
                {snapshot.years_experience} ans
              </Row>
            )}
          {snapshot.average_rating !== null && snapshot.average_rating !== undefined && (
            <Row icon={<Star className="h-4 w-4" aria-hidden="true" />} label="Notation">
              {snapshot.average_rating.toFixed(1)} / 5
              {snapshot.review_count
                ? ` (${snapshot.review_count} avis)`
                : null}
            </Row>
          )}
          {(snapshot.pricing_min_cents !== null &&
            snapshot.pricing_min_cents !== undefined) && (
            <Row icon={<Wallet className="h-4 w-4" aria-hidden="true" />} label="Tarif">
              {formatPriceRange(
                snapshot.pricing_min_cents,
                snapshot.pricing_max_cents,
                snapshot.pricing_currency,
                snapshot.pricing_type,
              )}
            </Row>
          )}
          {snapshot.region && (
            <Row icon={<MapPin className="h-4 w-4" aria-hidden="true" />} label="Région">
              {snapshot.region}
            </Row>
          )}
          {snapshot.languages && snapshot.languages.length > 0 && (
            <Row icon={<Globe2 className="h-4 w-4" aria-hidden="true" />} label="Langues">
              {snapshot.languages.join(", ").toUpperCase()}
            </Row>
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
      <div className="mt-0.5 text-rose-500">{icon}</div>
      <div className="flex-1">
        <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
        <dd className="text-sm text-slate-900">{children}</dd>
      </div>
    </div>
  )
}

// formatPriceRange turns the min/max cents into a human label depending on
// the pricing type (daily / hourly / project_*). Falls back to a simple
// "min – max currency" rendering when the type is unknown.
function formatPriceRange(
  minCents: number | null | undefined,
  maxCents: number | null | undefined,
  currency: string | undefined,
  pricingType: string | undefined,
): string {
  if (minCents === null || minCents === undefined) return ""
  const cur = currency?.toUpperCase() ?? "EUR"
  const min = (minCents / 100).toLocaleString("fr-FR")
  const max =
    maxCents !== null && maxCents !== undefined
      ? (maxCents / 100).toLocaleString("fr-FR")
      : null
  const suffix = pricingType === "daily" ? " /j" : pricingType === "hourly" ? " /h" : ""
  if (max && max !== min) {
    return `${min} – ${max} ${cur}${suffix}`
  }
  return `${min} ${cur}${suffix}`
}
