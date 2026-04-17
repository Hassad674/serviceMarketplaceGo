"use client"

import Link from "next/link"
import {
  Briefcase,
  CheckCircle2,
  CircleDot,
  Hourglass,
  ShieldAlert,
} from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { useReferralAttributions } from "../hooks/use-referrals"
import type { ReferralAttribution } from "../types"

interface ReferralMissionsSectionProps {
  referralId: string
  // viewerIsClient disables the hook's auto-fetch of commission
  // amounts — the backend strips them from the DTO too, but hiding the
  // pending / paid columns on the client side avoids rendering empty
  // cells for the client viewer.
  viewerIsClient: boolean
}

// ReferralMissionsSection lists every proposal attributed to the
// referral during its exclusivity window. This is the only place on
// the apporteur dashboard where they can see a live view of what's
// happening in the contract they set up — which missions the
// provider has won with the client, how far each one has progressed,
// and how much commission has accrued.
//
// Empty state: hidden entirely when there are zero attributions so
// the detail page stays uncluttered until something actually happens.
export function ReferralMissionsSection({
  referralId,
  viewerIsClient,
}: ReferralMissionsSectionProps) {
  const { data, isLoading, isError } = useReferralAttributions(referralId)

  if (isLoading) {
    return (
      <section className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <SectionHeader />
        <div className="mt-3 h-16 animate-shimmer rounded-lg bg-slate-100 dark:bg-slate-700" />
      </section>
    )
  }

  if (isError) {
    return null
  }

  const attributions = data ?? []
  if (attributions.length === 0) {
    return null
  }

  return (
    <section className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <SectionHeader count={attributions.length} />
      <ul className="mt-4 space-y-3">
        {attributions.map((a) => (
          <li key={a.id}>
            <AttributionCard
              attribution={a}
              viewerIsClient={viewerIsClient}
            />
          </li>
        ))}
      </ul>
    </section>
  )
}

function SectionHeader({ count }: { count?: number }) {
  return (
    <header className="flex items-center justify-between gap-2">
      <div className="flex items-center gap-2">
        <div className="grid h-8 w-8 place-items-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          <Briefcase className="h-4 w-4" aria-hidden="true" />
        </div>
        <div>
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
            Missions pendant cette mise en relation
          </h2>
          <p className="text-xs text-slate-500">
            Propositions signées entre les deux parties pendant la fenêtre
            d&apos;exclusivité.
          </p>
        </div>
      </div>
      {count !== undefined && (
        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-700 dark:text-slate-300">
          {count}
        </span>
      )}
    </header>
  )
}

function AttributionCard({
  attribution,
  viewerIsClient,
}: {
  attribution: ReferralAttribution
  viewerIsClient: boolean
}) {
  const status = proposalStatusBadge(attribution.proposal_status)
  const rate = attribution.rate_pct_snapshot
  const paid = attribution.total_commission_cents ?? 0
  const pending = attribution.pending_commission_cents ?? 0

  return (
    <article className="rounded-xl border border-slate-100 p-4 transition hover:border-rose-200 hover:bg-rose-50/30 dark:border-slate-700 dark:hover:border-rose-500/40">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <Link
            href={`/projects/${attribution.proposal_id}`}
            className="text-sm font-semibold text-slate-900 hover:underline dark:text-white"
          >
            {attribution.proposal_title ||
              `Proposition ${attribution.proposal_id.slice(0, 8)}…`}
          </Link>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <span
              className={cn(
                "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium",
                status.cls,
              )}
            >
              <status.icon className="h-3 w-3" aria-hidden="true" />
              {status.label}
            </span>
            <span className="text-[11px] text-slate-500">
              Attribuée le{" "}
              {new Date(attribution.attributed_at).toLocaleDateString("fr-FR")}
            </span>
            {rate !== undefined && (
              <span className="text-[11px] text-slate-500">
                · Taux {rate.toFixed(rate % 1 === 0 ? 0 : 1)} %
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
        <Stat
          label="Jalons payés"
          value={String(attribution.milestones_paid)}
        />
        <Stat
          label="Jalons en cours"
          value={String(attribution.milestones_pending)}
        />
        {!viewerIsClient && (
          <Stat
            label="Commission"
            value={formatEurCents(paid)}
            subtitle={
              pending > 0 ? `+ ${formatEurCents(pending)} en attente` : undefined
            }
            accent
          />
        )}
      </div>
    </article>
  )
}

function Stat({
  label,
  value,
  subtitle,
  accent,
}: {
  label: string
  value: string
  subtitle?: string
  accent?: boolean
}) {
  return (
    <div
      className={cn(
        "rounded-lg border border-slate-100 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-900/40",
        accent &&
          "border-rose-100 bg-rose-50 text-rose-700 dark:border-rose-500/20 dark:bg-rose-500/10",
      )}
    >
      <div className="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {label}
      </div>
      <div
        className={cn(
          "mt-0.5 text-sm font-semibold text-slate-900 dark:text-white",
          accent && "text-rose-700 dark:text-rose-400",
        )}
      >
        {value}
      </div>
      {subtitle && (
        <div className="mt-0.5 text-[10px] text-rose-600/80 dark:text-rose-300/80">
          {subtitle}
        </div>
      )}
    </div>
  )
}

type ProposalBadge = {
  label: string
  cls: string
  icon: typeof CheckCircle2
}

function proposalStatusBadge(status: string | undefined): ProposalBadge {
  switch (status) {
    case "paid":
    case "active":
    case "completion_requested":
      return {
        label: labelFor(status),
        cls: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
        icon: CircleDot,
      }
    case "completed":
      return {
        label: "Terminée",
        cls: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
        icon: CheckCircle2,
      }
    case "pending":
    case "accepted":
      return {
        label: labelFor(status),
        cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
        icon: Hourglass,
      }
    case "disputed":
      return {
        label: "En litige",
        cls: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
        icon: ShieldAlert,
      }
    case "declined":
    case "withdrawn":
      return {
        label: labelFor(status),
        cls: "bg-slate-100 text-slate-600",
        icon: Hourglass,
      }
    default:
      return {
        label: status ?? "—",
        cls: "bg-slate-100 text-slate-600",
        icon: CircleDot,
      }
  }
}

function labelFor(status: string): string {
  switch (status) {
    case "pending":
      return "En attente"
    case "accepted":
      return "Acceptée"
    case "paid":
      return "Financée"
    case "active":
      return "En cours"
    case "completion_requested":
      return "Complétion demandée"
    case "completed":
      return "Terminée"
    case "declined":
      return "Refusée"
    case "withdrawn":
      return "Retirée"
    default:
      return status
  }
}

function formatEurCents(cents: number): string {
  return new Intl.NumberFormat("fr-FR", {
    style: "currency",
    currency: "EUR",
  }).format(cents / 100)
}
