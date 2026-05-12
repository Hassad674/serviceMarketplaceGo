import { Award, Globe2, MapPin, Sparkles, Star, Wallet } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"
import type { ProviderSnapshot } from "../types"

interface AnonymizedProviderCardProps {
  snapshot: ProviderSnapshot
  className?: string
  /**
   * When `revealed` is true the card stops masking the provider.
   * For the apporteur owner the card becomes purely informational —
   * the only content is the display name. There is no profile link,
   * no eyebrow label, no masking explainer (the apporteur knows who
   * they introduced — the card is a confirmation, not a discovery
   * surface).
   * Defaults to false to preserve the masked behaviour for client +
   * provider viewers.
   */
  revealed?: boolean
  /**
   * Human-readable provider name resolved server-side (organization
   * name when the user owns an agency, "First Last" otherwise).
   * Required when `revealed` is true; rendered with the Soleil v2
   * Fraunces display face for the minimalist card.
   */
  displayName?: string
}

// AnonymizedProviderCard renders the provider's safe-to-reveal attributes
// for the client viewing a pre-active referral. Identity, name, photo and
// exact city are intentionally absent — the apporteur introduced them but
// the client does not get to skip the social step (Modèle A).
//
// When the apporteur (owner) views the card, the masked snapshot is
// replaced by a minimalist card showing just the display name — they
// already know who they introduced; the card is a confirmation, not a
// discovery surface.
export function AnonymizedProviderCard({
  snapshot,
  className,
  revealed = false,
  displayName,
}: AnonymizedProviderCardProps) {
  const t = useTranslations("referralIdentity")
  if (revealed) {
    return (
      <RevealedProviderCard
        title={t("providerTitle")}
        displayName={displayName}
        className={className}
      />
    )
  }
  return <MaskedProviderCard snapshot={snapshot} className={className} t={t} />
}

interface RevealedProviderCardProps {
  title: string
  displayName?: string
  className?: string
}

// RevealedProviderCard is the minimalist apporteur-only variant: just
// the display name in the Soleil v2 Fraunces face, role label
// ("Prestataire recommandé") above. No button, no badge, no explainer
// — pure informational confirmation.
function RevealedProviderCard({
  title,
  displayName,
  className,
}: RevealedProviderCardProps) {
  return (
    <article
      data-testid="anonymized-provider-revealed"
      className={cn(
        "rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]",
        className,
      )}
    >
      <div className="flex items-center gap-3">
        <div className="grid h-12 w-12 place-items-center rounded-full bg-primary-soft text-primary">
          <Sparkles className="h-6 w-6" aria-hidden="true" />
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

interface MaskedProviderCardProps {
  snapshot: ProviderSnapshot
  className?: string
  t: ReturnType<typeof useTranslations>
}

// MaskedProviderCard is the original masked-fields layout — preserved
// untouched for the provider + client viewers (Modèle A: identity is
// revealed at conversation activation, not on the dashboard).
function MaskedProviderCard({
  snapshot,
  className,
  t,
}: MaskedProviderCardProps) {
  const hasAnyField =
    (snapshot.expertise_domains?.length ?? 0) > 0 ||
    (snapshot.years_experience !== null && snapshot.years_experience !== undefined) ||
    (snapshot.average_rating !== null && snapshot.average_rating !== undefined) ||
    snapshot.region ||
    (snapshot.languages?.length ?? 0) > 0 ||
    snapshot.availability_state ||
    (snapshot.pricing_min_cents !== null && snapshot.pricing_min_cents !== undefined)

  return (
    <article
      className={cn(
        "rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]",
        className,
      )}
    >
      <header className="mb-4 flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid h-12 w-12 place-items-center rounded-full bg-primary-soft text-primary">
            <Sparkles className="h-6 w-6" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-foreground">
              {t("providerTitle")}
            </h2>
            <p className="text-xs text-muted-foreground">{t("maskedSubtitle")}</p>
          </div>
        </div>
      </header>

      {!hasAnyField ? (
        <p className="text-sm text-muted-foreground">{t("maskedNoDetail")}</p>
      ) : (
        <dl className="space-y-3 text-sm">
          {snapshot.expertise_domains && snapshot.expertise_domains.length > 0 && (
            <Row
              icon={<Award className="h-4 w-4" aria-hidden="true" />}
              label={t("rowExpertise")}
            >
              {snapshot.expertise_domains.join(", ")}
            </Row>
          )}
          {snapshot.years_experience !== null &&
            snapshot.years_experience !== undefined && (
              <Row
                icon={<Sparkles className="h-4 w-4" aria-hidden="true" />}
                label={t("rowExperience")}
              >
                {snapshot.years_experience} {t("yearsSuffix")}
              </Row>
            )}
          {snapshot.average_rating !== null &&
            snapshot.average_rating !== undefined && (
              <Row
                icon={<Star className="h-4 w-4" aria-hidden="true" />}
                label={t("rowRating")}
              >
                {snapshot.average_rating.toFixed(1)} / 5
                {snapshot.review_count
                  ? ` (${snapshot.review_count} ${t("reviewsSuffix")})`
                  : null}
              </Row>
            )}
          {snapshot.pricing_min_cents !== null &&
            snapshot.pricing_min_cents !== undefined && (
              <Row
                icon={<Wallet className="h-4 w-4" aria-hidden="true" />}
                label={t("rowPricing")}
              >
                {formatPriceRange(
                  snapshot.pricing_min_cents,
                  snapshot.pricing_max_cents,
                  snapshot.pricing_currency,
                  snapshot.pricing_type,
                )}
              </Row>
            )}
          {snapshot.region && (
            <Row
              icon={<MapPin className="h-4 w-4" aria-hidden="true" />}
              label={t("rowRegion")}
            >
              {snapshot.region}
            </Row>
          )}
          {snapshot.languages && snapshot.languages.length > 0 && (
            <Row
              icon={<Globe2 className="h-4 w-4" aria-hidden="true" />}
              label={t("rowLanguages")}
            >
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
      <div className="mt-0.5 text-primary">{icon}</div>
      <div className="flex-1">
        <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
        <dd className="text-sm text-foreground">{children}</dd>
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
