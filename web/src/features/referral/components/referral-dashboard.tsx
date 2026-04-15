"use client"

import Link from "next/link"
import { ArrowRight, Briefcase, CheckCircle2, Clock, Plus, Sparkles } from "lucide-react"

import { useIncomingReferrals, useMyReferrals } from "../hooks/use-referrals"
import type { Referral } from "../types"
import { formatRatePct } from "../types"
import { ReferralStatusBadge } from "./referral-status-badge"

// ReferralDashboard is the apporteur's home for the feature: a top row of
// stat cards summarising state, then three sections (pending, active,
// history) listing the referrals they own. Items where the current user
// is on the receiving side (provider/client) are surfaced in a separate
// "À traiter" section so they are not lost in the noise of the main list.
export function ReferralDashboard() {
  const mine = useMyReferrals()
  const incoming = useIncomingReferrals()

  const referrals = mine.data?.items ?? []
  const incomingItems = incoming.data?.items ?? []

  const stats = computeStats(referrals)
  const pending = referrals.filter((r) => r.status.startsWith("pending_"))
  const active = referrals.filter((r) => r.status === "active")
  const history = referrals.filter(
    (r) =>
      r.status === "rejected" ||
      r.status === "expired" ||
      r.status === "cancelled" ||
      r.status === "terminated",
  )

  return (
    <div className="space-y-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            Apports d&rsquo;affaires
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Vos mises en relation et leurs commissions.
          </p>
        </div>
        <Link
          href="/referrals/new"
          className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-rose-600"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          Nouvelle intro
        </Link>
      </header>

      <StatCards stats={stats} />

      {incomingItems.length > 0 && (
        <Section
          title="À traiter"
          description="Mises en relation où vous devez accepter, négocier ou refuser."
        >
          <ReferralList items={incomingItems} />
        </Section>
      )}

      <Section
        title="En attente de réponse"
        description="Vos intros qui attendent une action d'une autre partie."
        emptyState="Aucune intro en cours."
        loading={mine.isLoading}
      >
        <ReferralList items={pending} />
      </Section>

      <Section
        title="Mises en relation actives"
        description="Vos intros activées dans leur fenêtre d'exclusivité."
        emptyState="Aucune intro active pour le moment."
      >
        <ReferralList items={active} />
      </Section>

      <Section
        title="Historique"
        description="Intros terminées, expirées ou annulées."
        emptyState="L'historique apparaîtra ici."
      >
        <ReferralList items={history} />
      </Section>
    </div>
  )
}

interface Stats {
  pendingCount: number
  activeCount: number
  totalCount: number
}

function computeStats(referrals: Referral[]): Stats {
  return {
    pendingCount: referrals.filter((r) => r.status.startsWith("pending_")).length,
    activeCount: referrals.filter((r) => r.status === "active").length,
    totalCount: referrals.length,
  }
}

interface StatCardsProps {
  stats: Stats
}

function StatCards({ stats }: StatCardsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <StatCard
        icon={<Clock className="h-5 w-5 text-amber-500" aria-hidden="true" />}
        label="En cours"
        value={stats.pendingCount}
        accent="bg-amber-50"
      />
      <StatCard
        icon={<CheckCircle2 className="h-5 w-5 text-emerald-500" aria-hidden="true" />}
        label="Actives"
        value={stats.activeCount}
        accent="bg-emerald-50"
      />
      <StatCard
        icon={<Sparkles className="h-5 w-5 text-rose-500" aria-hidden="true" />}
        label="Total"
        value={stats.totalCount}
        accent="bg-rose-50"
      />
    </div>
  )
}

interface StatCardProps {
  icon: React.ReactNode
  label: string
  value: number | string
  accent: string
}

function StatCard({ icon, label, value, accent }: StatCardProps) {
  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div className={`grid h-10 w-10 place-items-center rounded-full ${accent}`}>
          {icon}
        </div>
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
            {label}
          </p>
          <p className="text-2xl font-semibold tabular-nums text-slate-900">
            {value}
          </p>
        </div>
      </div>
    </article>
  )
}

interface SectionProps {
  title: string
  description?: string
  emptyState?: string
  loading?: boolean
  children: React.ReactNode
}

function Section({ title, description, emptyState, loading, children }: SectionProps) {
  // Detect emptiness from children: if the only child is a list with no
  // items, show the empty state instead.
  return (
    <section>
      <header className="mb-3">
        <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
        {description && (
          <p className="mt-0.5 text-sm text-slate-500">{description}</p>
        )}
      </header>
      {loading ? <SectionSkeleton /> : children || (
        <p className="rounded-2xl border border-dashed border-slate-200 px-6 py-8 text-center text-sm text-slate-500">
          {emptyState}
        </p>
      )}
    </section>
  )
}

function SectionSkeleton() {
  return (
    <div className="space-y-2">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-16 animate-pulse rounded-2xl border border-slate-200 bg-slate-50"
        />
      ))}
    </div>
  )
}

interface ReferralListProps {
  items: Referral[]
}

function ReferralList({ items }: ReferralListProps) {
  if (items.length === 0) {
    return null
  }
  return (
    <ul className="space-y-2">
      {items.map((r) => (
        <li key={r.id}>
          <Link
            href={`/referrals/${r.id}`}
            className="group flex items-center justify-between gap-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:border-rose-200 hover:shadow-md"
          >
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <ReferralStatusBadge status={r.status} />
                <span className="text-xs text-slate-500">
                  v{r.version} · {formatRatePct(r.rate_pct)} · {r.duration_months}{" "}
                  mois
                </span>
              </div>
              <p className="mt-1.5 text-sm text-slate-700">
                <Briefcase className="mr-1.5 inline h-3.5 w-3.5 text-slate-400" aria-hidden="true" />
                Couple <code className="font-mono text-xs">{r.provider_id.slice(0, 8)}</code> →{" "}
                <code className="font-mono text-xs">{r.client_id.slice(0, 8)}</code>
              </p>
              {r.activated_at && r.expires_at && (
                <p className="mt-1 text-xs text-slate-500">
                  Activée le{" "}
                  {new Date(r.activated_at).toLocaleDateString("fr-FR")}, expire le{" "}
                  {new Date(r.expires_at).toLocaleDateString("fr-FR")}
                </p>
              )}
            </div>
            <ArrowRight
              className="h-5 w-5 text-slate-400 transition group-hover:translate-x-0.5 group-hover:text-rose-500"
              aria-hidden="true"
            />
          </Link>
        </li>
      ))}
    </ul>
  )
}
