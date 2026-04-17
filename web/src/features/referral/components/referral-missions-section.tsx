"use client"

import { Briefcase } from "lucide-react"

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
  const escrow = attribution.escrow_commission_cents ?? 0
  const clawedBack = attribution.clawed_back_commission_cents ?? 0
  const mDone = attribution.milestones_paid
  // milestones_total is the authoritative count from the proposal
  // (every proposal has >= 1 milestone by domain invariant). Falls
  // back to paid+pending only in case an older API is responding.
  const mTotal =
    attribution.milestones_total > 0
      ? attribution.milestones_total
      : mDone + attribution.milestones_pending
  const progress = mTotal > 0 ? Math.round((mDone / mTotal) * 100) : 0
  const isTerminated = attribution.proposal_status === "completed"
  const title =
    attribution.proposal_title ||
    `Proposition ${attribution.proposal_id.slice(0, 8)}…`
  const attributedOn = new Date(attribution.attributed_at).toLocaleDateString(
    "fr-FR",
    { day: "2-digit", month: "short", year: "numeric" },
  )

  // The apporteur is NOT a party to the underlying project (Modèle A
  // keeps the working conversation strictly between client and
  // provider), so this row is intentionally a plain informational
  // card: no Link wrapper, no chevron, no cursor change. The
  // apporteur sees proposal status + milestone progress for context,
  // nothing more.
  return (
    <div
      className="flex items-start gap-3 rounded-lg px-2 py-3"
      aria-label={`${title} — ${status.label}`}
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
          <span className="truncate text-sm font-semibold text-slate-900 dark:text-white">
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

        {/* Milestone progress line — {paid}/{total} drives the bar */}
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

      {/* Commission (right column) — primary value for apporteur.
          Shows the paid amount first; stacks escrow and clawback
          sub-lines so a mission in progress does not read "0 €" when
          money is held but not yet released. Client viewer sees no
          commission column at all (Modèle A). */}
      {!viewerIsClient && (
        <CommissionColumn
          paid={paid}
          escrow={escrow}
          clawedBack={clawedBack}
          isTerminated={isTerminated}
        />
      )}
    </div>
  )
}

// CommissionColumn renders the right-side money stack for the
// apporteur. The state matrix:
//   paid > 0                              -> primary rose number
//   paid == 0 && escrow > 0               -> primary muted "0 €" + escrow line
//   paid == 0 && escrow == 0 && !terminated -> "0 €" + "En attente du financement"
//   paid == 0 && escrow == 0 && terminated  -> "0 €" + "Aucun jalon libéré"
// Clawback line is always stacked last when > 0.
function CommissionColumn({
  paid,
  escrow,
  clawedBack,
  isTerminated,
}: {
  paid: number
  escrow: number
  clawedBack: number
  isTerminated: boolean
}) {
  const hasPaid = paid > 0
  const hasEscrow = escrow > 0
  const hasClawback = clawedBack > 0

  const primaryCls = hasPaid
    ? "text-rose-600 dark:text-rose-400"
    : "text-slate-400 dark:text-slate-500"

  let caption: string | null = null
  if (!hasPaid && !hasEscrow) {
    caption = isTerminated ? "Aucun jalon libéré" : "En attente du financement"
  }

  return (
    <div className="shrink-0 text-right">
      <div
        className={cn(
          "text-sm font-bold tabular-nums",
          primaryCls,
        )}
      >
        {formatEurCents(paid)}
      </div>
      {hasEscrow && (
        <div className="mt-0.5 text-[10px] font-medium tabular-nums text-amber-600 dark:text-amber-400">
          + {formatEurCents(escrow)} en séquestre
        </div>
      )}
      {hasClawback && (
        <div className="mt-0.5 text-[10px] font-medium tabular-nums text-red-600 dark:text-red-400">
          − {formatEurCents(clawedBack)} reprises
        </div>
      )}
      {caption ? (
        <div className="mt-0.5 text-[10px] text-slate-400 dark:text-slate-500">
          {caption}
        </div>
      ) : (
        <div className="text-[10px] uppercase tracking-wide text-slate-400 dark:text-slate-500">
          Commission
        </div>
      )}
    </div>
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
