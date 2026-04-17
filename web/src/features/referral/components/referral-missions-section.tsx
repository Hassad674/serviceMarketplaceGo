"use client"

import Link from "next/link"
import { Briefcase, ChevronRight } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { useReferralAttributions } from "../hooks/use-referrals"
import type { ReferralAttribution } from "../types"

interface ReferralMissionsSectionProps {
  referralId: string
  // viewerIsClient disables the hook's auto-fetch of commission
  // amounts — the backend strips them from the DTO too, but hiding the
  // commission column on the client side avoids rendering empty cells.
  viewerIsClient: boolean
}

// ReferralMissionsSection lists every proposal attributed to the
// referral during its exclusivity window. This is the only place on
// the apporteur dashboard where they can see a live view of what's
// happening in the contract they set up — which missions the
// provider has won with the client, how far each one has progressed,
// and how much commission has accrued.
//
// Layout is a dense, scannable list (one row ≈ 64px) instead of fat
// cards — with 5+ attributions the old layout required too much
// scrolling. The commission column is the visual anchor on the right;
// milestone progress is inline with a micro progress bar.
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
      <section className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <SectionHeader />
        <ul className="mt-3 divide-y divide-slate-100 dark:divide-slate-700/60">
          {[0, 1, 2].map((i) => (
            <li key={i} className="py-3">
              <div className="flex items-center gap-3">
                <div className="h-2 w-2 rounded-full bg-slate-200 dark:bg-slate-700" />
                <div className="flex-1 space-y-2">
                  <div className="h-3.5 w-2/3 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
                  <div className="h-2.5 w-1/2 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
                </div>
                <div className="h-5 w-16 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
              </div>
            </li>
          ))}
        </ul>
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
    <section className="rounded-2xl border border-slate-100 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <SectionHeader count={attributions.length} />
      <ul className="mt-2 divide-y divide-slate-100 dark:divide-slate-700/60">
        {attributions.map((a) => (
          <li key={a.id}>
            <AttributionRow
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
      <div className="flex min-w-0 items-center gap-2.5">
        <div className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          <Briefcase className="h-4 w-4" aria-hidden="true" />
        </div>
        <div className="min-w-0">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
            Missions pendant cette mise en relation
          </h2>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Propositions signées pendant la fenêtre d&apos;exclusivité.
          </p>
        </div>
      </div>
      {count !== undefined && (
        <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold tabular-nums text-slate-600 dark:bg-slate-700 dark:text-slate-300">
          {count}
        </span>
      )}
    </header>
  )
}

function AttributionRow({
  attribution,
  viewerIsClient,
}: {
  attribution: ReferralAttribution
  viewerIsClient: boolean
}) {
  const status = proposalStatus(attribution.proposal_status)
  const rate = attribution.rate_pct_snapshot
  const paid = attribution.total_commission_cents ?? 0
  const pending = attribution.pending_commission_cents ?? 0
  const mDone = attribution.milestones_paid
  const mPending = attribution.milestones_pending
  const mTotal = mDone + mPending
  const progress = mTotal > 0 ? Math.round((mDone / mTotal) * 100) : 0
  const title =
    attribution.proposal_title ||
    `Proposition ${attribution.proposal_id.slice(0, 8)}…`
  const attributedOn = new Date(attribution.attributed_at).toLocaleDateString(
    "fr-FR",
    { day: "2-digit", month: "short", year: "numeric" },
  )

  return (
    <Link
      href={`/projects/${attribution.proposal_id}`}
      className="group flex items-center gap-3 rounded-lg px-2 py-3 transition-colors hover:bg-slate-50 focus-visible:bg-slate-50 focus-visible:outline-none dark:hover:bg-slate-700/40 dark:focus-visible:bg-slate-700/40"
      aria-label={`${title} — ouvrir la mission`}
    >
      {/* Status dot — tiny, high contrast, left gutter */}
      <span
        className={cn(
          "mt-1.5 h-2 w-2 shrink-0 self-start rounded-full",
          status.dotCls,
        )}
        aria-hidden="true"
      />

      {/* Main content: title + meta lines */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-semibold text-slate-900 group-hover:text-rose-600 dark:text-white dark:group-hover:text-rose-400">
            {title}
          </span>
          <span
            className={cn(
              "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium",
              status.pillCls,
            )}
          >
            {status.label}
          </span>
        </div>

        {/* Milestone progress line */}
        <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500 dark:text-slate-400">
          <span className="tabular-nums">
            {mDone}/{mTotal} jalons
          </span>
          {mTotal > 0 && (
            <span
              className="relative h-1 w-16 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-700"
              role="progressbar"
              aria-valuenow={progress}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`Progression ${progress}%`}
            >
              <span
                className="absolute inset-y-0 left-0 rounded-full bg-rose-500 transition-all dark:bg-rose-400"
                style={{ width: `${progress}%` }}
              />
            </span>
          )}
          {!viewerIsClient && pending > 0 && (
            <>
              <span aria-hidden="true">·</span>
              <span className="text-amber-600 dark:text-amber-400">
                {formatEurCents(pending)} en attente
              </span>
            </>
          )}
        </div>

        {/* Attribution date + rate — tertiary info */}
        <div className="mt-0.5 text-[11px] text-slate-400 dark:text-slate-500">
          Attribuée le {attributedOn}
          {rate !== undefined && (
            <>
              {" · "}
              Taux {rate.toFixed(rate % 1 === 0 ? 0 : 1)} %
            </>
          )}
        </div>
      </div>

      {/* Commission (right column) — primary value for apporteur */}
      {!viewerIsClient && (
        <div className="shrink-0 text-right">
          <div className="text-sm font-bold tabular-nums text-rose-600 dark:text-rose-400">
            {formatEurCents(paid)}
          </div>
          <div className="text-[10px] uppercase tracking-wide text-slate-400 dark:text-slate-500">
            Commission
          </div>
        </div>
      )}

      <ChevronRight
        className="h-4 w-4 shrink-0 text-slate-300 transition-transform group-hover:translate-x-0.5 group-hover:text-rose-500 dark:text-slate-600 dark:group-hover:text-rose-400"
        aria-hidden="true"
      />
    </Link>
  )
}

type StatusMeta = {
  label: string
  dotCls: string
  pillCls: string
}

function proposalStatus(status: string | undefined): StatusMeta {
  switch (status) {
    case "paid":
    case "active":
    case "completion_requested":
      return {
        label: labelFor(status),
        dotCls: "bg-emerald-500 ring-2 ring-emerald-500/20",
        pillCls:
          "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
      }
    case "completed":
      return {
        label: "Terminée",
        dotCls: "bg-blue-500 ring-2 ring-blue-500/20",
        pillCls:
          "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
      }
    case "pending":
    case "accepted":
      return {
        label: labelFor(status),
        dotCls: "bg-amber-500 ring-2 ring-amber-500/20",
        pillCls:
          "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
      }
    case "disputed":
      return {
        label: "En litige",
        dotCls: "bg-red-500 ring-2 ring-red-500/20",
        pillCls: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
      }
    case "declined":
    case "withdrawn":
      return {
        label: labelFor(status),
        dotCls: "bg-slate-400 ring-2 ring-slate-400/20",
        pillCls:
          "bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
      }
    default:
      return {
        label: status ?? "—",
        dotCls: "bg-slate-400 ring-2 ring-slate-400/20",
        pillCls:
          "bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
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
    maximumFractionDigits: 0,
  }).format(cents / 100)
}
