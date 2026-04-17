"use client"

import { Calendar, Clock, MessageSquareQuote, Percent } from "lucide-react"

import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

import { useReferral } from "../hooks/use-referrals"
import type { Referral, ReferralActorRole } from "../types"
import { formatRatePct } from "../types"
import { AnonymizedClientCard } from "./anonymized-client-card"
import { AnonymizedProviderCard } from "./anonymized-provider-card"
import { NegotiationTimeline } from "./negotiation-timeline"
import { ReferralActions } from "./referral-actions"
import { ReferralMissionsSection } from "./referral-missions-section"
import { ReferralStatusBadge } from "./referral-status-badge"

interface ReferralDetailViewProps {
  referralId: string
}

// ReferralDetailView is the smart container for the referral detail page.
// It loads the referral, derives the viewer's role from the JWT user vs
// the referral parties, and dispatches to the right rendering branch:
//
//   - referrer  → full view with both anonymised cards + timeline + cancel/terminate
//   - provider  → anonymised client card + timeline + accept/negotiate/reject
//   - client    → anonymised provider card + accept/reject (no rate, no timeline)
export function ReferralDetailView({ referralId }: ReferralDetailViewProps) {
  const { data: referral, isLoading, error } = useReferral(referralId)
  const viewerId = useCurrentUserId()

  if (isLoading) return <DetailSkeleton />
  if (error || !referral) {
    return (
      <div role="alert" className="rounded-2xl border border-rose-200 bg-rose-50 p-6 text-sm text-rose-700">
        Impossible de charger cette mise en relation.
      </div>
    )
  }

  const viewerRole = resolveViewerRole(referral, viewerId)
  if (!viewerRole) {
    return (
      <div role="alert" className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-sm text-amber-700">
        Vous n&rsquo;êtes pas partie à cette mise en relation.
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <Header referral={referral} />
      <div className="grid gap-6 md:grid-cols-2">
        {viewerRole === "client" && (
          <AnonymizedProviderCard snapshot={referral.intro_snapshot.provider} />
        )}
        {viewerRole === "provider" && (
          <AnonymizedClientCard snapshot={referral.intro_snapshot.client} />
        )}
        {viewerRole === "referrer" && (
          <>
            <AnonymizedProviderCard snapshot={referral.intro_snapshot.provider} />
            <AnonymizedClientCard snapshot={referral.intro_snapshot.client} />
          </>
        )}
      </div>

      {referral.intro_message_for_me && (
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <header className="mb-2 flex items-center gap-2">
            <MessageSquareQuote className="h-4 w-4 text-rose-500" aria-hidden="true" />
            <h2 className="text-sm font-semibold text-slate-900">
              Mot de l&rsquo;apporteur
            </h2>
          </header>
          <p className="whitespace-pre-line text-sm text-slate-700">
            &ldquo;{referral.intro_message_for_me}&rdquo;
          </p>
        </section>
      )}

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <header className="mb-3">
          <h2 className="text-sm font-semibold text-slate-900">Vos actions</h2>
        </header>
        <ReferralActions referral={referral} viewerRole={viewerRole} />
      </section>

      {/* Attributed proposals — visible to all three parties once the
          intro is active. The client sees the list without commission
          amounts; apporteur and provider see the full picture. */}
      {referral.status === "active" && (
        <ReferralMissionsSection
          referralId={referral.id}
          viewerIsClient={viewerRole === "client"}
        />
      )}

      {/* Negotiation timeline is hidden from the client until activation
          to avoid leaking historical rate values (Modèle A). */}
      {(viewerRole !== "client" || referral.status === "active") && (
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <header className="mb-3">
            <h2 className="text-sm font-semibold text-slate-900">
              Historique de négociation
            </h2>
          </header>
          <NegotiationTimeline
            referralId={referral.id}
            showRate={viewerRole !== "client"}
          />
        </section>
      )}
    </div>
  )
}

function resolveViewerRole(
  referral: Referral,
  viewerId: string | undefined,
): ReferralActorRole | null {
  if (!viewerId) return null
  if (viewerId === referral.referrer_id) return "referrer"
  if (viewerId === referral.provider_id) return "provider"
  if (viewerId === referral.client_id) return "client"
  return null
}

interface HeaderProps {
  referral: Referral
}

function Header({ referral }: HeaderProps) {
  return (
    <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold text-slate-900">
          Mise en relation
        </h1>
        <ReferralStatusBadge status={referral.status} />
        <span className="text-xs text-slate-500">v{referral.version}</span>
      </div>
      <dl className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Metric icon={<Percent className="h-4 w-4" />} label="Commission">
          {formatRatePct(referral.rate_pct)}
        </Metric>
        <Metric icon={<Calendar className="h-4 w-4" />} label="Durée">
          {referral.duration_months} mois
        </Metric>
        <Metric icon={<Clock className="h-4 w-4" />} label="Dernière action">
          {new Date(referral.last_action_at).toLocaleDateString("fr-FR")}
        </Metric>
      </dl>
      {referral.activated_at && referral.expires_at && (
        <p className="mt-4 rounded-lg bg-emerald-50 px-3 py-2 text-xs text-emerald-700">
          Activée le {new Date(referral.activated_at).toLocaleDateString("fr-FR")}, fenêtre d&rsquo;exclusivité jusqu&rsquo;au{" "}
          {new Date(referral.expires_at).toLocaleDateString("fr-FR")}.
        </p>
      )}
    </header>
  )
}

interface MetricProps {
  icon: React.ReactNode
  label: string
  children: React.ReactNode
}

function Metric({ icon, label, children }: MetricProps) {
  return (
    <div className="flex items-center gap-3 rounded-lg bg-slate-50 px-3 py-2">
      <span className="text-rose-500">{icon}</span>
      <div>
        <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
        <dd className="text-sm font-medium text-slate-900">{children}</dd>
      </div>
    </div>
  )
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="h-32 animate-pulse rounded-2xl border border-slate-200 bg-slate-50" />
      <div className="grid gap-6 md:grid-cols-2">
        <div className="h-64 animate-pulse rounded-2xl border border-slate-200 bg-slate-50" />
        <div className="h-64 animate-pulse rounded-2xl border border-slate-200 bg-slate-50" />
      </div>
    </div>
  )
}
